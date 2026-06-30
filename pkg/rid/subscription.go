package rid

import (
	"context"

	dsserr "github.com/interuss/dss/pkg/errors"
	dssmodels "github.com/interuss/dss/pkg/models"
	ridmodels "github.com/interuss/dss/pkg/rid/models"
	"github.com/interuss/dss/pkg/rid/repos"
	"github.com/interuss/dss/pkg/timestamp"
	"github.com/interuss/stacktrace"
)

// Defined in requirement DSS0030.
const maxSubscriptionsPerArea = 10

// InsertSubscription, UpdateSubscription and DeleteSubscription hold the actual Subscription
// business logic, shared by the ridv1 and ridv2 Action implementations since both REST API
// versions perform the same underlying operation on the same ridmodels.Subscription.

func InsertSubscription(ctx context.Context, r repos.Repository, sub *ridmodels.Subscription) (*ridmodels.Subscription, error) {
	now, err := timestamp.RequestTimestampFromContext(ctx)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Unable to get request timestamp")
	}
	// Validate and perhaps correct StartTime and EndTime.
	if err := sub.AdjustTimeRange(now, nil); err != nil {
		return nil, stacktrace.Propagate(err, "Unable to adjust time range")
	}

	// ensure it doesn't exist yet
	old, err := r.GetSubscription(ctx, sub.ID)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error getting Subscription from repo")
	}
	if old != nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.AlreadyExists, "Subscription %s already exists", sub.ID)
	}

	// Check the user hasn't created too many subscriptions in this area.
	count, err := r.MaxSubscriptionCountInCellsByOwner(ctx, sub.Cells, sub.Owner)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Failed to fetch subscription count, rejecting request")
	}
	if count >= maxSubscriptionsPerArea {
		return nil, stacktrace.Propagate(
			stacktrace.NewErrorWithCode(dsserr.Exhausted, "Too many existing subscriptions in this area already"),
			"%s had %d subscriptions in the area", sub.Owner, count)
	}

	inserted, err := r.InsertSubscription(ctx, sub)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error inserting Subscription into repo")
	}

	return inserted, nil
}

func UpdateSubscription(ctx context.Context, r repos.Repository, sub *ridmodels.Subscription) (*ridmodels.Subscription, error) {
	old, err := r.GetSubscription(ctx, sub.ID)
	switch {
	case err != nil:
		return nil, stacktrace.Propagate(err, "Error getting Subscription from repo")
	case old == nil:
		// The user wants to update an existing subscription, but one wasn't found.
		return nil, stacktrace.NewErrorWithCode(dsserr.NotFound, "Subscription %s not found", sub.ID.String())
	case !sub.Version.Matches(old.Version):
		// The user wants to update a subscription but the version doesn't match.
		return nil, stacktrace.Propagate(
			stacktrace.NewErrorWithCode(dsserr.VersionMismatch, "Subscription version %s is not current", sub.Version),
			"Subscription currently at version %s but client specified %s", old.Version, sub.Version)
	case old.Owner != sub.Owner:
		return nil, stacktrace.Propagate(
			stacktrace.NewErrorWithCode(dsserr.PermissionDenied, "Subscription is owned by different client"),
			"Subscription owned by %s, but %s attempted to update", old.Owner, sub.Owner)
	}

	now, err := timestamp.RequestTimestampFromContext(ctx)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Unable to get request timestamp")
	}
	// Validate and perhaps correct StartTime and EndTime.
	if err := sub.AdjustTimeRange(now, old); err != nil {
		return nil, stacktrace.Propagate(err, "Error adjusting time range")
	}

	// Check the user hasn't created too many subscriptions in this area.
	count, err := r.MaxSubscriptionCountInCellsByOwner(ctx, sub.Cells, sub.Owner)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Failed to fetch subscription count, rejecting request")
	}
	if count >= maxSubscriptionsPerArea {
		return nil, stacktrace.Propagate(
			stacktrace.NewErrorWithCode(dsserr.Exhausted, "Too many existing subscriptions in this area already"),
			"%s had %d subscriptions in the area", sub.Owner, count)
	}

	updated, err := r.UpdateSubscription(ctx, sub)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error updating Subscription in repo")
	}

	return updated, nil
}

func DeleteSubscription(ctx context.Context, r repos.Repository, id dssmodels.ID, owner dssmodels.Owner, version *dssmodels.Version) (*ridmodels.Subscription, error) {
	old, err := r.GetSubscription(ctx, id)
	switch {
	case err != nil:
		return nil, stacktrace.Propagate(err, "Error getting Subscription from repo")
	case old == nil:
		return nil, stacktrace.NewErrorWithCode(dsserr.NotFound, "Subscription %s not found", id.String())
	case !version.Matches(old.Version):
		return nil, stacktrace.Propagate(
			stacktrace.NewErrorWithCode(dsserr.VersionMismatch, "Subscription version %s is not current", version),
			"Subscription currently at version %s but client specified %s", old.Version, version)
	case old.Owner != owner:
		return nil, stacktrace.Propagate(
			stacktrace.NewErrorWithCode(dsserr.PermissionDenied, "Subscription is owned by different client"),
			"Subscription owned by %s, but %s attempted to delete", old.Owner, owner)
	}

	ret, err := r.DeleteSubscription(ctx, old)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error deleting Subscription from repo")
	}

	return ret, nil
}
