package raftstore

import (
	"context"
	"encoding/json"
	"time"

	"github.com/golang/geo/s2"
	dsserr "github.com/interuss/dss/pkg/errors"
	"github.com/interuss/dss/pkg/geo"
	dssmodels "github.com/interuss/dss/pkg/models"
	"github.com/interuss/dss/pkg/raftstore/consensus"
	ridmodels "github.com/interuss/dss/pkg/rid/models"
	"github.com/interuss/stacktrace"
)

type getISAPayload struct {
	ID        dssmodels.ID `json:"id"`
	ForUpdate bool         `json:"for_update"`
}

type searchISAsPayload struct {
	Cells    s2.CellUnion `json:"cells"`
	Earliest *time.Time   `json:"earliest,omitempty"`
	Latest   *time.Time   `json:"latest,omitempty"`
}

type listExpiredISAsPayload struct {
	Writer    string    `json:"writer"`
	Threshold time.Time `json:"threshold"`
}

func (r *repo) GetISA(ctx context.Context, id dssmodels.ID, forUpdate bool) (*ridmodels.IdentificationServiceArea, error) {
	result, err := r.propose(ctx, getISA, &getISAPayload{ID: id, ForUpdate: forUpdate})
	if err != nil {
		return nil, err
	}

	isa, ok := result.(*ridmodels.IdentificationServiceArea)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return isa, nil
}

func (r *repo) DeleteISA(ctx context.Context, isa *ridmodels.IdentificationServiceArea) (*ridmodels.IdentificationServiceArea, error) {
	result, err := r.propose(ctx, deleteISA, isa)
	if err != nil {
		return nil, err
	}

	isa, ok := result.(*ridmodels.IdentificationServiceArea)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return isa, nil
}

func (r *repo) InsertISA(ctx context.Context, isa *ridmodels.IdentificationServiceArea) (*ridmodels.IdentificationServiceArea, error) {
	result, err := r.propose(ctx, insertISA, isa)
	if err != nil {
		return nil, err
	}

	isa, ok := result.(*ridmodels.IdentificationServiceArea)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return isa, nil
}

func (r *repo) UpdateISA(ctx context.Context, isa *ridmodels.IdentificationServiceArea) (*ridmodels.IdentificationServiceArea, error) {
	result, err := r.propose(ctx, updateISA, isa)
	if err != nil {
		return nil, err
	}

	isa, ok := result.(*ridmodels.IdentificationServiceArea)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return isa, nil
}

func (r *repo) SearchISAs(ctx context.Context, cells s2.CellUnion, earliest *time.Time, latest *time.Time) ([]*ridmodels.IdentificationServiceArea, error) {
	result, err := r.propose(ctx, searchISAs, &searchISAsPayload{Cells: cells, Earliest: earliest, Latest: latest})
	if err != nil {
		return nil, err
	}

	isa, ok := result.([]*ridmodels.IdentificationServiceArea)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return isa, nil
}

func (r *repo) ListExpiredISAs(ctx context.Context, writer string, threshold time.Time) ([]*ridmodels.IdentificationServiceArea, error) {
	result, err := r.propose(ctx, listExpiredISAs, &listExpiredISAsPayload{Writer: writer, Threshold: threshold})
	if err != nil {
		return nil, err
	}

	isa, ok := result.([]*ridmodels.IdentificationServiceArea)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return isa, nil
}

func (r *repo) CountISAs(ctx context.Context) (int64, error) {
	result, err := r.propose(ctx, countISAs, nil)
	if err != nil {
		return 0, err
	}

	count, ok := result.(int64)
	if !ok {
		return 0, stacktrace.NewError("invalid result type: %T", result)
	}

	return count, nil
}

type DeleteISATransactionPayload struct {
	ID      dssmodels.ID       `json:"id"`
	Owner   dssmodels.Owner    `json:"owner"`
	Version *dssmodels.Version `json:"version,omitempty"`
}

type ISATransactionResult struct {
	Ret  *ridmodels.IdentificationServiceArea
	Subs []*ridmodels.Subscription
}

