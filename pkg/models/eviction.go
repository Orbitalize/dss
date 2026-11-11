package models

import (
	"maps"
	"time"

	modelspb "github.com/interuss/dss/pkg/pb"
	"github.com/interuss/stacktrace"
	"google.golang.org/protobuf/types/known/durationpb"
)

// EvictionParameters contains the cleanup parameters for an entity type.
type EvictionParameters struct {
	// TTL is how long after their end entities are expired.
	TTL time.Duration
	// Limit is the maximum number of entities evicted, 0 means unlimited.
	Limit int
}

func (p *EvictionParameters) ToProto() *modelspb.EvictionParameters {
	if p == nil {
		return nil
	}
	return &modelspb.EvictionParameters{
		Ttl:   durationpb.New(p.TTL),
		Limit: int64(p.Limit),
	}
}

// EvictionParametersFromProto converts and validates parameters. Missing parameters give nil.
func EvictionParametersFromProto(pb *modelspb.EvictionParameters) (*EvictionParameters, error) {
	if pb == nil {
		return nil, nil
	}
	if err := pb.GetTtl().CheckValid(); err != nil {
		return nil, stacktrace.Propagate(err, "invalid TTL")
	}
	p := &EvictionParameters{
		TTL:   pb.GetTtl().AsDuration(),
		Limit: int(pb.GetLimit()),
	}
	if p.TTL <= 0 {
		return nil, stacktrace.NewError("TTL must be positive, got %s", p.TTL)
	}
	if p.Limit < 0 {
		return nil, stacktrace.NewError("limit must not be negative, got %d", p.Limit)
	}
	return p, nil
}

// EvictionParametersMapToProto converts the parameters of entity types keyed by name.
func EvictionParametersMapToProto(entities map[string]EvictionParameters) map[string]*modelspb.EvictionParameters {
	result := make(map[string]*modelspb.EvictionParameters, len(entities))
	for name, p := range entities {
		result[name] = p.ToProto()
	}
	return result
}

// EvictionParametersMapFromProto converts and validates the parameters of entity types keyed by name.
func EvictionParametersMapFromProto(pb map[string]*modelspb.EvictionParameters) (map[string]EvictionParameters, error) {
	result := make(map[string]EvictionParameters, len(pb))
	for name, p := range pb {
		params, err := EvictionParametersFromProto(p)
		if err != nil {
			return nil, stacktrace.Propagate(err, "invalid eviction parameters of %s", name)
		}
		if params == nil {
			return nil, stacktrace.NewError("missing eviction parameters of %s", name)
		}
		result[name] = *params
	}
	return result, nil
}

// EvictionConfig contains the eviction settings of a Raft group.
type EvictionConfig struct {
	// Interval is the time between two scheduled cleanup jobs.
	Interval time.Duration
	// Entities maps the evicted entity types to their parameters. The entity types absent are not evicted.
	Entities map[string]EvictionParameters
}

// Validate checks the interval. The eviction parameters are validated when converted from their proto, and the
// entity types by the eviction of the group.
func (c *EvictionConfig) Validate() error {
	if c.Interval <= 0 {
		return stacktrace.NewError("interval must be positive, got %s", c.Interval)
	}
	return nil
}

// Clone returns a copy of c that doesn't share its entities, or nil for a nil c.
func (c *EvictionConfig) Clone() *EvictionConfig {
	if c == nil {
		return nil
	}
	return &EvictionConfig{Interval: c.Interval, Entities: maps.Clone(c.Entities)}
}

func (c *EvictionConfig) ToProto() *modelspb.EvictionConfig {
	return &modelspb.EvictionConfig{
		Interval: durationpb.New(c.Interval),
		Entities: EvictionParametersMapToProto(c.Entities),
	}
}

func (c *EvictionConfig) FromProto(pb *modelspb.EvictionConfig) error {
	if err := pb.GetInterval().CheckValid(); err != nil {
		return stacktrace.Propagate(err, "invalid interval")
	}
	entities, err := EvictionParametersMapFromProto(pb.GetEntities())
	if err != nil {
		return err
	}
	*c = EvictionConfig{Interval: pb.GetInterval().AsDuration(), Entities: entities}
	return nil
}
