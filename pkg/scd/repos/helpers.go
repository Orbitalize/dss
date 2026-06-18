package repos

import (
	"context"

	"github.com/golang/geo/s2"
	restapi "github.com/interuss/dss/pkg/api/scdv1"
	dsserr "github.com/interuss/dss/pkg/errors"
	dssmodels "github.com/interuss/dss/pkg/models"
	scdmodels "github.com/interuss/dss/pkg/scd/models"
	"github.com/interuss/stacktrace"
)

type ValidOIRParams struct {
	ID                   dssmodels.ID
	OVN                  scdmodels.OVN
	NewOVN               scdmodels.OVN
	State                scdmodels.OperationalIntentState
	UExtent              *dssmodels.Volume4D
	Cells                s2.CellUnion
	SubscriptionID       dssmodels.ID
	USSBaseURL           string
	ImplicitSubscription struct {
		Requested      bool
		ID             dssmodels.ID
		BaseURL        string
		ForConstraints bool
	}
	Key map[scdmodels.OVN]bool
}

func (vp *ValidOIRParams) ToOIR(manager dssmodels.Manager, attachedSub *scdmodels.Subscription, version scdmodels.VersionNumber, pastOVNs []scdmodels.OVN) *scdmodels.OperationalIntent {
	// For OIR's in the accepted state, we may not have a attachedSub available,
	// in such cases the attachedSub ID on scdmodels.OperationalIntent will be nil
	// and will be replaced with the 'NullV4UUID' when sent over to a client.
	var subID *dssmodels.ID
	if attachedSub != nil {
		// Note: do _not_ use vp.subscriptionID here, as it may be empty
		subID = &attachedSub.ID
	}
	return &scdmodels.OperationalIntent{
		ID:       vp.ID,
		Manager:  manager,
		Version:  version,
		OVN:      vp.NewOVN, // non-empty only if the USS has requested an OVN
		PastOVNs: pastOVNs,

		StartTime:     vp.UExtent.StartTime,
		EndTime:       vp.UExtent.EndTime,
		AltitudeLower: vp.UExtent.SpatialVolume.AltitudeLo,
		AltitudeUpper: vp.UExtent.SpatialVolume.AltitudeHi,
		Cells:         vp.Cells,

		USSBaseURL:     vp.USSBaseURL,
		SubscriptionID: subID,
		State:          vp.State,
	}
}

// SubscriptionIsImplicitAndOnlyAttachedToOIR checks whether the subscription
// should be automatically removed when its owning OIR is deleted or re-subscribed.
// Returns true only if the subscription is implicit and exclusively attached to oirID.
func SubscriptionIsImplicitAndOnlyAttachedToOIR(ctx context.Context, r Repository, oirID dssmodels.ID, subscription *scdmodels.Subscription) (bool, error) {
	if subscription == nil {
		return false, nil
	}
	if !subscription.ImplicitSubscription {
		return false, nil
	}
	dependentOps, err := r.GetDependentOperationalIntents(ctx, subscription.ID)
	if err != nil {
		return false, stacktrace.Propagate(err, "Could not find dependent OperationalIntents")
	}
	if len(dependentOps) == 0 {
		return false, stacktrace.NewError("An implicit Subscription had no dependent OperationalIntents")
	} else if len(dependentOps) == 1 && dependentOps[0] == oirID {
		return true, nil
	}
	return false, nil
}

// GetRelevantSubscriptionsAndIncrementIndices retrieves subscriptions
// intersecting notifyVolume that are interested in operational intents,
// increments their notification indices, and returns them.
func GetRelevantSubscriptionsAndIncrementIndices(ctx context.Context, r Repository, notifyVolume *dssmodels.Volume4D) (Subscriptions, error) {
	allsubs, err := r.SearchSubscriptions(ctx, notifyVolume)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Failed to search for impacted subscriptions.")
	}

	subs := Subscriptions{}
	for _, sub := range allsubs {
		if sub.NotifyForOperationalIntents {
			subs = append(subs, sub)
		}
	}

	if err := subs.IncrementNotificationIndices(ctx, r); err != nil {
		return nil, stacktrace.Propagate(err, "Failed to increment notification indices of relevant subscriptions")
	}

	return subs, nil
}