func (r *repo) deleteISATransactionApplier(ctx context.Context, proposal consensus.Proposal) (*ISATransactionResult, error) {
	var payload DeleteISATransactionPayload
	err := json.Unmarshal(proposal.Value, &payload)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to unmarshal payload")
	}

	old, err := r.memRepo.GetISA(ctx, payload.ID, true)
	switch {
	case err != nil:
		return nil, stacktrace.Propagate(err, "Error getting ISA")
	case old == nil:
		return nil, stacktrace.NewErrorWithCode(dsserr.NotFound, "ISA %s not found", payload.ID.String())
	case !payload.Version.Matches(old.Version):
		return nil, stacktrace.NewErrorWithCode(dsserr.VersionMismatch,
			"ISA currently at version %s but client specified %s", old.Version, payload.Version)
	case old.Owner != payload.Owner:
		return nil, stacktrace.NewErrorWithCode(dsserr.PermissionDenied,
			"ISA owned by %s, but %s attempted to delete", old.Owner, payload.Owner)
	}

	checkpoint := r.memStore.Checkpoint()
	ret, err := r.memRepo.DeleteISA(ctx, old)
	if err != nil {
		restoreErr := r.memStore.Restore(checkpoint)
		if restoreErr != nil {
			return nil, stacktrace.Propagate(err, "Error restoring store")
		}

		return nil, stacktrace.Propagate(err, "Error deleting ISA")
	}

	subs, err := r.memRepo.UpdateNotificationIdxsInCells(ctx, old.Cells)
	if err != nil {
		restoreErr := r.memStore.Restore(checkpoint)
		if restoreErr != nil {
			return nil, stacktrace.Propagate(err, "Error restoring store")
		}

		return nil, stacktrace.Propagate(err, "Error updating notification indices")
	}

	return &ISATransactionResult{Ret: ret, Subs: subs}, nil
}

func (r *repo) insertISATransactionApplier(ctx context.Context, proposal consensus.Proposal) (*ISATransactionResult, error) {
	var isa *ridmodels.IdentificationServiceArea
	err := json.Unmarshal(proposal.Value, &isa)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to unmarshal payload")
	}

	old, err := r.memRepo.GetISA(ctx, isa.ID, false)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error getting ISA")
	}
	if old != nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.AlreadyExists, "ISA %s already exists", isa.ID)
	}

	checkpoint := r.memStore.Checkpoint()
	subs, err := r.memRepo.UpdateNotificationIdxsInCells(ctx, isa.Cells)
	if err != nil {
		restoreErr := r.memStore.Restore(checkpoint)
		if restoreErr != nil {
			return nil, stacktrace.Propagate(err, "Error restoring store")
		}

		return nil, stacktrace.Propagate(err, "Error updating notification indices")
	}

	ret, err := r.memRepo.InsertISA(ctx, isa)
	if err != nil {
		restoreErr := r.memStore.Restore(checkpoint)
		if restoreErr != nil {
			return nil, stacktrace.Propagate(err, "Error restoring store")
		}

		return nil, stacktrace.Propagate(err, "Error inserting ISA")
	}

	return &ISATransactionResult{Ret: ret, Subs: subs}, nil
}

func (r *repo) updateISATransactionApplier(ctx context.Context, proposal consensus.Proposal) (*ISATransactionResult, error) {
	var isa *ridmodels.IdentificationServiceArea
	err := json.Unmarshal(proposal.Value, &isa)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to unmarshal payload")
	}

	old, err := r.memRepo.GetISA(ctx, isa.ID, true)
	switch {
	case err != nil:
		return nil, stacktrace.Propagate(err, "Error getting ISA")
	case old == nil:
		return nil, stacktrace.NewErrorWithCode(dsserr.NotFound, "ISA %s not found", isa.ID)
	case old.Owner != isa.Owner:
		return nil, stacktrace.NewErrorWithCode(dsserr.PermissionDenied,
			"ISA owned by %s, but %s attempted to update", old.Owner, isa.Owner)
	case !old.Version.Matches(isa.Version):
		return nil, stacktrace.NewErrorWithCode(dsserr.VersionMismatch,
			"ISA currently at version %s but client specified %s", old.Version, isa.Version)
	}

	if err := isa.AdjustTimeRange(proposal.Timestamp, old); err != nil {
		return nil, stacktrace.Propagate(err, "Error adjusting time range")
	}

	checkpoint := r.memStore.Checkpoint()
	ret, err := r.memRepo.UpdateISA(ctx, isa)
	if err != nil {
		restoreErr := r.memStore.Restore(checkpoint)
		if restoreErr != nil {
			return nil, stacktrace.Propagate(err, "Error restoring store")
		}

		return nil, stacktrace.Propagate(err, "Error updating ISA")
	}

	cells := s2.CellUnionFromUnion(old.Cells, isa.Cells)
	geo.Levelify(&cells)
	subs, err := r.memRepo.UpdateNotificationIdxsInCells(ctx, cells)
	if err != nil {
		restoreErr := r.memStore.Restore(checkpoint)
		if restoreErr != nil {
			return nil, stacktrace.Propagate(err, "Error restoring store")
		}

		return nil, stacktrace.Propagate(err, "Error updating notification indices")
	}

	return &ISATransactionResult{Ret: ret, Subs: subs}, nil
}
