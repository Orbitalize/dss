package raftstore

import (
	"context"
	"encoding/json"
	"time"

	"github.com/golang/geo/s2"
	restapi "github.com/interuss/dss/pkg/api/scdv1"
	dsserr "github.com/interuss/dss/pkg/errors"
	dssmodels "github.com/interuss/dss/pkg/models"
	"github.com/interuss/dss/pkg/raftstore/consensus"
	scdmodels "github.com/interuss/dss/pkg/scd/models"
	"github.com/interuss/dss/pkg/scd/repos"
	"github.com/interuss/stacktrace"
)

func (r *repo) GetOperationalIntent(ctx context.Context, id dssmodels.ID) (*scdmodels.OperationalIntent, error) {
	result, err := r.propose(ctx, getOperationalIntent, id)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to propose getOperationalIntent")
	}

	intent, ok := result.(*scdmodels.OperationalIntent)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return intent, nil
}

func (r *repo) DeleteOperationalIntent(ctx context.Context, id dssmodels.ID) error {
	_, err := r.propose(ctx, deleteOperationalIntent, id)
	return err
}

func (r *repo) UpsertOperationalIntent(ctx context.Context, operation *scdmodels.OperationalIntent) (*scdmodels.OperationalIntent, error) {
	result, err := r.propose(ctx, upsertOperationalIntent, operation)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to propose upsertOperationalIntent")
	}

	intent, ok := result.(*scdmodels.OperationalIntent)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return intent, nil
}

func (r *repo) SearchOperationalIntents(ctx context.Context, v4d *dssmodels.Volume4D) ([]*scdmodels.OperationalIntent, error) {
	result, err := r.propose(ctx, searchOperationalIntents, v4d)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to propose searchOperationalIntents")
	}

	intents, ok := result.([]*scdmodels.OperationalIntent)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return intents, nil
}

func (r *repo) GetDependentOperationalIntents(ctx context.Context, subscriptionID dssmodels.ID) ([]dssmodels.ID, error) {
	result, err := r.propose(ctx, getDependentOperationalIntents, subscriptionID)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to propose getDependentOperationalIntents")
	}

	idList, ok := result.([]dssmodels.ID)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return idList, nil
}

func (r *repo) ListExpiredOperationalIntents(ctx context.Context, threshold time.Time) ([]*scdmodels.OperationalIntent, error) {
	result, err := r.propose(ctx, listExpiredOperationalIntents, threshold)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to propose listExpiredOperationalIntents")
	}

	intents, ok := result.([]*scdmodels.OperationalIntent)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return intents, nil
}

func (r *repo) CountOperationalIntents(ctx context.Context) (int64, error) {
	result, err := r.propose(ctx, countOperationalIntents, nil)
	if err != nil {
		return 0, stacktrace.Propagate(err, "failed to propose countOperationalIntents")
	}

	count, ok := result.(int64)
	if !ok {
		return 0, stacktrace.NewError("invalid result type: %T", result)
	}

	return count, nil
}

