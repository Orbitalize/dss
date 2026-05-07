package store

import (
	"context"

	dssstore "github.com/interuss/dss/pkg/store"
	"github.com/interuss/dss/pkg/store/params"
	"github.com/interuss/dss/pkg/surveillance/repos"
	survsqlstore "github.com/interuss/dss/pkg/surveillance/store/sqlstore"
	"github.com/interuss/stacktrace"
	"go.uber.org/zap"
)

type Store = dssstore.Store[repos.Repository]

// Init selects and initializes the surveillance store backend.
func Init(ctx context.Context, logger *zap.Logger, withCheckCron bool) (Store, error) {
	storeType := params.GetStoreParameters().StoreType
	switch storeType {
	case "sql":
		return survsqlstore.Init(ctx, logger, withCheckCron)
	default:
		return nil, stacktrace.NewError("Unsupported store type %q for surveillance", storeType)
	}
}
