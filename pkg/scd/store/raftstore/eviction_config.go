package raftstore

import (
	"context"

	"github.com/interuss/dss/pkg/pb/scdpb"
	scdmodels "github.com/interuss/dss/pkg/scd/models"
	"github.com/interuss/dss/pkg/scd/operations"
)

func (r *repo) GetEvictionConfig(ctx context.Context) (*scdmodels.EvictionConfig, error) {
	return r.consensus.HandleReadPayload[*scdmodels.EvictionConfig](ctx, operations.ProposalPayload(&scdpb.Payload{Payload: &scdpb.Payload_GetEvictionConfigRepoCall{
		GetEvictionConfigRepoCall: &scdpb.GetEvictionConfigRepoCall{},
	}}))
}

func (r *repo) SetEvictionConfig(ctx context.Context, config scdmodels.EvictionConfig) (*scdmodels.EvictionConfig, error) {
	return r.consensus.HandleWritePayload[*scdmodels.EvictionConfig](ctx, operations.ProposalPayload(&scdpb.Payload{Payload: &scdpb.Payload_SetEvictionConfigRepoCall{
		SetEvictionConfigRepoCall: &scdpb.SetEvictionConfigRepoCall{Config: config.ToProto()},
	}}))
}
