package raftstore

import (
	"context"
	"testing"
	"time"

	"github.com/golang/geo/s2"
	dssmodels "github.com/interuss/dss/pkg/models"
	modelspb "github.com/interuss/dss/pkg/pb"
	"github.com/interuss/dss/pkg/pb/scdpb"
	"github.com/interuss/dss/pkg/raftstore/consensus"
	scdmodels "github.com/interuss/dss/pkg/scd/models"
	"github.com/interuss/dss/pkg/scd/operations"
	scdmemstore "github.com/interuss/dss/pkg/scd/store/memstore"
	"github.com/interuss/dss/pkg/timestamp"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestApplyEvictionRequest(t *testing.T) {
	now := time.Date(2026, time.November, 13, 12, 0, 0, 0, time.UTC)
	ctx := timestamp.NewContext(context.Background(), now)

	memStore, err := scdmemstore.Init(ctx, zap.NewNop())
	require.NoError(t, err)
	r := newRepo(memStore)

	start := now.Add(-3 * time.Hour)
	for id, end := range map[dssmodels.ID]time.Time{
		"00000000-0000-0000-0000-000000000001": now.Add(-2 * time.Hour),
		"00000000-0000-0000-0000-000000000002": now,
	} {
		_, err := memStore.GetRepo().UpsertSubscription(ctx, &scdmodels.Subscription{
			ID:         id,
			Manager:    "manager",
			USSBaseURL: "https://example.com",
			CellsVolume4D: &dssmodels.CellsVolume4D{
				Cells:     s2.CellUnion{s2.CellID(8768904281496485888)},
				StartTime: &start,
				EndTime:   &end,
			},
		})
		require.NoError(t, err)
	}

	apply := func(req *modelspb.EvictionRequest) (any, error) {
		return r.Apply(ctx, consensus.Proposal{Timestamp: now, Payload: operations.ProposalPayload(&scdpb.Payload{
			Payload: &scdpb.Payload_EvictionRequest{EvictionRequest: req},
		})})
	}

	require.Equal(t, []string{evictionOperationalIntents, evictionSubscriptions}, r.eviction.EntityTypes())

	// Deleting needs the memstore DeleteExpired* of #1974: only counting is checked until it is merged.
	got, err := apply(&modelspb.EvictionRequest{Request: &modelspb.EvictionRequest_Cleanup{Cleanup: &modelspb.CleanupRepoCall{
		Entities: map[string]*modelspb.EvictionParameters{evictionSubscriptions: {Ttl: durationpb.New(time.Hour)}},
	}}})
	require.NoError(t, err)
	require.Equal(t, map[string]int{evictionSubscriptions: 1}, got, "the subscription ended at least one TTL ago is counted")

	config := dssmodels.EvictionConfig{Interval: 24 * time.Hour, Entities: map[string]dssmodels.EvictionParameters{evictionOperationalIntents: {TTL: 2688 * time.Hour}}}
	got, err = apply(&modelspb.EvictionRequest{Request: &modelspb.EvictionRequest_SetConfig{SetConfig: &modelspb.SetEvictionConfigRepoCall{Config: config.ToProto()}}})
	require.NoError(t, err)
	require.Equal(t, &config, got)
	got, err = apply(&modelspb.EvictionRequest{Request: &modelspb.EvictionRequest_GetConfig{GetConfig: &modelspb.GetEvictionConfigRepoCall{}}})
	require.NoError(t, err)
	require.Equal(t, &config, got, "the settings are stored in the memstore")
}