func (r *repo) deleteOperationalIntentTransactionApplier(ctx context.Context, proposal consensus.Proposal) (*restapi.ChangeOperationalIntentReferenceResponse, error) {
	var req *restapi.DeleteOperationalIntentReferenceRequest
	if err := json.Unmarshal(proposal.Value, &req); err != nil {
		return nil, stacktrace.Propagate(err, "failed to unmarshal delete operational intent request")
	}

	id, err := dssmodels.IDFromString(string(req.Entityid))
	if err != nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Invalid ID format: `%s`", req.Entityid)
	}

	ovn := scdmodels.OVN(req.Ovn)
	if ovn == "" {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Missing OVN for operational intent to modify")
	}

	old, err := r.memRepo.GetOperationalIntent(ctx, id)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Unable to get OperationIntent from repo")
	}
	if old == nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.NotFound, "OperationalIntent %s not found", id)
	}

	if old.Manager != dssmodels.Manager(*req.Auth.ClientID) {
		return nil, stacktrace.NewErrorWithCode(dsserr.PermissionDenied,
			"OperationalIntent owned by %s, but %s attempted to delete", old.Manager, *req.Auth.ClientID)
	}

	if old.OVN != ovn {
		return nil, stacktrace.NewErrorWithCode(dsserr.VersionMismatch,
			"Current version is %s but client specified version %s", old.OVN, ovn)
	}

	// Get the Subscription supporting the OperationalIntent, if one is defined
	var previousSubscription *scdmodels.Subscription
	if old.SubscriptionID != nil {
		previousSubscription, err = r.memRepo.GetSubscription(ctx, *old.SubscriptionID)
		if err != nil {
			return nil, stacktrace.Propagate(err, "Unable to get OperationalIntent's Subscription from repo")
		}
		if previousSubscription == nil {
			return nil, stacktrace.NewError("OperationalIntent's Subscription missing from repo")
		}
	}

	removeImplicitSubscription, err := repos.SubscriptionIsImplicitAndOnlyAttachedToOIR(ctx, r.memRepo, id, previousSubscription)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Could not determine if Subscription can be removed")
	}

	// Gather the subscriptions that need to be notified
	notifyVolume := &dssmodels.Volume4D{
		StartTime: old.StartTime,
		EndTime:   old.EndTime,
		SpatialVolume: &dssmodels.Volume3D{
			AltitudeHi: old.AltitudeUpper,
			AltitudeLo: old.AltitudeLower,
			Footprint: dssmodels.GeometryFunc(func() (s2.CellUnion, error) {
				return old.Cells, nil
			}),
		}}

	cp := r.memStore.Checkpoint()
	subsToNotify, err := repos.GetRelevantSubscriptionsAndIncrementIndices(ctx, r.memRepo, notifyVolume)
	if err != nil {
		restoreErr := r.memStore.Restore(cp)
		if restoreErr != nil {
			return nil, stacktrace.Propagate(restoreErr, "Failed to restore store")
		}

		return nil, stacktrace.Propagate(err, "could not obtain relevant subscriptions")
	}

	if err := r.memRepo.DeleteOperationalIntent(ctx, id); err != nil {
		restoreErr := r.memStore.Restore(cp)
		if restoreErr != nil {
			return nil, stacktrace.Propagate(restoreErr, "Failed to restore store")
		}

		return nil, stacktrace.Propagate(err, "Unable to delete OperationalIntent from repo")
	}

	// removeImplicitSubscription is only true if the OIR had a subscription defined
	if removeImplicitSubscription {
		// Automatically remove a now-unused implicit Subscription
		err = r.memRepo.DeleteSubscription(ctx, previousSubscription.ID)
		if err != nil {
			restoreErr := r.memStore.Restore(cp)
			if restoreErr != nil {
				return nil, stacktrace.Propagate(restoreErr, "Failed to restore store")
			}

			return nil, stacktrace.Propagate(err, "Unable to delete associated implicit Subscription")
		}
	}

	return &restapi.ChangeOperationalIntentReferenceResponse{
		OperationalIntentReference: *old.ToRest(),
		Subscribers:                repos.MakeSubscribersToNotify(subsToNotify),
	}, nil
}

func (r *repo) getOperationalIntentTransactionApplier(ctx context.Context, proposal consensus.Proposal) (*restapi.GetOperationalIntentReferenceResponse, error) {
	var req *restapi.GetOperationalIntentReferenceRequest
	if err := json.Unmarshal(proposal.Value, &req); err != nil {
		return nil, stacktrace.Propagate(err, "failed to unmarshal get operational intent request")
	}

	id, err := dssmodels.IDFromString(string(req.Entityid))
	if err != nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Invalid ID format: `%s`", req.Entityid)
	}

	op, err := r.memRepo.GetOperationalIntent(ctx, id)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Unable to get OperationalIntent from repo")
	}
	if op == nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.NotFound, "OperationalIntent %s not found", id)
	}

	if op.Manager != dssmodels.Manager(*req.Auth.ClientID) {
		op.OVN = scdmodels.NoOvnPhrase
	}

	return &restapi.GetOperationalIntentReferenceResponse{
		OperationalIntentReference: *op.ToRest(),
	}, nil
}

