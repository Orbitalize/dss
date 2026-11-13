package raftstore

import (
	"context"
	"time"

	"github.com/jonboulle/clockwork"
	"go.uber.org/zap"
)

// fallbackInterval is waited when interval returns a non-positive duration.
const fallbackInterval = time.Minute

// RunAsLeader calls job after each interval while isLeader returns true, until ctx is done. It blocks, so it is meant
// to run in its own goroutine.
// interval is called before each wait, so that a change applies from the next wait, and the first call to job happens
// after one interval. A non-positive interval waits fallbackInterval. Errors returned by job are logged and the loop
// continues.
func RunAsLeader(ctx context.Context, logger *zap.Logger, isLeader func() bool, interval func() time.Duration, job func(context.Context) error) {
	runAsLeader(ctx, clockwork.NewRealClock(), logger, isLeader, interval, job)
}

func runAsLeader(ctx context.Context, clock clockwork.Clock, logger *zap.Logger, isLeader func() bool, interval func() time.Duration, job func(context.Context) error) {
	for {
		d := interval()
		if d <= 0 {
			d = fallbackInterval
		}

		timer := clock.NewTimer(d)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.Chan():
		}

		if !isLeader() {
			continue
		}
		if err := job(ctx); err != nil {
			logger.Warn("leader job failed", zap.Error(err))
		}
	}
}
