package memstore

import (
	"context"

	dsserr "github.com/interuss/dss/pkg/errors"
	dssmodels "github.com/interuss/dss/pkg/models"
	"github.com/interuss/stacktrace"
)

func (r *repo) GetEvictionConfig(_ context.Context) (*dssmodels.EvictionConfig, error) {
	return r.state.EvictionConfig.Clone(), nil
}

func (r *repo) SetEvictionConfig(_ context.Context, config dssmodels.EvictionConfig) (*dssmodels.EvictionConfig, error) {
	if err := config.Validate(); err != nil {
		return nil, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Invalid eviction config")
	}
	r.state.EvictionConfig = config.Clone()
	return config.Clone(), nil
}
