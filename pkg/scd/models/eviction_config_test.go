package models

import (
	"testing"
	"time"

	dssmodels "github.com/interuss/dss/pkg/models"
	"github.com/interuss/dss/pkg/pb/scdpb"
	"github.com/stretchr/testify/require"
)

func TestEvictionConfigValidate(t *testing.T) {
	require.NoError(t, (&EvictionConfig{Interval: time.Hour}).Validate())
	require.Error(t, (&EvictionConfig{}).Validate())
	require.Error(t, (&EvictionConfig{Interval: -time.Hour}).Validate())
}

func TestEvictionConfigClone(t *testing.T) {
	c := &EvictionConfig{Interval: time.Hour, OperationalIntents: &dssmodels.EvictionParameters{TTL: time.Minute}}
	cp := c.Clone()
	require.Equal(t, c, cp)
	cp.OperationalIntents.TTL = time.Hour
	require.Equal(t, time.Minute, c.OperationalIntents.TTL, "the clone doesn't share the eviction parameters")
}

func TestEvictionConfigProtoRoundTrip(t *testing.T) {
	for name, c := range map[string]EvictionConfig{
		"all set":                      {Interval: time.Hour, OperationalIntents: &dssmodels.EvictionParameters{TTL: time.Minute, Limit: 2}, Subscriptions: &dssmodels.EvictionParameters{TTL: time.Hour}},
		"operational intents disabled": {Interval: time.Hour, Subscriptions: &dssmodels.EvictionParameters{TTL: time.Hour}},
		"all disabled":                 {Interval: time.Hour},
	} {
		t.Run(name, func(t *testing.T) {
			var got EvictionConfig
			require.NoError(t, got.FromProto(c.ToProto()))
			require.Equal(t, c, got)
		})
	}

	var got EvictionConfig
	require.Error(t, got.FromProto(&scdpb.EvictionConfig{}), "a missing interval is invalid")
}
