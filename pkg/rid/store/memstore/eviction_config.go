package memstore

import (
	"context"

	dsserr "github.com/interuss/dss/pkg/errors"
	ridmodels "github.com/interuss/dss/pkg/rid/models"
	"github.com/interuss/stacktrace"
)

func (r *repo) GetEvictionConfig(_ context.Context) (*ridmodels.EvictionConfig, error) {
	return r.state.EvictionConfig.Clone(), nil
}

func (r *repo) SetEvictionConfig(_ context.Context, config ridmodels.EvictionConfig) (*ridmodels.EvictionConfig, error) {
	if err := config.Validate(); err != nil {
		return nil, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Invalid eviction config")
	}
	r.state.EvictionConfig = config.Clone()
	return config.Clone(), nil
}
