package raftstore

import (
	"context"
	"time"

	"github.com/golang/geo/s2"
	dssmodels "github.com/interuss/dss/pkg/models"
	scdmodels "github.com/interuss/dss/pkg/scd/models"
	"github.com/interuss/stacktrace"
)

func (r *repo) SearchSubscriptions(ctx context.Context, v4d *dssmodels.Volume4D) ([]*scdmodels.Subscription, error) {
	result, err := r.propose(ctx, searchSubscriptions, v4d)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to propose %s", searchSubscriptions)
	}

	subs, ok := result.([]*scdmodels.Subscription)
	if !ok {
		return nil, stacktrace.NewError("unexpected result type %T for %s", result, searchSubscriptions)
	}

	return subs, nil
}

func (r *repo) GetSubscription(ctx context.Context, id dssmodels.ID) (*scdmodels.Subscription, error) {
	result, err := r.propose(ctx, getSubscription, id)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to propose %s", getSubscription)
	}

	sub, ok := result.(*scdmodels.Subscription)
	if !ok {
		return nil, stacktrace.NewError("unexpected result type %T for %s", result, getSubscription)
	}

	return sub, nil
}

func (r *repo) UpsertSubscription(ctx context.Context, sub *scdmodels.Subscription) (*scdmodels.Subscription, error) {
	result, err := r.propose(ctx, upsertSubscription, sub)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to propose %s", upsertSubscription)
	}

	upsertedSub, ok := result.(*scdmodels.Subscription)
	if !ok {
		return nil, stacktrace.NewError("unexpected result type %T for %s", result, upsertSubscription)
	}

	return upsertedSub, nil
}

func (r *repo) DeleteSubscription(ctx context.Context, id dssmodels.ID) error {
	_, err := r.propose(ctx, deleteSubscription, id)
	return err
}

func (r *repo) IncrementNotificationIndices(ctx context.Context, subscriptionIds []dssmodels.ID) ([]int, error) {
	result, err := r.propose(ctx, incrementNotificationIdxs, subscriptionIds)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to propose %s", incrementNotificationIdxs)
	}

	indices, ok := result.([]int)
	if !ok {
		return nil, stacktrace.NewError("unexpected result type %T for %s", result, incrementNotificationIdxs)
	}

	return indices, nil
}

type lockSubscriptionsOnCellsPayload struct {
	Cells           s2.CellUnion   `json:"cells"`
	SubscriptionIds []dssmodels.ID `json:"subscription_ids"`
	StartTime       *time.Time     `json:"start_time,omitempty"`
	EndTime         *time.Time     `json:"end_time,omitempty"`
}

func (r *repo) LockSubscriptionsOnCells(ctx context.Context, cells s2.CellUnion, subscriptionIds []dssmodels.ID, startTime *time.Time, endTime *time.Time) error {
	_, err := r.propose(ctx, lockSubscriptionCells, &lockSubscriptionsOnCellsPayload{
		Cells:           cells,
		SubscriptionIds: subscriptionIds,
		StartTime:       startTime,
		EndTime:         endTime,
	})

	return err
}

func (r *repo) ListExpiredSubscriptions(ctx context.Context, threshold time.Time) ([]*scdmodels.Subscription, error) {
	result, err := r.propose(ctx, listExpiredSubscriptions, threshold)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to propose %s", listExpiredSubscriptions)
	}

	subs, ok := result.([]*scdmodels.Subscription)
	if !ok {
		return nil, stacktrace.NewError("unexpected result type %T for %s", result, listExpiredSubscriptions)
	}

	return subs, nil
}

func (r *repo) CountSubscriptions(ctx context.Context) (int64, error) {
	result, err := r.propose(ctx, countSubscriptions, nil)
	if err != nil {
		return 0, stacktrace.Propagate(err, "failed to propose %s", countSubscriptions)
	}

	count, ok := result.(int64)
	if !ok {
		return 0, stacktrace.NewError("unexpected result type %T for %s", result, countSubscriptions)
	}

	return count, nil
}
