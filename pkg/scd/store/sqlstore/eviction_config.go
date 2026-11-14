package sqlstore

import (
	"context"

	dsserr "github.com/interuss/dss/pkg/errors"
	dssmodels "github.com/interuss/dss/pkg/models"
	"github.com/interuss/stacktrace"
)

func (c *repo) GetEvictionConfig(_ context.Context) (*dssmodels.EvictionConfig, error) {
	return nil, stacktrace.NewErrorWithCode(dsserr.NotImplemented, "GetEvictionConfig not implemented for sqlstore")
}

func (c *repo) SetEvictionConfig(_ context.Context, _ dssmodels.EvictionConfig) (*dssmodels.EvictionConfig, error) {
	return nil, stacktrace.NewErrorWithCode(dsserr.NotImplemented, "SetEvictionConfig not implemented for sqlstore")
}
