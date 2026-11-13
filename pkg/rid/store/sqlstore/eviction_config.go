package sqlstore

import (
	"context"

	dsserr "github.com/interuss/dss/pkg/errors"
	ridmodels "github.com/interuss/dss/pkg/rid/models"
	"github.com/interuss/stacktrace"
)

func (r *repo) GetEvictionConfig(_ context.Context) (*ridmodels.EvictionConfig, error) {
	return nil, stacktrace.NewErrorWithCode(dsserr.NotImplemented, "GetEvictionConfig not implemented for sqlstore")
}

func (r *repo) SetEvictionConfig(_ context.Context, _ ridmodels.EvictionConfig) (*ridmodels.EvictionConfig, error) {
	return nil, stacktrace.NewErrorWithCode(dsserr.NotImplemented, "SetEvictionConfig not implemented for sqlstore")
}
