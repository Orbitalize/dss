package models

import (
	"time"

	dssmodels "github.com/interuss/dss/pkg/models"
	"github.com/interuss/dss/pkg/pb/ridpb"
	"github.com/interuss/stacktrace"
	"google.golang.org/protobuf/types/known/durationpb"
)

// EvictionConfig contains the eviction settings of the RID group.
type EvictionConfig struct {
	// Interval is the time between two scheduled cleanup jobs.
	Interval time.Duration
	// ISAs and Subscriptions are not evicted when nil.
	ISAs          *dssmodels.EvictionParameters
	Subscriptions *dssmodels.EvictionParameters
}

// Validate checks the interval. The eviction parameters are validated when converted from their proto.
func (c *EvictionConfig) Validate() error {
	if c.Interval <= 0 {
		return stacktrace.NewError("interval must be positive, got %s", c.Interval)
	}

	return nil
}

// Clone returns a copy of c that doesn't share its eviction parameters, or nil for a nil c.
func (c *EvictionConfig) Clone() *EvictionConfig {
	if c == nil {
		return nil
	}
	return &EvictionConfig{
		Interval:      c.Interval,
		ISAs:          cloneEvictionParameters(c.ISAs),
		Subscriptions: cloneEvictionParameters(c.Subscriptions),
	}
}

func cloneEvictionParameters(p *dssmodels.EvictionParameters) *dssmodels.EvictionParameters {
	if p == nil {
		return nil
	}
	cp := *p
	return &cp
}

func (c *EvictionConfig) ToProto() *ridpb.EvictionConfig {
	return &ridpb.EvictionConfig{
		Interval:      durationpb.New(c.Interval),
		Isas:          c.ISAs.ToProto(),
		Subscriptions: c.Subscriptions.ToProto(),
	}
}

func (c *EvictionConfig) FromProto(pb *ridpb.EvictionConfig) error {
	if err := pb.GetInterval().CheckValid(); err != nil {
		return stacktrace.Propagate(err, "invalid interval")
	}
	isas, err := dssmodels.EvictionParametersFromProto(pb.GetIsas())
	if err != nil {
		return stacktrace.Propagate(err, "invalid ISAs eviction parameters")
	}
	subs, err := dssmodels.EvictionParametersFromProto(pb.GetSubscriptions())
	if err != nil {
		return stacktrace.Propagate(err, "invalid Subscriptions eviction parameters")
	}
	*c = EvictionConfig{
		Interval:      pb.GetInterval().AsDuration(),
		ISAs:          isas,
		Subscriptions: subs,
	}
	return nil
}
