package raftstore

import (
	"context"

	dssmodels "github.com/interuss/dss/pkg/models"
)

func (r *repo) GetEvictionConfig(ctx context.Context) (*dssmodels.EvictionConfig, error) {
	return r.eviction.GetConfig(ctx)
}

func (r *repo) SetEvictionConfig(ctx context.Context, config dssmodels.EvictionConfig) (*dssmodels.EvictionConfig, error) {
	return r.eviction.SetConfig(ctx, config)
}
