package sqlstore

import (
	"context"

	dsserr "github.com/interuss/dss/pkg/errors"
	scdmodels "github.com/interuss/dss/pkg/scd/models"
	"github.com/interuss/stacktrace"
)

func (c *repo) GetEvictionConfig(_ context.Context) (*scdmodels.EvictionConfig, error) {
	return nil, stacktrace.NewErrorWithCode(dsserr.NotImplemented, "GetEvictionConfig not implemented for sqlstore")
}

func (c *repo) SetEvictionConfig(_ context.Context, _ scdmodels.EvictionConfig) (*scdmodels.EvictionConfig, error) {
	return nil, stacktrace.NewErrorWithCode(dsserr.NotImplemented, "SetEvictionConfig not implemented for sqlstore")
}
