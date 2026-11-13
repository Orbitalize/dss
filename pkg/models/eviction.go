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

// EvictionParametersUpdate changes the stored eviction parameters of an entity type.
// The zero value leaves them unchanged.
type EvictionParametersUpdate struct {
	// Set is true when the update changes the parameters to Parameters. A nil Parameters disables the entity type.
	Set        bool
	Parameters *EvictionParameters
}

// Apply returns the parameters resulting from u applied to current.
func (u EvictionParametersUpdate) Apply(current *EvictionParameters) *EvictionParameters {
	if !u.Set {
		return current
	}
	if u.Parameters == nil {
		return nil
	}
	p := *u.Parameters
	return &p
}

// ToProto returns nil for an update leaving the parameters unchanged.
func (u EvictionParametersUpdate) ToProto() *modelspb.EvictionParametersUpdate {
	if !u.Set {
		return nil
	}
	if u.Parameters == nil {
		return &modelspb.EvictionParametersUpdate{Change: &modelspb.EvictionParametersUpdate_Disable{Disable: true}}
	}
	return &modelspb.EvictionParametersUpdate{Change: &modelspb.EvictionParametersUpdate_Parameters{Parameters: u.Parameters.ToProto()}}
}

// EvictionParametersUpdateFromProto converts and validates an update. A nil pb, or one without a change, leaves the
// parameters unchanged.
func EvictionParametersUpdateFromProto(pb *modelspb.EvictionParametersUpdate) (EvictionParametersUpdate, error) {
	switch change := pb.GetChange().(type) {
	case *modelspb.EvictionParametersUpdate_Parameters:
		if change.Parameters == nil {
			return EvictionParametersUpdate{}, stacktrace.NewError("missing parameters")
		}
		p, err := EvictionParametersFromProto(change.Parameters)
		if err != nil {
			return EvictionParametersUpdate{}, stacktrace.Propagate(err, "invalid parameters")
		}
		return EvictionParametersUpdate{Set: true, Parameters: p}, nil
	case *modelspb.EvictionParametersUpdate_Disable:
		return EvictionParametersUpdate{Set: change.Disable}, nil
	default:
		return EvictionParametersUpdate{}, nil
	}
}
