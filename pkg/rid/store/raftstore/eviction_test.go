package raftstore

import (
	"context"
	"testing"
	"time"

	"github.com/golang/geo/s2"
	dssmodels "github.com/interuss/dss/pkg/models"
	modelspb "github.com/interuss/dss/pkg/pb"
	"github.com/interuss/dss/pkg/pb/ridpb"
	"github.com/interuss/dss/pkg/raftstore/consensus"
	ridmodels "github.com/interuss/dss/pkg/rid/models"
	"github.com/interuss/dss/pkg/rid/operations"
	ridmemstore "github.com/interuss/dss/pkg/rid/store/memstore"
	"github.com/interuss/dss/pkg/timestamp"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestApplyEvictionRequest(t *testing.T) {
	now := time.Date(2026, time.November, 13, 12, 0, 0, 0, time.UTC)
	ctx := timestamp.NewContext(context.Background(), now)

	memStore, err := ridmemstore.Init(ctx, zap.NewNop())
	require.NoError(t, err)
	r := newRepo(memStore)

	start := now.Add(-3 * time.Hour)
	for id, end := range map[dssmodels.ID]time.Time{
		"00000000-0000-0000-0000-000000000001": now.Add(-2 * time.Hour),
		"00000000-0000-0000-0000-000000000002": now,
	} {
		_, err := memStore.GetRepo().InsertISA(ctx, &ridmodels.IdentificationServiceArea{
			ID:    id,
			Owner: "owner",
			URL:   "https://example.com",
			CellsVolume4D: &dssmodels.CellsVolume4D{
				Cells:     s2.CellUnion{s2.CellID(17106221850767130624)},
				StartTime: &start,
				EndTime:   &end,
			},
		})
		require.NoError(t, err)
	}

	apply := func(req *modelspb.EvictionRequest) (any, error) {
		return r.Apply(ctx, consensus.Proposal{Timestamp: now, Payload: operations.ProposalPayload(&ridpb.Payload{
			Payload: &ridpb.Payload_EvictionRequest{EvictionRequest: req},
		})})
	}
	cleanup := func(del bool) (any, error) {
		return apply(&modelspb.EvictionRequest{Request: &modelspb.EvictionRequest_Cleanup{Cleanup: &modelspb.CleanupRepoCall{
			Entities: map[string]*modelspb.EvictionParameters{evictionISAs: {Ttl: durationpb.New(time.Hour)}},
			Delete:   del,
		}}})
	}

	require.Equal(t, []string{evictionISAs, evictionSubscriptions}, r.eviction.EntityTypes())

	// Deleting needs the memstore DeleteExpired* of #1974: only counting is checked until it is merged.
	got, err := cleanup(false)
	require.NoError(t, err)
	require.Equal(t, map[string]int{evictionISAs: 1}, got, "the ISA ended at least one TTL ago is counted")

	config := dssmodels.EvictionConfig{Interval: time.Hour, Entities: map[string]dssmodels.EvictionParameters{evictionSubscriptions: {TTL: time.Hour}}}
	got, err = apply(&modelspb.EvictionRequest{Request: &modelspb.EvictionRequest_SetConfig{SetConfig: &modelspb.SetEvictionConfigRepoCall{Config: config.ToProto()}}})
	require.NoError(t, err)
	require.Equal(t, &config, got)
	got, err = apply(&modelspb.EvictionRequest{Request: &modelspb.EvictionRequest_GetConfig{GetConfig: &modelspb.GetEvictionConfigRepoCall{}}})
	require.NoError(t, err)
	require.Equal(t, &config, got, "the settings are stored in the memstore")
}
