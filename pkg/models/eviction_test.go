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
