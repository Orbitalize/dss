package raftstore

import (
	"context"
	"encoding/json"
	"time"

	"github.com/golang/geo/s2"
	dsserr "github.com/interuss/dss/pkg/errors"
	dssmodels "github.com/interuss/dss/pkg/models"
	"github.com/interuss/dss/pkg/raftstore/consensus"
	ridmodels "github.com/interuss/dss/pkg/rid/models"
	"github.com/interuss/stacktrace"
)

type searchSubscriptionsByOwnerPayload struct {
	Cells s2.CellUnion    `json:"cells"`
	Owner dssmodels.Owner `json:"owner"`
}

type maxSubscriptionCountInCellsByOwnerPayload struct {
	Cells s2.CellUnion    `json:"cells"`
	Owner dssmodels.Owner `json:"owner"`
}

type listExpiredSubscriptionsPayload struct {
	Writer    string    `json:"writer"`
	Threshold time.Time `json:"threshold"`
}

func (r *repo) GetSubscription(ctx context.Context, id dssmodels.ID) (*ridmodels.Subscription, error) {
	result, err := r.propose(ctx, getSubscription, id)
	if err != nil {
		return nil, err
	}

	sub, ok := result.(*ridmodels.Subscription)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return sub, nil
}

func (r *repo) DeleteSubscription(ctx context.Context, sub *ridmodels.Subscription) (*ridmodels.Subscription, error) {
	result, err := r.propose(ctx, deleteSubscription, sub)
	if err != nil {
		return nil, err
	}

	out, ok := result.(*ridmodels.Subscription)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return out, nil
}

func (r *repo) InsertSubscription(ctx context.Context, sub *ridmodels.Subscription) (*ridmodels.Subscription, error) {
	result, err := r.propose(ctx, insertSubscription, sub)
	if err != nil {
		return nil, err
	}

	out, ok := result.(*ridmodels.Subscription)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return out, nil
}

func (r *repo) UpdateSubscription(ctx context.Context, sub *ridmodels.Subscription) (*ridmodels.Subscription, error) {
	result, err := r.propose(ctx, updateSubscription, sub)
	if err != nil {
		return nil, err
	}

	out, ok := result.(*ridmodels.Subscription)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return out, nil
}

func (r *repo) SearchSubscriptions(ctx context.Context, cells s2.CellUnion) ([]*ridmodels.Subscription, error) {
	result, err := r.propose(ctx, searchSubscriptions, cells)
	if err != nil {
		return nil, err
	}

	out, ok := result.([]*ridmodels.Subscription)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return out, nil
}

func (r *repo) SearchSubscriptionsByOwner(ctx context.Context, cells s2.CellUnion, owner dssmodels.Owner) ([]*ridmodels.Subscription, error) {
	result, err := r.propose(ctx, searchSubscriptionsByOwner, &searchSubscriptionsByOwnerPayload{Cells: cells, Owner: owner})
	if err != nil {
		return nil, err
	}

	out, ok := result.([]*ridmodels.Subscription)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return out, nil
}

func (r *repo) UpdateNotificationIdxsInCells(ctx context.Context, cells s2.CellUnion) ([]*ridmodels.Subscription, error) {
	result, err := r.propose(ctx, updateNotificationIdxsInCells, cells)
	if err != nil {
		return nil, err
	}

	out, ok := result.([]*ridmodels.Subscription)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return out, nil
}

func (r *repo) MaxSubscriptionCountInCellsByOwner(ctx context.Context, cells s2.CellUnion, owner dssmodels.Owner) (int, error) {
	result, err := r.propose(ctx, maxSubscriptionCountInCellsByOwner, &maxSubscriptionCountInCellsByOwnerPayload{Cells: cells, Owner: owner})
	if err != nil {
		return 0, err
	}

	count, ok := result.(int)
	if !ok {
		return 0, stacktrace.NewError("invalid result type: %T", result)
	}

	return count, nil
}

func (r *repo) ListExpiredSubscriptions(ctx context.Context, writer string, threshold time.Time) ([]*ridmodels.Subscription, error) {
	result, err := r.propose(ctx, listExpiredSubscriptions, &listExpiredSubscriptionsPayload{Writer: writer, Threshold: threshold})
	if err != nil {
		return nil, err
	}

	out, ok := result.([]*ridmodels.Subscription)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return out, nil
}

func (r *repo) CountSubscriptions(ctx context.Context) (int64, error) {
	result, err := r.propose(ctx, countSubscriptions, nil)
	if err != nil {
		return 0, err
	}

	count, ok := result.(int64)
	if !ok {
		return 0, stacktrace.NewError("invalid result type: %T", result)
	}

	return count, nil
}