// MakeSubscribersToNotify converts a list of Subscriptions into the REST
// SubscriberToNotify format, grouping subscription states by USS base URL.
func MakeSubscribersToNotify(subscriptions []*scdmodels.Subscription) []restapi.SubscriberToNotify {
	result := []restapi.SubscriberToNotify{}
	subscriptionsByURL := map[string][]restapi.SubscriptionState{}
	for _, sub := range subscriptions {
		subState := restapi.SubscriptionState{
			SubscriptionId:    restapi.SubscriptionID(sub.ID.String()),
			NotificationIndex: restapi.SubscriptionNotificationIndex(sub.NotificationIndex),
		}
		subscriptionsByURL[sub.USSBaseURL] = append(subscriptionsByURL[sub.USSBaseURL], subState)
	}
	for url, states := range subscriptionsByURL {
		result = append(result, restapi.SubscriberToNotify{
			UssBaseUrl:    restapi.SubscriptionUssBaseURL(url),
			Subscriptions: states,
		})
	}
	return result
}

// ValidateUpsertRequestAgainstPreviousOIR checks manager ownership and OVN
// consistency against the current stored OIR (if any).
func ValidateUpsertRequestAgainstPreviousOIR(requestingManager dssmodels.Manager, providedOVN scdmodels.OVN, previousOIR *scdmodels.OperationalIntent) error {
	if previousOIR != nil {
		if previousOIR.Manager != requestingManager {
			return stacktrace.NewErrorWithCode(dsserr.PermissionDenied,
				"OperationalIntent owned by %s, but %s attempted to modify", previousOIR.Manager, requestingManager)
		}
		if previousOIR.OVN != providedOVN {
			return stacktrace.NewErrorWithCode(dsserr.VersionMismatch,
				"Current version is %s but client specified version %s", previousOIR.OVN, providedOVN)
		}
		return nil
	}
	if providedOVN != "" {
		return stacktrace.NewErrorWithCode(dsserr.NotFound, "OperationalIntent does not exist and therefore is not version %s", providedOVN)
	}
	return nil
}

// ComputeNotificationVolume returns the 4D volume that should be searched for
// subscriptions to notify: the union of the new extent and the previous OIR's
// extent (or just the new extent if there was no previous OIR).
func ComputeNotificationVolume(previousOIR *scdmodels.OperationalIntent, requestedExtent *dssmodels.Volume4D) (*dssmodels.Volume4D, error) {
	if previousOIR == nil {
		return requestedExtent, nil
	}
	oldVolume := &dssmodels.Volume4D{
		StartTime: previousOIR.StartTime,
		EndTime:   previousOIR.EndTime,
		SpatialVolume: &dssmodels.Volume3D{
			AltitudeHi: previousOIR.AltitudeUpper,
			AltitudeLo: previousOIR.AltitudeLower,
			Footprint: dssmodels.GeometryFunc(func() (s2.CellUnion, error) {
				return previousOIR.Cells, nil
			}),
		},
	}
	notifyVolume, err := dssmodels.UnionVolumes4D(requestedExtent, oldVolume)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error constructing 4D volumes union")
	}
	return notifyVolume, nil
}

