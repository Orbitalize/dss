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

	if req.Auth.ClientID == nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.PermissionDenied, "Missing manager")
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

	removeImplicitSubscription, err := subscriptionIsImplicitAndOnlyAttachedToOIR(ctx, r, id, previousSubscription)
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

	subsToNotify, err := getRelevantSubscriptionsAndIncrementIndices(ctx, r, notifyVolume)
	if err != nil {
		return nil, stacktrace.Propagate(err, "could not obtain relevant subscriptions")
	}

	// Delete OperationalIntent from repo
	if err := r.memRepo.DeleteOperationalIntent(ctx, id); err != nil {
		return nil, stacktrace.Propagate(err, "Unable to delete OperationalIntent from repo")
	}

	// removeImplicitSubscription is only true if the OIR had a subscription defined
	if removeImplicitSubscription {
		// Automatically remove a now-unused implicit Subscription
		err = r.memRepo.DeleteSubscription(ctx, previousSubscription.ID)
		if err != nil {
			return nil, stacktrace.Propagate(err, "Unable to delete associated implicit Subscription")
		}
	}

	return &restapi.ChangeOperationalIntentReferenceResponse{
		OperationalIntentReference: *old.ToRest(),
		Subscribers:                makeSubscribersToNotify(subsToNotify),
	}, nil
}
