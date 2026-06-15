package raftstore

import (
	"context"
	"encoding/json"

	auxmodels "github.com/interuss/dss/pkg/aux_/models"
	"github.com/interuss/dss/pkg/aux_/repos"
	auxmemstore "github.com/interuss/dss/pkg/aux_/store/memstore"
	dsserr "github.com/interuss/dss/pkg/errors"
	"github.com/interuss/dss/pkg/raftstore"
	"github.com/interuss/dss/pkg/raftstore/consensus"
	"github.com/interuss/stacktrace"
	"go.uber.org/zap"
)

const storeID = "aux_"

const (
	saveOwnMetadata raftstore.RequestType = "saveOwnMetadata"
	getDSSMetadata  raftstore.RequestType = "getDSSMetadata"
	recordHeartbeat raftstore.RequestType = "recordHeartbeat"
)

type saveOwnMetadataPayload struct {
	Locality       string `json:"locality"`
	PublicEndpoint string `json:"public_endpoint"`
}

// repo is a full implementation of aux_.repos.Repository for Raft-based storage.
type repo struct {
	consensus *consensus.Consensus
	mem       repos.Repository
}

func Init(ctx context.Context, logger *zap.Logger) (*raftstore.Store[repos.Repository], error) {
	memStore, err := auxmemstore.Init(ctx, logger)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to initialize aux memstore")
	}

	mem, err := memStore.Interact(ctx)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to obtain aux memstore repository")
	}

	r := &repo{mem: mem}
	store, consensus, err := raftstore.Init(ctx, logger, storeID, r)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to initialize aux raftstore")
	}

	r.consensus = consensus
	return store, nil
}

func (r *repo) GetRepo() repos.Repository { return r }

func (r *repo) IsReadOnly(requestType raftstore.RequestType) bool {
	return requestType == getDSSMetadata
}

func (r *repo) GetSnapshot() ([]byte, error) {
	return nil, stacktrace.NewErrorWithCode(dsserr.NotImplemented, "not implemented yet")
}

func (r *repo) RestoreFromSnapshot([]byte) error {
	return stacktrace.NewErrorWithCode(dsserr.NotImplemented, "not implemented yet")
}

func (r *repo) Apply(ctx context.Context, proposal consensus.Proposal) (any, error) {
	switch raftstore.RequestType(proposal.RequestType) {
	case saveOwnMetadata:
		var p saveOwnMetadataPayload
		if err := json.Unmarshal(proposal.Value, &p); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", saveOwnMetadata)
		}

		return nil, r.mem.SaveOwnMetadata(ctx, p.Locality, p.PublicEndpoint)

	case getDSSMetadata:
		return r.mem.GetDSSMetadata(ctx)

	case recordHeartbeat:
		var hb auxmodels.Heartbeat
		if err := json.Unmarshal(proposal.Value, &hb); err != nil {
			return nil, stacktrace.Propagate(err, "failed to unmarshal %s payload", recordHeartbeat)
		}

		return nil, r.mem.RecordHeartbeat(ctx, hb)

	default:
		return nil, stacktrace.NewError("unknown request type: %q", proposal.RequestType)
	}
}