// ValidateKeyAndProvideConflictResponse checks that the caller's key contains
// OVNs for all currently-stored OIRs and constraints in the area. Returns a
// conflict response (and a MissingOVNs error) if any are absent.
func ValidateKeyAndProvideConflictResponse(
	ctx context.Context,
	r Repository,
	requestingManager dssmodels.Manager,
	uExtent *dssmodels.Volume4D,
	key map[scdmodels.OVN]bool,
	id dssmodels.ID,
	attachedSubscription *scdmodels.Subscription,
) (*restapi.AirspaceConflictResponse, error) {
	var missingOps []*scdmodels.OperationalIntent
	relevantOps, err := r.SearchOperationalIntents(ctx, uExtent)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Unable to SearchOperations")
	}
	for _, relevantOp := range relevantOps {
		_, ok := key[relevantOp.OVN]
		if !ok && relevantOp.RequiresKey() && relevantOp.ID != id {
			missingOps = append(missingOps, relevantOp)
		}
	}

	var missingConstraints []*scdmodels.Constraint
	if attachedSubscription != nil && attachedSubscription.NotifyForConstraints {
		constraints, err := r.SearchConstraints(ctx, uExtent)
		if err != nil {
			return nil, stacktrace.Propagate(err, "Unable to SearchConstraints")
		}
		for _, relevantConstraint := range constraints {
			if _, ok := key[relevantConstraint.OVN]; !ok {
				missingConstraints = append(missingConstraints, relevantConstraint)
			}
		}
	}

	if len(missingOps) == 0 && len(missingConstraints) == 0 {
		return nil, nil
	}

	msg := "Current OVNs not provided for one or more OperationalIntents or Constraints"
	responseConflict := &restapi.AirspaceConflictResponse{Message: &msg}

	if len(missingOps) > 0 {
		responseConflict.MissingOperationalIntents = new([]restapi.OperationalIntentReference)
		for _, missingOp := range missingOps {
			p := missingOp.ToRest()
			if missingOp.Manager != requestingManager {
				noOvnPhrase := restapi.EntityOVN(scdmodels.NoOvnPhrase)
				p.Ovn = &noOvnPhrase
			}
			*responseConflict.MissingOperationalIntents = append(*responseConflict.MissingOperationalIntents, *p)
		}
	}

	if len(missingConstraints) > 0 {
		responseConflict.MissingConstraints = new([]restapi.ConstraintReference)
		for _, missingConstraint := range missingConstraints {
			c := missingConstraint.ToRest()
			if missingConstraint.Manager != requestingManager {
				noOvnPhrase := restapi.EntityOVN(scdmodels.NoOvnPhrase)
				c.Ovn = &noOvnPhrase
			}
			*responseConflict.MissingConstraints = append(*responseConflict.MissingConstraints, *c)
		}
	}

	return responseConflict, stacktrace.NewErrorWithCode(dsserr.MissingOVNs, "Missing OVNs: %v", msg)
}

// EnsureSubscriptionCoversOIR extends an implicit subscription's time/space
// bounds if necessary to cover the OIR's extent, or returns an error for
// explicit subscriptions that don't already cover it.
func EnsureSubscriptionCoversOIR(ctx context.Context, r Repository, sub *scdmodels.Subscription, uExtent *dssmodels.Volume4D, cells s2.CellUnion) (*scdmodels.Subscription, error) {
	updateSub := false
	if sub.StartTime != nil && sub.StartTime.After(*uExtent.StartTime) {
		if sub.ImplicitSubscription {
			sub.StartTime = uExtent.StartTime
			updateSub = true
		} else {
			return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Subscription does not begin until after the OperationalIntent starts")
		}
	}
	if sub.EndTime != nil && sub.EndTime.Before(*uExtent.EndTime) {
		if sub.ImplicitSubscription {
			sub.EndTime = uExtent.EndTime
			updateSub = true
		} else {
			return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Subscription ends before the OperationalIntent ends")
		}
	}
	if !sub.Cells.Contains(cells) {
		if sub.ImplicitSubscription {
			sub.Cells = s2.CellUnionFromUnion(sub.Cells, cells)
			updateSub = true
		} else {
			return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Subscription does not cover entire spatial area of the OperationalIntent")
		}
	}
	if updateSub {
		upsertedSub, err := r.UpsertSubscription(ctx, sub)
		if err != nil {
			return nil, stacktrace.Propagate(err, "Failed to update existing Subscription")
		}
		return upsertedSub, nil
	}
	return sub, nil
}

// CreateAndStoreNewImplicitSubscription creates a new implicit subscription with the given pre-generated ID.
func CreateAndStoreNewImplicitSubscription(ctx context.Context, r Repository, manager dssmodels.Manager, validParams *ValidOIRParams) (*scdmodels.Subscription, error) {
	subToUpsert := scdmodels.Subscription{
		ID:                          validParams.ImplicitSubscription.ID,
		Manager:                     manager,
		StartTime:                   validParams.UExtent.StartTime,
		EndTime:                     validParams.UExtent.EndTime,
		AltitudeLo:                  validParams.UExtent.SpatialVolume.AltitudeLo,
		AltitudeHi:                  validParams.UExtent.SpatialVolume.AltitudeHi,
		Cells:                       validParams.Cells,
		USSBaseURL:                  validParams.ImplicitSubscription.BaseURL,
		NotifyForOperationalIntents: true,
		NotifyForConstraints:        validParams.ImplicitSubscription.ForConstraints,
		ImplicitSubscription:        true,
	}
	return r.UpsertSubscription(ctx, &subToUpsert)
}
