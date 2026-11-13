package models

import (
	"testing"
	"time"

	modelspb "github.com/interuss/dss/pkg/pb"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestEvictionParametersFromProto(t *testing.T) {
	testCases := []struct {
		name    string
		pb      *modelspb.EvictionParameters
		want    *EvictionParameters
		wantErr bool
	}{{
		name: "missing is disabled",
		pb:   nil,
		want: nil,
	}, {
		name: "present is enabled",
		pb:   &modelspb.EvictionParameters{Ttl: durationpb.New(time.Minute), Limit: 2},
		want: &EvictionParameters{TTL: time.Minute, Limit: 2},
	}, {
		name:    "present without fields is validated",
		pb:      &modelspb.EvictionParameters{},
		wantErr: true,
	}, {
		name:    "negative TTL",
		pb:      &modelspb.EvictionParameters{Ttl: durationpb.New(-time.Minute)},
		wantErr: true,
	}, {
		name:    "negative limit",
		pb:      &modelspb.EvictionParameters{Ttl: durationpb.New(time.Minute), Limit: -1},
		wantErr: true,
	}}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := EvictionParametersFromProto(testCase.pb)
			if testCase.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testCase.want, got)
		})
	}
}

func TestEvictionParametersUpdateApply(t *testing.T) {
	current := &EvictionParameters{TTL: time.Hour, Limit: 5}
	replacement := &EvictionParameters{TTL: time.Minute}

	require.Equal(t, current, EvictionParametersUpdate{}.Apply(current), "an empty update changes nothing")
	require.Nil(t, EvictionParametersUpdate{}.Apply(nil), "an empty update keeps an entity type disabled")
	require.Nil(t, EvictionParametersUpdate{Set: true}.Apply(current), "disable")
	require.Equal(t, replacement, EvictionParametersUpdate{Set: true, Parameters: replacement}.Apply(current), "replace")
	require.Equal(t, replacement, EvictionParametersUpdate{Set: true, Parameters: replacement}.Apply(nil), "enable")

	got := EvictionParametersUpdate{Set: true, Parameters: replacement}.Apply(nil)
	got.Limit = 3
	require.Zero(t, replacement.Limit, "the applied parameters don't alias the update")
}

func TestEvictionParametersUpdateProtoRoundTrip(t *testing.T) {
	for name, u := range map[string]EvictionParametersUpdate{
		"unchanged": {},
		"disable":   {Set: true},
		"replace":   {Set: true, Parameters: &EvictionParameters{TTL: time.Minute, Limit: 2}},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := EvictionParametersUpdateFromProto(u.ToProto())
			require.NoError(t, err)
			require.Equal(t, u, got)
		})
	}

	t.Run("without a change is unchanged", func(t *testing.T) {
		got, err := EvictionParametersUpdateFromProto(&modelspb.EvictionParametersUpdate{})
		require.NoError(t, err)
		require.Equal(t, EvictionParametersUpdate{}, got)
	})

	t.Run("invalid parameters are rejected", func(t *testing.T) {
		_, err := EvictionParametersUpdateFromProto(&modelspb.EvictionParametersUpdate{
			Change: &modelspb.EvictionParametersUpdate_Parameters{Parameters: &modelspb.EvictionParameters{}},
		})
		require.Error(t, err)
	})
}

func TestEvictionParametersRoundTrip(t *testing.T) {
	for _, p := range []*EvictionParameters{
		{TTL: 30 * time.Minute, Limit: 5},
		nil,
	} {
		got, err := EvictionParametersFromProto(p.ToProto())
		require.NoError(t, err)
		require.Equal(t, p, got)
	}
}
