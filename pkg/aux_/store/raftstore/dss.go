package raftstore

import (
	"context"
	"strconv"

	auxmodels "github.com/interuss/dss/pkg/aux_/models"
	"github.com/interuss/dss/pkg/raftstore/consensus"
	raftparams "github.com/interuss/dss/pkg/raftstore/params"
	"github.com/interuss/stacktrace"
)

func (r *repo) SaveOwnMetadata(ctx context.Context, locality string, publicEndpoint string) error {
	p, err := consensus.NewProposal(ctx, storeID, saveOwnMetadata, saveOwnMetadataPayload{
		Locality:       locality,
		PublicEndpoint: publicEndpoint,
	}, false, nil)
	if err != nil {
		return stacktrace.Propagate(err, "failed to build %s proposal", saveOwnMetadata)
	}
	_, err = r.consensus.ProposeValue(ctx, p)
	return stacktrace.Propagate(err, "failed to propose %s", saveOwnMetadata)
}

func (r *repo) GetDSSMetadata(ctx context.Context) ([]*auxmodels.DSSMetadata, error) {
	p, err := consensus.NewProposal(ctx, storeID, getDSSMetadata, nil, true, nil)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to build %s proposal", getDSSMetadata)
	}
	result, err := r.consensus.ProposeValue(ctx, p)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to propose %s", getDSSMetadata)
	}
	if result == nil {
		return nil, nil
	}
	return result.([]*auxmodels.DSSMetadata), nil
}

func (r *repo) RecordHeartbeat(ctx context.Context, heartbeat auxmodels.Heartbeat) error {
	p, err := consensus.NewProposal(ctx, storeID, recordHeartbeat, heartbeat, false, nil)
	if err != nil {
		return stacktrace.Propagate(err, "failed to build %s proposal", recordHeartbeat)
	}
	_, err = r.consensus.ProposeValue(ctx, p)
	return stacktrace.Propagate(err, "failed to propose %s", recordHeartbeat)
}

func (r *repo) GetDSSAirspaceRepresentationID(_ context.Context) (string, error) {
	return strconv.Itoa(int(raftparams.GetClusterID())), nil
}
