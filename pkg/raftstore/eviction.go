package raftstore

import (
	"context"
	"maps"
	"slices"
	"time"

	dsserr "github.com/interuss/dss/pkg/errors"
	dssmodels "github.com/interuss/dss/pkg/models"
	modelspb "github.com/interuss/dss/pkg/pb"
	"github.com/interuss/dss/pkg/pb/raftpb"
	"github.com/interuss/dss/pkg/raftstore/consensus"
	"github.com/interuss/dss/pkg/timestamp"
	"github.com/interuss/stacktrace"
)

// evict deletes, or only counts when del is false, the entities expired according to p and returns their number.
// A nil p skips the entity type.
// The threshold is computed from the proposal timestamp of ctx to ensure deterministic behavior.
func evict(
	ctx context.Context,
	p *dssmodels.EvictionParameters,
	del bool,
	deleteExpired func(ctx context.Context, threshold time.Time, limit int) ([]dssmodels.ID, error),
	countExpired func(ctx context.Context, threshold time.Time) (int, error),
) (int, error) {
	if p == nil {
		return 0, nil
	}
	threshold := timestamp.MustFromContext(ctx).Add(-p.TTL)
	if del {
		ids, err := deleteExpired(ctx, threshold, p.Limit)
		return len(ids), err
	}
	return countExpired(ctx, threshold)
}

// EntityEvictor deletes or counts the expired entities of an entity type, in the memstore of a Raft group.
type EntityEvictor struct {
	DeleteExpired func(ctx context.Context, threshold time.Time, limit int) ([]dssmodels.ID, error)
	CountExpired  func(ctx context.Context, threshold time.Time) (int, error)
}

// EvictionConfigStore stores the eviction settings of a Raft group, in its memstore.
type EvictionConfigStore interface {
	// GetEvictionConfig returns the eviction settings, or nil if they were never set.
	GetEvictionConfig(ctx context.Context) (*dssmodels.EvictionConfig, error)
	SetEvictionConfig(ctx context.Context, config dssmodels.EvictionConfig) (*dssmodels.EvictionConfig, error)
}

// Eviction evicts the expired entities of a Raft group.
type Eviction struct {
	Consensus *consensus.Consensus
	payload  func(*modelspb.EvictionRequest) *raftpb.Payload
	entities map[string]EntityEvictor
	config   EvictionConfigStore
}

func NewEviction(payload func(*modelspb.EvictionRequest) *raftpb.Payload, entities map[string]EntityEvictor, config EvictionConfigStore) *Eviction {
	return &Eviction{payload: payload, entities: entities, config: config}
}

// EntityTypes returns the names of the entity types of the group, sorted.
func (e *Eviction) EntityTypes() []string {
	return slices.Sorted(maps.Keys(e.entities))
}

// GetConfig returns the eviction settings of the group, or nil if they were never set.
func (e *Eviction) GetConfig(ctx context.Context) (*dssmodels.EvictionConfig, error) {
	return e.Consensus.HandleReadPayload[*dssmodels.EvictionConfig](ctx, e.payload(&modelspb.EvictionRequest{
		Request: &modelspb.EvictionRequest_GetConfig{GetConfig: &modelspb.GetEvictionConfigRepoCall{}},
	}))
}

// SetConfig replaces the eviction settings of the group and returns the stored settings.
func (e *Eviction) SetConfig(ctx context.Context, config dssmodels.EvictionConfig) (*dssmodels.EvictionConfig, error) {
	if err := e.checkEntityTypes(config.Entities); err != nil {
		return nil, err
	}
	return e.Consensus.HandleWritePayload[*dssmodels.EvictionConfig](ctx, e.payload(&modelspb.EvictionRequest{
		Request: &modelspb.EvictionRequest_SetConfig{SetConfig: &modelspb.SetEvictionConfigRepoCall{Config: config.ToProto()}},
	}))
}

// Run runs a cleanup job on the entity types of entities and returns the number of entities per entity type.
// Deletions are proposed through the log, counts are served by the read path.
func (e *Eviction) Run(ctx context.Context, entities map[string]dssmodels.EvictionParameters, del bool) (map[string]int, error) {
	if err := e.checkEntityTypes(entities); err != nil {
		return nil, err
	}
	payload := e.payload(&modelspb.EvictionRequest{Request: &modelspb.EvictionRequest_Cleanup{Cleanup: &modelspb.CleanupRepoCall{
		Entities: dssmodels.EvictionParametersMapToProto(entities),
		Delete:   del,
	}}})
	if del {
		return e.Consensus.HandleWritePayload[map[string]int](ctx, payload)
	}
	return e.Consensus.HandleReadPayload[map[string]int](ctx, payload)
}

// Apply applies an eviction request on the memstore of the group. A cleanup must only be applied with delete set
// through the log, as the read path doesn't replicate changes.
func (e *Eviction) Apply(ctx context.Context, req *modelspb.EvictionRequest) (any, error) {
	switch r := req.GetRequest().(type) {
	case *modelspb.EvictionRequest_Cleanup:
		return e.applyCleanup(ctx, r.Cleanup)

	case *modelspb.EvictionRequest_GetConfig:
		return e.config.GetEvictionConfig(ctx)

	case *modelspb.EvictionRequest_SetConfig:
		var config dssmodels.EvictionConfig
		if err := config.FromProto(r.SetConfig.GetConfig()); err != nil {
			return nil, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Invalid eviction config")
		}
		if err := e.checkEntityTypes(config.Entities); err != nil {
			return nil, err
		}
		return e.config.SetEvictionConfig(ctx, config)

	default:
		return nil, stacktrace.NewError("unknown eviction request %T", req.GetRequest())
	}
}

// applyCleanup evicts the entity types in sorted order, so that every node applies the job the same way.
func (e *Eviction) applyCleanup(ctx context.Context, req *modelspb.CleanupRepoCall) (map[string]int, error) {
	entities, err := dssmodels.EvictionParametersMapFromProto(req.GetEntities())
	if err != nil {
		return nil, stacktrace.PropagateWithCode(err, dsserr.BadRequest, "Invalid cleanup job")
	}
	if err := e.checkEntityTypes(entities); err != nil {
		return nil, err
	}

	counts := make(map[string]int, len(entities))
	for _, name := range slices.Sorted(maps.Keys(entities)) {
		evictor := e.entities[name]
		p := entities[name]
		counts[name], err = evict(ctx, &p, req.GetDelete(), evictor.DeleteExpired, evictor.CountExpired)
		if err != nil {
			return nil, stacktrace.Propagate(err, "failed to evict %s", name)
		}
	}
	return counts, nil
}

// checkEntityTypes returns a dsserr.BadRequest error if entities has an entity type unknown for the group.
func (e *Eviction) checkEntityTypes(entities map[string]dssmodels.EvictionParameters) error {
	for name := range entities {
		if _, ok := e.entities[name]; !ok {
			return stacktrace.NewErrorWithCode(dsserr.BadRequest, "Unknown entity type %q, expected one of %v", name, e.EntityTypes())
		}
	}
	return nil
}
