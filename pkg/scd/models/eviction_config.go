package models

import (
	"time"

	dssmodels "github.com/interuss/dss/pkg/models"
	"github.com/interuss/dss/pkg/pb/scdpb"
	"github.com/interuss/stacktrace"
	"google.golang.org/protobuf/types/known/durationpb"
)

// EvictionConfig contains the eviction settings of the SCD group.
type EvictionConfig struct {
	// Interval is the time between two scheduled cleanup jobs.
	Interval time.Duration
	// OperationalIntents and Subscriptions are not evicted when nil.
	OperationalIntents *dssmodels.EvictionParameters
	Subscriptions      *dssmodels.EvictionParameters
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
		Interval:           c.Interval,
		OperationalIntents: cloneEvictionParameters(c.OperationalIntents),
		Subscriptions:      cloneEvictionParameters(c.Subscriptions),
	}
}

func cloneEvictionParameters(p *dssmodels.EvictionParameters) *dssmodels.EvictionParameters {
	if p == nil {
		return nil
	}
	cp := *p
	return &cp
}

func (c *EvictionConfig) ToProto() *scdpb.EvictionConfig {
	return &scdpb.EvictionConfig{
		Interval:           durationpb.New(c.Interval),
		OperationalIntents: c.OperationalIntents.ToProto(),
		Subscriptions:      c.Subscriptions.ToProto(),
	}
}

func (c *EvictionConfig) FromProto(pb *scdpb.EvictionConfig) error {
	if err := pb.GetInterval().CheckValid(); err != nil {
		return stacktrace.Propagate(err, "invalid interval")
	}
	ois, err := dssmodels.EvictionParametersFromProto(pb.GetOperationalIntents())
	if err != nil {
		return stacktrace.Propagate(err, "invalid Operational Intents eviction parameters")
	}
	subs, err := dssmodels.EvictionParametersFromProto(pb.GetSubscriptions())
	if err != nil {
		return stacktrace.Propagate(err, "invalid Subscriptions eviction parameters")
	}
	*c = EvictionConfig{
		Interval:           pb.GetInterval().AsDuration(),
		OperationalIntents: ois,
		Subscriptions:      subs,
	}
	return nil
}
