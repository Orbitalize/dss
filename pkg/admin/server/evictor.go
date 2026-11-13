package server

import (
	"context"
	"time"
)

// EntityEviction contains the eviction parameters of an entity type.
type EntityEviction struct {
	// TTL is how long after their end entities are expired.
	TTL time.Duration
	// Limit is the maximum number of entities evicted by a cleanup job, 0 means unlimited.
	Limit int
}

// EvictionConfig contains the eviction settings of a Raft group.
type EvictionConfig struct {
	// Interval is the time between two scheduled cleanup jobs.
	Interval time.Duration
	// Entities maps the evicted entity types to their parameters. The entity types absent are not evicted.
	Entities map[string]EntityEviction
}

// EvictionRun is a cleanup job run on demand.
type EvictionRun struct {
	// Entities maps the entity types to evict to their parameters. The entity types absent are skipped.
	Entities map[string]EntityEviction
	// Delete deletes the expired entities, otherwise they are only counted.
	Delete bool
}

// Evictor evicts the expired entities of a Raft group.
type Evictor interface {
	// EntityTypes returns the entity types of the group, as named in the admin API.
	EntityTypes() []string

	// GetEvictionConfig returns the eviction settings, or nil if they were never set.
	GetEvictionConfig(ctx context.Context) (*EvictionConfig, error)

	// SetEvictionConfig replaces the eviction settings and returns the stored settings.
	SetEvictionConfig(ctx context.Context, config EvictionConfig) (*EvictionConfig, error)

	// RunEviction runs a cleanup job and returns the number of entities per entity type of run.
	RunEviction(ctx context.Context, run EvictionRun) (map[string]int, error)
}
