package raftstore

import (
	"context"

	dsserr "github.com/interuss/dss/pkg/errors"
	"github.com/interuss/dss/pkg/raftstore"
	"github.com/interuss/dss/pkg/raftstore/consensus"
	"github.com/interuss/dss/pkg/rid/repos"
	"github.com/interuss/stacktrace"
	"go.uber.org/zap"
)

// repo is a full implementation of rid.repos.Repository for Raft-based storage.
type repo struct{}

func Init(ctx context.Context, logger *zap.Logger) (*raftstore.Store[repos.Repository], error) {
	store, _, err := raftstore.Init(ctx, logger, "rid", &repo{})
	return store, err
}

func (r *repo) GetRepo() repos.Repository { return r }

func (r *repo) IsReadOnly(_ raftstore.RequestType) bool { return false }

func (r *repo) GetSnapshot() ([]byte, error) {
	return nil, stacktrace.NewErrorWithCode(dsserr.NotImplemented, "not implemented yet")
}

func (r *repo) RestoreFromSnapshot([]byte) error {
	return stacktrace.NewErrorWithCode(dsserr.NotImplemented, "not implemented yet")
}

func (r *repo) Apply(_ context.Context, _ consensus.Proposal) (any, error) {
	return nil, stacktrace.NewErrorWithCode(dsserr.NotImplemented, "not implemented yet")
}