func (r *repo) queryOperationalIntentTransactionApplier(ctx context.Context, proposal consensus.Proposal) (*restapi.QueryOperationalIntentReferenceResponse, error) {
	var req *restapi.QueryOperationalIntentReferencesRequest
	if err := json.Unmarshal(proposal.Value, &req); err != nil {
		return nil, stacktrace.Propagate(err, "failed to unmarshal query operational intent request")
	}

	vol4, err := dssmodels.Volume4DFromSCDRest(req.Body.AreaOfInterest)
	if err != nil {
		return nil, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Error parsing geometry")
	}

	ops, err := r.memRepo.SearchOperationalIntents(ctx, vol4)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Unable to query for OperationalIntents in repo")
	}

	response := &restapi.QueryOperationalIntentReferenceResponse{
		OperationalIntentReferences: make([]restapi.OperationalIntentReference, 0, len(ops)),
	}
	for _, op := range ops {
		p := op.ToRest()
		if op.Manager != dssmodels.Manager(*req.Auth.ClientID) {
			noOvnPhrase := restapi.EntityOVN(scdmodels.NoOvnPhrase)
			p.Ovn = &noOvnPhrase
		}
		response.OperationalIntentReferences = append(response.OperationalIntentReferences, *p)
	}

	return response, nil
}

// ImplicitSubscriptionParams carries the data needed to create an implicit subscription
// inside the raft applier. NewSubID must be pre-generated by the proposer to keep the
// applier deterministic.
type ImplicitSubscriptionParams struct {
	Requested      bool         `json:"requested"`
	NewSubID       dssmodels.ID `json:"new_sub_id"`
	BaseURL        string       `json:"base_url"`
	ForConstraints bool         `json:"for_constraints"`
}

// UpsertOperationalIntentTransactionPayload is the serialized form of all pre-validated
// parameters passed through Raft for an OIR upsert.
type UpsertOperationalIntentTransactionPayload struct {
	Manager              dssmodels.Manager                `json:"manager"`
	ID                   dssmodels.ID                     `json:"id"`
	Ovn                  scdmodels.OVN                    `json:"ovn"`
	NewOvn               scdmodels.OVN                    `json:"new_ovn,omitempty"`
	State                scdmodels.OperationalIntentState `json:"state"`
	USSBaseURL           string                           `json:"uss_base_url"`
	SubscriptionID       dssmodels.ID                     `json:"subscription_id,omitempty"`
	ImplicitSubscription ImplicitSubscriptionParams       `json:"implicit_subscription"`
	StartTime            *time.Time                       `json:"start_time,omitempty"`
	EndTime              *time.Time                       `json:"end_time,omitempty"`
	AltitudeLo           *float32                         `json:"altitude_lo,omitempty"`
	AltitudeHi           *float32                         `json:"altitude_hi,omitempty"`
	Cells                s2.CellUnion                     `json:"cells"`
	Key                  []scdmodels.OVN                  `json:"key"`
}

// UpsertOperationalIntentTransactionResult carries the response produced by the applier
// back to the handler. Either ResponseOK or ResponseConflict will be set, never both.
type UpsertOperationalIntentTransactionResult struct {
	ResponseOK       *restapi.ChangeOperationalIntentReferenceResponse `json:"response_ok,omitempty"`
	ResponseConflict *restapi.AirspaceConflictResponse                 `json:"response_conflict,omitempty"`
}

