package raftstore

import (
	"context"
	"sync"

	"github.com/interuss/dss/pkg/logging"
	"github.com/interuss/dss/pkg/raftstore/consensus"
	raftparams "github.com/interuss/dss/pkg/raftstore/params"
	"github.com/interuss/dss/pkg/timestamp"
	"github.com/interuss/stacktrace"
	"go.uber.org/zap"
)

var (
	sharedConsensus     *consensus.Consensus
	sharedConsensusOnce sync.Once
	sharedConsensusErr  error
)

type RequestType = string

type RaftRepo[R any] interface {
	GetRepo() R
	// Apply is called on every committed entry. The proposal must be applied atomically.
	Apply(ctx context.Context, proposal consensus.Proposal) (any, error)
	GetSnapshot() ([]byte, error)
	RestoreFromSnapshot(data []byte) error
	IsReadOnly(requestType RequestType) bool
}

type Store[R any] struct {
	logger *zap.Logger

	name      string
	raftRepo  RaftRepo[R]
	consensus *consensus.Consensus
	cancel    context.CancelFunc
}

func Init[R any](ctx context.Context, logger *zap.Logger, name string, r RaftRepo[R]) (*Store[R], *consensus.Consensus, error) {
	// scd, rid and aux will share the same consensus instance, so we initialize it once.
	sharedConsensusOnce.Do(func() {
		params := raftparams.GetConnectParameters()
		peers, err := params.PeerMap()
		if err != nil {
			sharedConsensusErr = stacktrace.Propagate(err, "failed to parse peer map")
			return
		}

		sharedConsensus, sharedConsensusErr = consensus.NewConsensus(ctx, logger, peers, params)
		if sharedConsensusErr != nil {
			sharedConsensusErr = stacktrace.Propagate(sharedConsensusErr, "failed to initialize consensus")
		}
	})
	if sharedConsensusErr != nil {
		return nil, nil, sharedConsensusErr
	}

	ctx, cancel := context.WithCancel(ctx)

	store := &Store[R]{
		name:      name,
		raftRepo:  r,
		consensus: sharedConsensus,
		logger:    logging.WithValuesFromContext(ctx, logger),
		cancel:    cancel,
	}

	commitCh := sharedConsensus.RegisterStore(name, r.GetSnapshot)
	go store.processCommits(ctx, commitCh)

	return store, sharedConsensus, nil
}

// Transact proposes an entry to Raft and blocks until it is committed and applied.
// The processCommits loop will call Apply on the proposal when it is committed.
func (s *Store[R]) Transact(ctx context.Context, requestType RequestType, payload any, _ func(context.Context, R) error) (any, error) {
	proposal, err := consensus.NewProposal(ctx, s.name, requestType, payload, s.raftRepo.IsReadOnly(requestType), nil)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to create proposal")
	}

	return s.consensus.ProposeValue(ctx, proposal)
}

func (s *Store[R]) Interact(_ context.Context) (R, error) {
	return s.raftRepo.GetRepo(), nil
}

// Close shuts down the consensus instance and processCommits loop.
func (s *Store[R]) Close() error {
	s.consensus.Stop(context.Background())
	s.cancel()
	return nil
}

// processCommits reads committed entries from the consensus layer and applies them via Apply.
func (s *Store[R]) processCommits(ctx context.Context, commitCh <-chan consensus.EntryCommit) {
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("stopping commit processing loop")
			return
		case commit, ok := <-commitCh:
			if !ok {
				s.logger.Info("commit channel closed, stopping commit processing loop")
				return
			}

			if commit.SnapshotData != nil {
				if err := s.raftRepo.RestoreFromSnapshot(commit.SnapshotData); err != nil {
					s.logger.Error("failed to restore from snapshot", zap.Error(err))
				}
				continue
			}

			ctx = timestamp.WithTimestamp(ctx, commit.Prop.Timestamp)
			result, err := s.raftRepo.Apply(ctx, commit.Prop)
			commit.Done <- consensus.ProposalResult{Result: result, Error: err}
		}
	}
}
