package memstore

import (
	"context"
	"testing"
	"time"

	dsserr "github.com/interuss/dss/pkg/errors"
	dssmodels "github.com/interuss/dss/pkg/models"
	ridmodels "github.com/interuss/dss/pkg/rid/models"
	"github.com/interuss/stacktrace"
	"github.com/stretchr/testify/require"
)

var testEvictionConfig = ridmodels.EvictionConfig{
	Interval: time.Hour,
	ISAs:     &dssmodels.EvictionParameters{TTL: 30 * time.Minute},
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

	replacement := ridmodels.EvictionConfig{
		Interval:      time.Minute,
		Subscriptions: &dssmodels.EvictionParameters{TTL: time.Hour, Limit: 3},
	}
	_, err = r.SetEvictionConfig(ctx, replacement)
	require.NoError(t, err)
	stored, err = r.GetEvictionConfig(ctx)
	require.NoError(t, err)
	require.Equal(t, &replacement, stored, "the whole configuration is replaced: ISAs, not given, are disabled")
}

func TestSetEvictionConfigInvalid(t *testing.T) {
	ctx := context.Background()
	r := setUpStore(t)

	_, err := r.SetEvictionConfig(ctx, ridmodels.EvictionConfig{ISAs: testEvictionConfig.ISAs})
	require.Error(t, err, "the interval is required")
	require.Equal(t, dsserr.BadRequest, stacktrace.GetCode(err))
	got, err := r.GetEvictionConfig(ctx)
	require.NoError(t, err)
	require.Nil(t, got, "an invalid set changes nothing")

	_, err = r.SetEvictionConfig(ctx, testEvictionConfig)
	require.NoError(t, err)
	_, err = r.SetEvictionConfig(ctx, ridmodels.EvictionConfig{Interval: -time.Hour})
	require.Error(t, err)
	got, err = r.GetEvictionConfig(ctx)
	require.NoError(t, err)
	require.Equal(t, &testEvictionConfig, got, "an invalid set changes nothing")
}
