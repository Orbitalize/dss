package sqlstore

import (
	"context"

	ridsqlstore "github.com/interuss/dss/pkg/rid/store/sqlstore"
	"github.com/interuss/dss/pkg/sqlstore"
	"github.com/interuss/dss/pkg/surveillance/repos"
	"go.uber.org/zap"
)

const (
	// The current major schema version per sqlstore type.
	currentCrdbMajorSchemaVersion     = 4
	currentYugabyteMajorSchemaVersion = 1
)

// Init initializes the SQL-backed surveillance store. It return a concrete sqlstore.Store[surveillance.repos.Repository] providing the
// ability to interact with a database-backed store of surveillance information.
func Init(ctx context.Context, logger *zap.Logger, withCheckCron bool) (*sqlstore.Store[repos.Repository], error) {
	return ridsqlstore.InitCustom(
		ctx,
		"surveillance",
		currentCrdbMajorSchemaVersion,
		currentYugabyteMajorSchemaVersion,
		logger,
		withCheckCron,
	)
}
