package server

import (
	"context"
	"testing"
	"time"

	restapi "github.com/interuss/dss/pkg/api/adminv1"
	dsserr "github.com/interuss/dss/pkg/errors"
	"github.com/interuss/stacktrace"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeEvictor struct {
	config  *EvictionConfig
	lastRun *EvictionRun
	setErr  error
}

func (f *fakeEvictor) EntityTypes() []string { return []string{"isas", "subscriptions"} }

func (f *fakeEvictor) GetEvictionConfig(context.Context) (*EvictionConfig, error) {
	return f.config, nil
}

func (f *fakeEvictor) SetEvictionConfig(_ context.Context, config EvictionConfig) (*EvictionConfig, error) {
	if f.setErr != nil {
		return nil, f.setErr
	}
	f.config = &config
	return &config, nil
}

func (f *fakeEvictor) RunEviction(_ context.Context, run EvictionRun) (map[string]int, error) {
	f.lastRun = &run
	counts := map[string]int{}
	for entityType := range run.Entities {
		counts[entityType] = len(entityType)
	}
	return counts, nil
}

func setUpServer() (*Server, *fakeEvictor) {
	evictor := &fakeEvictor{}
	return &Server{Logger: zap.NewNop(), evictors: map[string]Evictor{"rid": evictor}}, evictor
}

func TestGetEvictionConfig(t *testing.T) {
	ctx := context.Background()
	s, evictor := setUpServer()

	resp := s.GetEvictionConfig(ctx, &restapi.GetEvictionConfigRequest{GroupName: "aux"})
	require.NotNil(t, resp.Response404, "a group without evictor")

	resp = s.GetEvictionConfig(ctx, &restapi.GetEvictionConfigRequest{GroupName: "rid"})
	require.NotNil(t, resp.Response200)
	require.Nil(t, resp.Response200.Config, "not configured")

	evictor.config = &EvictionConfig{
		Interval: time.Hour,
		Entities: map[string]EntityEviction{"subscriptions": {TTL: time.Minute}, "isas": {TTL: 30 * time.Minute, Limit: 5}},
	}
	resp = s.GetEvictionConfig(ctx, &restapi.GetEvictionConfigRequest{GroupName: "rid"})
	require.Equal(t, &restapi.EvictionConfig{
		Interval: "1h0m0s",
		Entities: []restapi.EntityEviction{
			{Type: "isas", Ttl: "30m0s", Limit: new(int64(5))},
			{Type: "subscriptions", Ttl: "1m0s", Limit: new(int64(0))},
		},
	}, resp.Response200.Config, "entity types in the order of the evictor")
}

func TestPutEvictionConfig(t *testing.T) {
	ctx := context.Background()
	s, evictor := setUpServer()

	resp := s.PutEvictionConfig(ctx, &restapi.PutEvictionConfigRequest{GroupName: "rid", Body: &restapi.EvictionConfig{
		Interval: "1h",
		Entities: []restapi.EntityEviction{{Type: "isas", Ttl: "30m"}},
	}})
	require.NotNil(t, resp.Response200)
	require.Equal(t, &EvictionConfig{Interval: time.Hour, Entities: map[string]EntityEviction{"isas": {TTL: 30 * time.Minute}}}, evictor.config)
	require.Equal(t, "1h0m0s", resp.Response200.Interval)

	for name, body := range map[string]*restapi.EvictionConfig{
		"missing interval":    {Entities: []restapi.EntityEviction{}},
		"invalid interval":    {Interval: "soon", Entities: []restapi.EntityEviction{}},
		"non-positive":        {Interval: "0s", Entities: []restapi.EntityEviction{}},
		"unknown entity type": {Interval: "1h", Entities: []restapi.EntityEviction{{Type: "constraints", Ttl: "1h"}}},
		"entity type twice":   {Interval: "1h", Entities: []restapi.EntityEviction{{Type: "isas", Ttl: "1h"}, {Type: "isas", Ttl: "2h"}}},
		"non-positive TTL":    {Interval: "1h", Entities: []restapi.EntityEviction{{Type: "isas", Ttl: "-1h"}}},
		"negative limit":      {Interval: "1h", Entities: []restapi.EntityEviction{{Type: "isas", Ttl: "1h", Limit: new(int64(-1))}}},
	} {
		t.Run(name, func(t *testing.T) {
			resp := s.PutEvictionConfig(ctx, &restapi.PutEvictionConfigRequest{GroupName: "rid", Body: body})
			require.NotNil(t, resp.Response400)
		})
	}

	t.Run("store rejection", func(t *testing.T) {
		evictor.setErr = stacktrace.NewErrorWithCode(dsserr.BadRequest, "rejected")
		resp := s.PutEvictionConfig(ctx, &restapi.PutEvictionConfigRequest{GroupName: "rid", Body: &restapi.EvictionConfig{Interval: "1h"}})
		require.NotNil(t, resp.Response400)
	})

	resp = s.PutEvictionConfig(ctx, &restapi.PutEvictionConfigRequest{GroupName: "aux", Body: &restapi.EvictionConfig{Interval: "1h"}})
	require.NotNil(t, resp.Response404)
}

func TestCreateEvictionRun(t *testing.T) {
	ctx := context.Background()
	s, evictor := setUpServer()

	resp := s.CreateEvictionRun(ctx, &restapi.CreateEvictionRunRequest{GroupName: "rid", Body: &restapi.EvictionRunParameters{
		Entities: []restapi.EntityEviction{{Type: "subscriptions", Ttl: "1h"}, {Type: "isas", Ttl: "30m", Limit: new(int64(2))}},
		Delete:   new(true),
	}})
	require.NotNil(t, resp.Response200)
	require.Equal(t, []restapi.EntityEvictionCount{{Type: "subscriptions", Count: 13}, {Type: "isas", Count: 4}}, resp.Response200.Counts, "counts in the order of the request")
	require.Equal(t, &EvictionRun{
		Entities: map[string]EntityEviction{"subscriptions": {TTL: time.Hour}, "isas": {TTL: 30 * time.Minute, Limit: 2}},
		Delete:   true,
	}, evictor.lastRun)

	resp = s.CreateEvictionRun(ctx, &restapi.CreateEvictionRunRequest{GroupName: "rid", Body: &restapi.EvictionRunParameters{
		Entities: []restapi.EntityEviction{{Type: "isas", Ttl: "30m"}},
	}})
	require.NotNil(t, resp.Response200)
	require.False(t, evictor.lastRun.Delete, "only counting by default")

	resp = s.CreateEvictionRun(ctx, &restapi.CreateEvictionRunRequest{GroupName: "rid", Body: &restapi.EvictionRunParameters{
		Entities: []restapi.EntityEviction{{Type: "isas", Ttl: "30m", Limit: new(int64(2))}},
	}})
	require.NotNil(t, resp.Response400, "a limit while only counting")

	resp = s.CreateEvictionRun(ctx, &restapi.CreateEvictionRunRequest{GroupName: "scd", Body: &restapi.EvictionRunParameters{}})
	require.NotNil(t, resp.Response404)
}
