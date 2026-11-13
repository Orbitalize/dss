package raftstore

import (
	"context"

	"github.com/interuss/dss/pkg/pb/ridpb"
	ridmodels "github.com/interuss/dss/pkg/rid/models"
	"github.com/interuss/dss/pkg/rid/operations"
)

func (r *repo) GetEvictionConfig(ctx context.Context) (*ridmodels.EvictionConfig, error) {
	return r.consensus.HandleReadPayload[*ridmodels.EvictionConfig](ctx, operations.ProposalPayload(&ridpb.Payload{Payload: &ridpb.Payload_GetEvictionConfigRepoCall{
		GetEvictionConfigRepoCall: &ridpb.GetEvictionConfigRepoCall{},
	}}))
}

func (r *repo) SetEvictionConfig(ctx context.Context, config ridmodels.EvictionConfig) (*ridmodels.EvictionConfig, error) {
	return r.consensus.HandleWritePayload[*ridmodels.EvictionConfig](ctx, operations.ProposalPayload(&ridpb.Payload{Payload: &ridpb.Payload_SetEvictionConfigRepoCall{
		SetEvictionConfigRepoCall: &ridpb.SetEvictionConfigRepoCall{Config: config.ToProto()},
	}}))
}