func (r *repo) insertSubscriptionTransactionApplier(ctx context.Context, proposal consensus.Proposal) (*ridmodels.Subscription, error) {
	var payload *ridmodels.Subscription
	if err := json.Unmarshal(proposal.Value, &payload); err != nil {
		return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", insertSubscription)
	}

	old, err := r.memRepo.GetSubscription(ctx, payload.ID)
	if err != nil {
		return nil, stacktrace.Propagate(err, "error getting Subscription from repo")
	}
	if old != nil {
		return nil, stacktrace.NewError("subscription %s already exists", payload.ID)
	}

	count, err := r.memRepo.MaxSubscriptionCountInCellsByOwner(ctx, payload.Cells, payload.Owner)
	if err != nil {
		return nil, stacktrace.Propagate(err,
			"Failed to fetch subscription count, rejecting request")
	}
	if count >= ridmodels.MaxSubscriptionsPerArea {
		return nil, stacktrace.Propagate(
			stacktrace.NewError("too many existing subscriptions in this area already"),
			"%s had %d subscriptions in the area", payload.Owner, count)
	}

	checkpoint := r.memStore.Checkpoint()
	ret, err := r.memRepo.InsertSubscription(ctx, payload)
	if err != nil {
		restoreErr := r.memStore.Restore(checkpoint)
		if restoreErr != nil {
			return nil, stacktrace.Propagate(err, "Error restoring store")
		}

		return nil, stacktrace.Propagate(err, "Error inserting Subscription")
	}
	return ret, nil
}

func (r *repo) updateSubscriptionTransactionApplier(ctx context.Context, proposal consensus.Proposal) (*ridmodels.Subscription, error) {
	var payload *ridmodels.Subscription
	if err := json.Unmarshal(proposal.Value, &payload); err != nil {
		return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", updateSubscription)
	}

	old, err := r.memRepo.GetSubscription(ctx, payload.ID)
	switch {
	case err != nil:
		return nil, stacktrace.Propagate(err, "Error getting Subscription from repo")
	case old == nil:
		return nil, stacktrace.NewErrorWithCode(dsserr.NotFound, "Subscription %s not found", payload.ID.String())
	case !payload.Version.Matches(old.Version):
		return nil, stacktrace.Propagate(
			stacktrace.NewErrorWithCode(dsserr.VersionMismatch, "Subscription version %s is not current", payload.Version),
			"Subscription currently at version %s but client specified %s", old.Version, payload.Version)
	case old.Owner != payload.Owner:
		return nil, stacktrace.Propagate(
			stacktrace.NewErrorWithCode(dsserr.PermissionDenied, "Subscription is owned by different client"),
			"Subscription owned by %s, but %s attempted to update", old.Owner, payload.Owner)
	}
	if err := payload.AdjustTimeRange(proposal.Timestamp, old); err != nil {
		return nil, stacktrace.Propagate(err, "Error adjusting time range")
	}

	count, err := r.memRepo.MaxSubscriptionCountInCellsByOwner(ctx, payload.Cells, payload.Owner)
	if err != nil {
		return nil, stacktrace.Propagate(err,
			"Failed to fetch subscription count, rejecting request")
	}
	if count >= ridmodels.MaxSubscriptionsPerArea {
		return nil, stacktrace.Propagate(
			stacktrace.NewErrorWithCode(dsserr.Exhausted, "Too many existing subscriptions in this area already"),
			"%s had %d subscriptions in the area", payload.Owner, count)
	}

	checkpoint := r.memStore.Checkpoint()
	ret, err := r.memRepo.UpdateSubscription(ctx, payload)
	if err != nil {
		restoreErr := r.memStore.Restore(checkpoint)
		if restoreErr != nil {
			return nil, stacktrace.Propagate(err, "Error restoring store")
		}

		return nil, stacktrace.Propagate(err, "Error updating Subscription")
	}
	return ret, nil
}

type DeleteSubscriptionPayload struct {
	ID      dssmodels.ID       `json:"id"`
	Owner   dssmodels.Owner    `json:"owner"`
	Version *dssmodels.Version `json:"version"`
}

func (r *repo) deleteSubscriptionTransactionApplier(ctx context.Context, proposal consensus.Proposal) (*ridmodels.Subscription, error) {
	var payload *ridmodels.Subscription
	if err := json.Unmarshal(proposal.Value, &payload); err != nil {
		return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", deleteSubscription)
	}

	old, err := r.memRepo.GetSubscription(ctx, payload.ID)
	switch {
	case err != nil:
		return nil, stacktrace.Propagate(err, "Error getting Subscription from repo")
	case old == nil:
		return nil, stacktrace.NewErrorWithCode(dsserr.NotFound, "Subscription %s not found", payload.ID.String())
	case !payload.Version.Matches(old.Version):
		return nil, stacktrace.Propagate(
			stacktrace.NewErrorWithCode(dsserr.VersionMismatch, "Subscription version %s is not current", payload.Version),
			"Subscription currently at version %s but client specified %s", old.Version, payload.Version)
	case old.Owner != payload.Owner:
		return nil, stacktrace.Propagate(
			stacktrace.NewErrorWithCode(dsserr.PermissionDenied, "Subscription is owned by different client"),
			"Subscription owned by %s, but %s attempted to delete", old.Owner, payload.Owner)
	}

	checkpoint := r.memStore.Checkpoint()
	ret, err := r.memRepo.DeleteSubscription(ctx, old)
	if err != nil {
		restoreErr := r.memStore.Restore(checkpoint)
		if restoreErr != nil {
			return nil, stacktrace.Propagate(err, "Error restoring store")
		}

		return nil, stacktrace.Propagate(err, "Error deleting Subscription")
	}
	return ret, nil
}
