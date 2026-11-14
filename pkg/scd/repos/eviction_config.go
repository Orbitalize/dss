package repos

import (
	"context"

	dssmodels "github.com/interuss/dss/pkg/models"
)

// EvictionConfig is the repository for the eviction settings of the SCD group.
type EvictionConfig interface {
	// GetEvictionConfig returns the eviction settings, or nil if they were never set, in which case nothing is evicted.
	GetEvictionConfig(ctx context.Context) (*dssmodels.EvictionConfig, error)

	// SetEvictionConfig replaces the eviction settings with config and returns the stored settings.
	SetEvictionConfig(ctx context.Context, config dssmodels.EvictionConfig) (*dssmodels.EvictionConfig, error)
}