func (r *repo) upsertOperationalIntentTransactionApplier(ctx context.Context, proposal consensus.Proposal) (*UpsertOperationalIntentTransactionResult, error) {
	var payload *UpsertOperationalIntentTransactionPayload
	if err := json.Unmarshal(proposal.Value, &payload); err != nil {
		return nil, stacktrace.Propagate(err, "failed to unmarshal upsert operational intent request")
	}

	upsertResult := &UpsertOperationalIntentTransactionResult{}

	// Reconstruct the 4D volume from individual serializable fields.
	uExtent := &dssmodels.Volume4D{
		StartTime: payload.StartTime,
		EndTime:   payload.EndTime,
		SpatialVolume: &dssmodels.Volume3D{
			AltitudeHi: payload.AltitudeHi,
			AltitudeLo: payload.AltitudeLo,
			Footprint: dssmodels.GeometryFunc(func() (s2.CellUnion, error) {
				return payload.Cells, nil
			}),
		},
	}

	// Reconstruct key set.
	key := make(map[scdmodels.OVN]bool, len(payload.Key))
	for _, ovn := range payload.Key {
		key[ovn] = true
	}

	// --- Read phase (no writes yet) ---

	old, err := r.memRepo.GetOperationalIntent(ctx, payload.ID)
	if err != nil {
		return upsertResult, stacktrace.Propagate(err, "Could not get OperationalIntent from repo")
	}

	if err := repos.ValidateUpsertRequestAgainstPreviousOIR(payload.Manager, payload.Ovn, old); err != nil {
		return upsertResult, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), "Request validation failed")
	}

	var (
		version     = scdmodels.VersionNumber(1)
		pastOVNs    = make([]scdmodels.OVN, 0)
		previousSub *scdmodels.Subscription
	)
	if old != nil {
		version = old.Version + 1
		pastOVNs = append(old.PastOVNs, payload.Ovn)

		if old.SubscriptionID != nil {
			previousSub, err = r.memRepo.GetSubscription(ctx, *old.SubscriptionID)
			if err != nil {
				return upsertResult, stacktrace.Propagate(err, "Unable to get OperationalIntent's Subscription from repo")
			}
		}
	}

	previousSubIsBeingReplaced := previousSub != nil && payload.SubscriptionID != previousSub.ID
	removePreviousImplicitSubscription := false
	if previousSubIsBeingReplaced {
		removePreviousImplicitSubscription, err = repos.SubscriptionIsImplicitAndOnlyAttachedToOIR(ctx, r.memRepo, payload.ID, previousSub)
		if err != nil {
			return upsertResult, stacktrace.Propagate(err, "Could not determine if previous Subscription can be removed")
		}
	}

	// Fetch the explicit subscription if needed (read-only).
	var explicitSub *scdmodels.Subscription
	if !payload.SubscriptionID.Empty() && (previousSub == nil || previousSubIsBeingReplaced) {
		explicitSub, err = r.memRepo.GetSubscription(ctx, payload.SubscriptionID)
		if err != nil {
			return upsertResult, stacktrace.Propagate(err, "Unable to get requested Subscription from store")
		}
		if explicitSub == nil {
			return upsertResult, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Specified Subscription %s does not exist", payload.SubscriptionID)
		}
		if explicitSub.Manager != payload.Manager {
			return upsertResult, stacktrace.Propagate(
				stacktrace.NewErrorWithCode(dsserr.PermissionDenied, "Specified Subscription is owned by different client"),
				"Subscription %s owned by %s, but %s attempted to use it for an OperationalIntent",
				payload.SubscriptionID, explicitSub.Manager, payload.Manager,
			)
		}
	} else if !payload.SubscriptionID.Empty() {
		explicitSub = previousSub
	}

	// --- Write phase ---

	cp := r.memStore.Checkpoint()

	attachedSub := previousSub
	if payload.SubscriptionID.Empty() {
		if payload.ImplicitSubscription.Requested {
			subToUpsert := &scdmodels.Subscription{
				ID:                          payload.ImplicitSubscription.NewSubID,
				Manager:                     payload.Manager,
				StartTime:                   uExtent.StartTime,
				EndTime:                     uExtent.EndTime,
				AltitudeLo:                  uExtent.SpatialVolume.AltitudeLo,
				AltitudeHi:                  uExtent.SpatialVolume.AltitudeHi,
				Cells:                       payload.Cells,
				USSBaseURL:                  payload.ImplicitSubscription.BaseURL,
				NotifyForOperationalIntents: true,
				NotifyForConstraints:        payload.ImplicitSubscription.ForConstraints,
				ImplicitSubscription:        true,
			}
			attachedSub, err = r.memRepo.UpsertSubscription(ctx, subToUpsert)
			if err != nil {
				_ = r.memStore.Restore(cp)
				return upsertResult, stacktrace.Propagate(err, "Failed to create implicit subscription")
			}
		} else {
			attachedSub = nil
		}
	} else {
		attachedSub = explicitSub
		// Extend implicit subscription bounds if needed (may write).
		attachedSub, err = repos.EnsureSubscriptionCoversOIR(ctx, r.memRepo, attachedSub, uExtent, payload.Cells)
		if err != nil {
			_ = r.memStore.Restore(cp)
			return upsertResult, stacktrace.Propagate(err, "Failed to ensure subscription covers OIR")
		}
	}

	if payload.State.RequiresKey() {
		responseConflict, err := repos.ValidateKeyAndProvideConflictResponse(ctx, r.memRepo, payload.Manager, uExtent, key, payload.ID, attachedSub)
		if err != nil {
			_ = r.memStore.Restore(cp)
			upsertResult.ResponseConflict = responseConflict
			return upsertResult, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), "Failed to validate key")
		}
	}

	var subID *dssmodels.ID
	if attachedSub != nil {
		subID = &attachedSub.ID
	}
	op := &scdmodels.OperationalIntent{
		ID:             payload.ID,
		Manager:        payload.Manager,
		Version:        version,
		OVN:            payload.NewOvn,
		PastOVNs:       pastOVNs,
		StartTime:      uExtent.StartTime,
		EndTime:        uExtent.EndTime,
		AltitudeLower:  uExtent.SpatialVolume.AltitudeLo,
		AltitudeUpper:  uExtent.SpatialVolume.AltitudeHi,
		Cells:          payload.Cells,
		USSBaseURL:     payload.USSBaseURL,
		SubscriptionID: subID,
		State:          payload.State,
	}

	op, err = r.memRepo.UpsertOperationalIntent(ctx, op)
	if err != nil {
		_ = r.memStore.Restore(cp)
		return upsertResult, stacktrace.Propagate(err, "Failed to upsert OperationalIntent in repo")
	}

	if removePreviousImplicitSubscription {
		if err = r.memRepo.DeleteSubscription(ctx, previousSub.ID); err != nil {
			_ = r.memStore.Restore(cp)
			return upsertResult, stacktrace.Propagate(err, "Unable to delete previous implicit Subscription")
		}
	}

	notifyVolume, err := repos.ComputeNotificationVolume(old, uExtent)
	if err != nil {
		_ = r.memStore.Restore(cp)
		return upsertResult, stacktrace.Propagate(err, "Failed to compute notification volume")
	}

	subsToNotify, err := repos.GetRelevantSubscriptionsAndIncrementIndices(ctx, r.memRepo, notifyVolume)
	if err != nil {
		_ = r.memStore.Restore(cp)
		return upsertResult, stacktrace.Propagate(err, "Failed to notify relevant Subscriptions")
	}

	upsertResult.ResponseOK = &restapi.ChangeOperationalIntentReferenceResponse{
		OperationalIntentReference: *op.ToRest(),
		Subscribers:                repos.MakeSubscribersToNotify(subsToNotify),
	}

	return upsertResult, nil
}
