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

func TestEvictionParametersMapFromProto(t *testing.T) {
	_, err := EvictionParametersMapFromProto(map[string]*modelspb.EvictionParameters{"isas": nil})
	require.Error(t, err, "an entity type without parameters")
	_, err = EvictionParametersMapFromProto(map[string]*modelspb.EvictionParameters{"isas": {}})
	require.Error(t, err, "invalid parameters")
}

func TestEvictionConfigValidate(t *testing.T) {
	require.NoError(t, (&EvictionConfig{Interval: time.Hour}).Validate())
	require.Error(t, (&EvictionConfig{}).Validate())
	require.Error(t, (&EvictionConfig{Interval: -time.Hour}).Validate())
}

func TestEvictionConfigClone(t *testing.T) {
	c := &EvictionConfig{Interval: time.Hour, Entities: map[string]EvictionParameters{"isas": {TTL: time.Minute}}}
	cp := c.Clone()
	require.Equal(t, c, cp)
	cp.Entities["subscriptions"] = EvictionParameters{TTL: time.Hour}
	require.Len(t, c.Entities, 1, "the clone doesn't share the entities")

	require.Nil(t, (*EvictionConfig)(nil).Clone())
}

func TestEvictionConfigProtoRoundTrip(t *testing.T) {
	for name, c := range map[string]EvictionConfig{
		"entities set": {Interval: time.Hour, Entities: map[string]EvictionParameters{"isas": {TTL: time.Minute, Limit: 2}, "subscriptions": {TTL: time.Hour}}},
		"all disabled": {Interval: time.Hour, Entities: map[string]EvictionParameters{}},
	} {
		t.Run(name, func(t *testing.T) {
			var got EvictionConfig
			require.NoError(t, got.FromProto(c.ToProto()))
			require.Equal(t, c, got)
		})
	}

	var got EvictionConfig
	require.Error(t, got.FromProto(&modelspb.EvictionConfig{}), "a missing interval is invalid")
}
