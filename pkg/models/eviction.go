package models

import (
	"time"

	modelspb "github.com/interuss/dss/pkg/pb"
	"github.com/interuss/stacktrace"
	"google.golang.org/protobuf/types/known/durationpb"
)

// EvictionParameters contains the cleanup parameters for an entity type.
// An entity type is cleaned up only when its parameters are set (non-nil).
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
