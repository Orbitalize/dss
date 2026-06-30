package rid

import (
	"context"
	"time"

	"github.com/golang/geo/s2"
	dsserr "github.com/interuss/dss/pkg/errors"
	"github.com/interuss/dss/pkg/geo"
	dssmodels "github.com/interuss/dss/pkg/models"
	ridmodels "github.com/interuss/dss/pkg/rid/models"
	"github.com/interuss/dss/pkg/rid/repos"
	"github.com/interuss/dss/pkg/timestamp"
	"github.com/interuss/stacktrace"
)

// ISAResult bundles an IdentificationServiceArea together with the Subscriptions affected by the
// mutation that produced it, since an Action's Execute can only return a single value.
type ISAResult struct {
	ISA           *ridmodels.IdentificationServiceArea
	Subscriptions []*ridmodels.Subscription
}

type GetISAAction struct {
	ID dssmodels.ID
}

func (a *GetISAAction) RequestType() string { return "GetISA" }

func (a *GetISAAction) IsReadOnly() bool { return true }

func (a *GetISAAction) Payload() any { return a }

func (a *GetISAAction) Execute(ctx context.Context, r repos.Repository) (any, error) {
	return r.GetISA(ctx, a.ID, false)
}

type SearchISAsAction struct {
	Cells    s2.CellUnion
	Earliest *time.Time
	Latest   *time.Time
}

func (a *SearchISAsAction) RequestType() string { return "SearchISAs" }

func (a *SearchISAsAction) IsReadOnly() bool { return true }

func (a *SearchISAsAction) Payload() any { return a }

func (a *SearchISAsAction) Execute(ctx context.Context, r repos.Repository) (any, error) {
	earliest := a.Earliest
	now, err := timestamp.RequestTimestampFromContext(ctx)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Unable to get request timestamp")
	}
	if earliest == nil || earliest.Before(now) {
		earliest = &now
	}

	return r.SearchISAs(ctx, a.Cells, earliest, a.Latest)
}

type DeleteISAAction struct {
	ID      dssmodels.ID
	Owner   dssmodels.Owner
	Version *dssmodels.Version
}

func (a *DeleteISAAction) RequestType() string { return "DeleteISA" }

func (a *DeleteISAAction) IsReadOnly() bool { return false }

func (a *DeleteISAAction) Payload() any { return a }

func (a *DeleteISAAction) Execute(ctx context.Context, r repos.Repository) (any, error) {
	old, err := r.GetISA(ctx, a.ID, true)
	switch {
	case err != nil:
		return nil, stacktrace.Propagate(err, "Error getting ISA")
	case old == nil:
		return nil, stacktrace.NewErrorWithCode(dsserr.NotFound, "ISA %s not found", a.ID.String())
	case !a.Version.Matches(old.Version):
		return nil, stacktrace.NewErrorWithCode(dsserr.VersionMismatch,
			"ISA currently at version %s but client specified %s", old.Version, a.Version)
	case old.Owner != a.Owner:
		return nil, stacktrace.NewErrorWithCode(dsserr.PermissionDenied,
			"ISA owned by %s, but %s attempted to delete", old.Owner, a.Owner)
	}

	ret, err := r.DeleteISA(ctx, old)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error deleting ISA")
	}

	subs, err := r.UpdateNotificationIdxsInCells(ctx, old.Cells)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error updating notification indices")
	}

	return &ISAResult{ISA: ret, Subscriptions: subs}, nil
}

type InsertISAAction struct {
	ISA *ridmodels.IdentificationServiceArea
}

func (a *InsertISAAction) RequestType() string { return "InsertISA" }

func (a *InsertISAAction) IsReadOnly() bool { return false }

func (a *InsertISAAction) Payload() any { return a }

func (a *InsertISAAction) Execute(ctx context.Context, r repos.Repository) (any, error) {
	now, err := timestamp.RequestTimestampFromContext(ctx)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Unable to get request timestamp")
	}
	// Validate and perhaps correct StartTime and EndTime.
	if err := a.ISA.AdjustTimeRange(now, nil); err != nil {
		return nil, stacktrace.Propagate(err, "Error adjusting time range")
	}

	// ensure it doesn't exist yet
	old, err := r.GetISA(ctx, a.ISA.ID, false)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error getting ISA")
	}
	if old != nil {
		return nil, stacktrace.NewErrorWithCode(dsserr.AlreadyExists, "ISA %s already exists", a.ISA.ID)
	}

	// UpdateNotificationIdxsInCells is done in a Txn along with insert since
	// they are both modifying the db. Insert a susbcription alone does
	// not do this, so that does not need to use a txn (in subscription.go).
	subs, err := r.UpdateNotificationIdxsInCells(ctx, a.ISA.Cells)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error updating notification indices")
	}
	ret, err := r.InsertISA(ctx, a.ISA)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error inserting ISA")
	}

	return &ISAResult{ISA: ret, Subscriptions: subs}, nil
}

type UpdateISAAction struct {
	ISA *ridmodels.IdentificationServiceArea
}

func (a *UpdateISAAction) RequestType() string { return "UpdateISA" }

func (a *UpdateISAAction) IsReadOnly() bool { return false }

func (a *UpdateISAAction) Payload() any { return a }

func (a *UpdateISAAction) Execute(ctx context.Context, r repos.Repository) (any, error) {
	old, err := r.GetISA(ctx, a.ISA.ID, true)
	switch {
	case err != nil:
		return nil, stacktrace.Propagate(err, "Error getting ISA")
	case old == nil:
		return nil, stacktrace.NewErrorWithCode(dsserr.NotFound, "ISA %s not found", a.ISA.ID)
	case old.Owner != a.ISA.Owner:
		return nil, stacktrace.NewErrorWithCode(dsserr.PermissionDenied,
			"ISA owned by %s, but %s attempted to modify", old.Owner, a.ISA.Owner)
	case !old.Version.Matches(a.ISA.Version):
		return nil, stacktrace.NewErrorWithCode(dsserr.VersionMismatch,
			"ISA currently at version %s but client specified %s", old.Version, a.ISA.Version)
	}

	now, err := timestamp.RequestTimestampFromContext(ctx)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Unable to get request timestamp")
	}
	// Validate and perhaps correct StartTime and EndTime.
	if err := a.ISA.AdjustTimeRange(now, old); err != nil {
		return nil, stacktrace.Propagate(err, "Error adjusting time range")
	}

	ret, err := r.UpdateISA(ctx, a.ISA)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error updating ISA")
	}

	// TODO steeling, we should change this to a Custom type, to obfuscate
	// some of these metrics and prevent us from doing the wrong thing.
	cells := s2.CellUnionFromUnion(old.Cells, a.ISA.Cells)
	geo.Levelify(&cells)
	// UpdateNotificationIdxsInCells is done in a Txn along with insert since
	// they are both modifying the db. Insert a susbcription alone does
	// not do this, so that does not need to use a txn (in subscription.go).
	subs, err := r.UpdateNotificationIdxsInCells(ctx, cells)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error updating notification indices")
	}

	return &ISAResult{ISA: ret, Subscriptions: subs}, nil
}
