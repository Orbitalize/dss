package repos

import (
	"context"

	ridmodels "github.com/interuss/dss/pkg/rid/models"
)

// EvictionConfig is the repository for the eviction settings of the RID group.
type EvictionConfig interface {
	// GetEvictionConfig returns the eviction settings, or nil if they were never set, in which case nothing is evicted.
	GetEvictionConfig(ctx context.Context) (*ridmodels.EvictionConfig, error)

	// SetEvictionConfig replaces the eviction settings with config and returns the stored settings.
	SetEvictionConfig(ctx context.Context, config ridmodels.EvictionConfig) (*ridmodels.EvictionConfig, error)
}
