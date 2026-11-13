package memstore

import (
	"context"
	"testing"
	"time"

	dsserr "github.com/interuss/dss/pkg/errors"
	dssmodels "github.com/interuss/dss/pkg/models"
	scdmodels "github.com/interuss/dss/pkg/scd/models"
	"github.com/interuss/stacktrace"
	"github.com/stretchr/testify/require"
)

var testEvictionConfig = scdmodels.EvictionConfig{
	Interval:           24 * time.Hour,
	OperationalIntents: &dssmodels.EvictionParameters{TTL: 2688 * time.Hour},
}

func TestEvictionConfigNotSet(t *testing.T) {
	got, err := setUpStore(t).GetEvictionConfig(context.Background())
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestSetEvictionConfig(t *testing.T) {
	ctx := context.Background()
	r := setUpStore(t)

	got, err := r.SetEvictionConfig(ctx, testEvictionConfig)
	require.NoError(t, err)
	require.Equal(t, &testEvictionConfig, got)
	stored, err := r.GetEvictionConfig(ctx)
	require.NoError(t, err)
	require.Equal(t, &testEvictionConfig, stored)

	replacement := scdmodels.EvictionConfig{
		Interval:      time.Hour,
		Subscriptions: &dssmodels.EvictionParameters{TTL: time.Hour, Limit: 3},
	}
	_, err = r.SetEvictionConfig(ctx, replacement)
	require.NoError(t, err)
	stored, err = r.GetEvictionConfig(ctx)
	require.NoError(t, err)
	require.Equal(t, &replacement, stored, "the whole configuration is replaced: operational intents, not given, are disabled")
}

func TestSetEvictionConfigInvalid(t *testing.T) {
	ctx := context.Background()
	r := setUpStore(t)

	_, err := r.SetEvictionConfig(ctx, scdmodels.EvictionConfig{OperationalIntents: testEvictionConfig.OperationalIntents})
	require.Error(t, err, "the interval is required")
	require.Equal(t, dsserr.BadRequest, stacktrace.GetCode(err))
	got, err := r.GetEvictionConfig(ctx)
	require.NoError(t, err)
	require.Nil(t, got, "an invalid set changes nothing")

	_, err = r.SetEvictionConfig(ctx, testEvictionConfig)
	require.NoError(t, err)
	_, err = r.SetEvictionConfig(ctx, scdmodels.EvictionConfig{Interval: -time.Hour})
	require.Error(t, err)
	got, err = r.GetEvictionConfig(ctx)
	require.NoError(t, err)
	require.Equal(t, &testEvictionConfig, got, "an invalid set changes nothing")
}
