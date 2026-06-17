package raftstore

import (
	"context"
	"encoding/json"

	restapi "github.com/interuss/dss/pkg/api/scdv1"
	dsserr "github.com/interuss/dss/pkg/errors"
	dssmodels "github.com/interuss/dss/pkg/models"
	"github.com/interuss/dss/pkg/raftstore/consensus"
	scdmodels "github.com/interuss/dss/pkg/scd/models"
	"github.com/interuss/stacktrace"
	"github.com/jackc/pgx/v5"
)

func (r *repo) GetUssAvailability(ctx context.Context, id dssmodels.Manager) (*scdmodels.UssAvailabilityStatus, error) {
	result, err := r.propose(ctx, getUSSAvailability, id)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to propose getUSSAvailability")
	}

	status, ok := result.(*scdmodels.UssAvailabilityStatus)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return status, nil
}

func (r *repo) UpsertUssAvailability(ctx context.Context, ussa *scdmodels.UssAvailabilityStatus) (*scdmodels.UssAvailabilityStatus, error) {
	result, err := r.propose(ctx, upsertUSSAvailability, ussa)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to propose upsertUSSAvailability")
	}

	status, ok := result.(*scdmodels.UssAvailabilityStatus)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return status, nil
}

func (r *repo) getUSSAvailabilityTransactionApplier(ctx context.Context, proposal consensus.Proposal) (*restapi.UssAvailabilityStatusResponse, error) {
	var req *restapi.GetUssAvailabilityRequest
	if err := json.Unmarshal(proposal.Value, &req); err != nil {
		return nil, stacktrace.Propagate(err, "failed to unmarshal get USS availability request")
	}

	id := dssmodels.ManagerFromString(req.UssId)
	if id == "" {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "UssId not provided")
	}

	ussa, err := r.memRepo.GetUssAvailability(ctx, id)
	if err != nil && err != pgx.ErrNoRows {
		return nil, stacktrace.Propagate(err, "Could not get USS availability from repo")
	}
	if ussa == nil {
		return &restapi.UssAvailabilityStatusResponse{
			Status: restapi.UssAvailabilityStatus{
				Availability: restapi.UssAvailabilityState_Unknown,
				Uss:          id.String(),
			},
			Version: "",
		}, nil
	}
	return &restapi.UssAvailabilityStatusResponse{
		Status:  *ussa.ToRest(),
		Version: ussa.Version.String(),
	}, nil
}

func (r *repo) setUSSAvailabilityTransactionApplier(ctx context.Context, proposal consensus.Proposal) (*restapi.UssAvailabilityStatusResponse, error) {
	var req *restapi.SetUssAvailabilityRequest
	if err := json.Unmarshal(proposal.Value, &req); err != nil {
		return nil, stacktrace.Propagate(err, "failed to unmarshal set USS availability request")
	}

	if req.UssId == "" {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "ussID not provided")
	}

	availability, err := scdmodels.UssAvailabilityStateFromRest(req.Body.Availability)
	if err != nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.BadRequest, "Invalid availability state")
	}

	id := dssmodels.ManagerFromString(req.UssId)
	version := scdmodels.OVN(req.Body.OldVersion)

	old, err := r.memRepo.GetUssAvailability(ctx, id)
	if err != nil && err != pgx.ErrNoRows {
		return nil, stacktrace.Propagate(err, "Could not get USS availability from repo")
	}

	switch {
	case old == nil && !version.Empty():
		return nil, stacktrace.NewErrorWithCode(dsserr.AlreadyExists, "availability for USS %s already exists", id.String())
	case old != nil && old.Version != version:
		return nil, stacktrace.NewErrorWithCode(dsserr.VersionMismatch, "USS availability version %s is not current", version)
	}

	cp := r.memStore.Checkpoint()

	ussa, err := r.memRepo.UpsertUssAvailability(ctx, &scdmodels.UssAvailabilityStatus{
		Uss:          id,
		Availability: availability,
	})
	if err != nil {
		if restoreErr := r.memStore.Restore(cp); restoreErr != nil {
			return nil, stacktrace.Propagate(restoreErr, "Failed to restore store")
		}
		return nil, stacktrace.Propagate(err, "Could not upsert USS Availability into repo")
	}
	if ussa == nil {
		if restoreErr := r.memStore.Restore(cp); restoreErr != nil {
			return nil, stacktrace.Propagate(restoreErr, "Failed to restore store")
		}
		return nil, stacktrace.NewError("UpsertUssAvailability returned no USS availability for ID: %s", id)
	}
	return &restapi.UssAvailabilityStatusResponse{
		Status:  *ussa.ToRest(),
		Version: ussa.Version.String(),
	}, nil
}
