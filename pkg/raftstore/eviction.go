package raftstore

import (
	"context"
	"time"

	dssmodels "github.com/interuss/dss/pkg/models"
	"github.com/interuss/dss/pkg/timestamp"
)

// Evict deletes, or only counts when del is false, the entities expired according to p and returns their number.
// A nil p skips the entity type.
// The limit only applies to deletions: like listing, counting reports every expired entity.
// The threshold is computed from the proposal timestamp of ctx to ensure deterministic behavior.
func Evict(
	ctx context.Context,
	p *dssmodels.EvictionParameters,
	del bool,
	deleteExpired func(ctx context.Context, threshold time.Time, limit int) ([]dssmodels.ID, error),
	countExpired func(ctx context.Context, threshold time.Time) (int, error),
) (int, error) {
	if p == nil {
		return 0, nil
	}
	threshold := timestamp.MustFromContext(ctx).Add(-p.TTL)
	if del {
		ids, err := deleteExpired(ctx, threshold, p.Limit)
		return len(ids), err
	}
	return countExpired(ctx, threshold)
}
