package memstore

import (
	"context"
	"time"

	dssmodels "github.com/interuss/dss/pkg/models"
)

// countExpired returns the number of records whose end time is at or before threshold.
func countExpired[R expiringRecord](store map[dssmodels.ID]R, threshold time.Time) int {
	count := 0
	for _, rec := range store {
		// ends_at <= threshold
		if !rec.endTime().After(threshold) {
			count++
		}
	}
	return count
}

func (r *repo) CountExpiredOperationalIntents(_ context.Context, threshold time.Time) (int, error) {
	return countExpired(r.state.OperationalIntents, threshold), nil
}

func (r *repo) CountExpiredSubscriptions(_ context.Context, threshold time.Time) (int, error) {
	return countExpired(r.state.Subscriptions, threshold), nil
}
