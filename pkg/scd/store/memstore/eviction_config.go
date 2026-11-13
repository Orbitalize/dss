package memstore

import (
	"context"

	dsserr "github.com/interuss/dss/pkg/errors"
	scdmodels "github.com/interuss/dss/pkg/scd/models"
	"github.com/interuss/stacktrace"
)

func (r *repo) GetEvictionConfig(_ context.Context) (*scdmodels.EvictionConfig, error) {
	return r.state.EvictionConfig.Clone(), nil
}

func (r *repo) SetEvictionConfig(_ context.Context, config scdmodels.EvictionConfig) (*scdmodels.EvictionConfig, error) {
	if err := config.Validate(); err != nil {
		return nil, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Invalid eviction config")
	}
	r.state.EvictionConfig = config.Clone()
	return config.Clone(), nil
}
