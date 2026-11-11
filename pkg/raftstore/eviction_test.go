package raftstore

import (
	"context"
	"testing"
	"time"

	dssmodels "github.com/interuss/dss/pkg/models"
	"github.com/interuss/dss/pkg/timestamp"
	"github.com/stretchr/testify/require"
)

func TestEvict(t *testing.T) {
	now := time.Date(2026, time.November, 11, 12, 0, 0, 0, time.UTC)
	ctx := timestamp.NewContext(context.Background(), now)
	params := &dssmodels.EvictionParameters{TTL: time.Hour, Limit: 2}

	var deleteThreshold, countThreshold time.Time
	var deleteLimit int
	deleteExpired := func(_ context.Context, threshold time.Time, limit int) ([]dssmodels.ID, error) {
		deleteThreshold, deleteLimit = threshold, limit
		return []dssmodels.ID{"a", "b"}, nil
	}
	countExpired := func(_ context.Context, threshold time.Time) (int, error) {
		countThreshold = threshold
		return 3, nil
	}

	t.Run("nil parameters do nothing", func(t *testing.T) {
		count, err := Evict(ctx, nil, true,
			func(context.Context, time.Time, int) ([]dssmodels.ID, error) {
				t.Fatal("deleteExpired called")
				return nil, nil
			},
			func(context.Context, time.Time) (int, error) {
				t.Fatal("countExpired called")
				return 0, nil
			})
		require.NoError(t, err)
		require.Zero(t, count)
	})

	t.Run("delete", func(t *testing.T) {
		count, err := Evict(ctx, params, true, deleteExpired, countExpired)
		require.NoError(t, err)
		require.Equal(t, 2, count)
		require.Equal(t, now.Add(-time.Hour), deleteThreshold)
		require.Equal(t, 2, deleteLimit)
	})

	t.Run("count ignores the limit", func(t *testing.T) {
		count, err := Evict(ctx, params, false, deleteExpired, countExpired)
		require.NoError(t, err)
		require.Equal(t, 3, count)
		require.Equal(t, now.Add(-time.Hour), countThreshold)
	})
}
