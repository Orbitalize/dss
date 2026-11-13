package raftstore

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type testLoop struct {
	t     *testing.T
	ctx   context.Context
	clock *clockwork.FakeClock
	calls atomic.Int32
	stop  func()
	done  chan struct{}
}

// startLoop starts runAsLeader on a fake clock and waits for its first timer. job is called after the call is counted.
func startLoop(t *testing.T, isLeader func() bool, interval func() time.Duration, job func(context.Context) error) *testLoop {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	l := &testLoop{t: t, ctx: ctx, clock: clockwork.NewFakeClock(), done: make(chan struct{})}

	loopCtx, stop := context.WithCancel(ctx)
	l.stop = stop
	go func() {
		defer close(l.done)
		runAsLeader(loopCtx, l.clock, zap.NewNop(), isLeader, interval, func(ctx context.Context) error {
			l.calls.Add(1)
			return job(ctx)
		})
	}()
	t.Cleanup(func() {
		stop()
		<-l.done
		cancel()
	})

	l.waitForTimer()
	return l
}

// advance moves the fake clock and waits until the loop waits on its next timer, i.e. the tick is fully processed.
func (l *testLoop) advance(d time.Duration) {
	l.t.Helper()
	l.clock.Advance(d)
	l.waitForTimer()
}

func (l *testLoop) waitForTimer() {
	l.t.Helper()
	require.NoError(l.t, l.clock.BlockUntilContext(l.ctx, 1))
}

func leader() bool { return true }

func hourly() time.Duration { return time.Hour }

func noop(context.Context) error { return nil }

func TestRunAsLeaderRunsAfterEachInterval(t *testing.T) {
	l := startLoop(t, leader, hourly, noop)

	l.advance(59 * time.Minute)
	require.Zero(t, l.calls.Load(), "the first run happens after one interval")

	l.advance(time.Minute)
	require.Equal(t, int32(1), l.calls.Load())

	l.advance(time.Hour)
	require.Equal(t, int32(2), l.calls.Load())
}

func TestRunAsLeaderOnlyOnLeader(t *testing.T) {
	var isLeader atomic.Bool
	l := startLoop(t, isLeader.Load, hourly, noop)

	l.advance(time.Hour)
	require.Zero(t, l.calls.Load(), "a follower doesn't run the job")

	isLeader.Store(true)
	l.advance(time.Hour)
	require.Equal(t, int32(1), l.calls.Load(), "the job runs once the node is leader")
}

func TestRunAsLeaderReadsIntervalBeforeEachWait(t *testing.T) {
	var interval atomic.Int64
	interval.Store(int64(time.Hour))
	l := startLoop(t, leader, func() time.Duration { return time.Duration(interval.Load()) }, func(context.Context) error {
		interval.Store(int64(time.Minute))
		return nil
	})

	l.advance(time.Hour)
	require.Equal(t, int32(1), l.calls.Load())

	l.advance(time.Minute)
	require.Equal(t, int32(2), l.calls.Load(), "the new interval applies from the next wait")
}

func TestRunAsLeaderFallbackInterval(t *testing.T) {
	l := startLoop(t, leader, func() time.Duration { return 0 }, noop)

	l.advance(fallbackInterval - time.Second)
	require.Zero(t, l.calls.Load())

	l.advance(time.Second)
	require.Equal(t, int32(1), l.calls.Load(), "a non-positive interval waits the fallback interval")
}

func TestRunAsLeaderContinuesAfterJobError(t *testing.T) {
	l := startLoop(t, leader, hourly, func(context.Context) error { return errors.New("failure") })

	l.advance(time.Hour)
	l.advance(time.Hour)
	require.Equal(t, int32(2), l.calls.Load())
}

func TestRunAsLeaderStopsWithContext(t *testing.T) {
	l := startLoop(t, leader, hourly, noop)

	l.stop()
	select {
	case <-l.done:
	case <-l.ctx.Done():
		t.Fatal("the loop didn't stop")
	}
}
