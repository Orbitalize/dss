package actions

import (
	"context"
	"encoding/json"

	"github.com/golang/geo/s2"
	restapi "github.com/interuss/dss/pkg/api/scdv1"
	dsserr "github.com/interuss/dss/pkg/errors"
	dssmodels "github.com/interuss/dss/pkg/models"
	scdmodels "github.com/interuss/dss/pkg/scd/models"
	"github.com/interuss/dss/pkg/scd/repos"
	"github.com/interuss/stacktrace"
	"github.com/jackc/pgx/v5"
)

const DeleteConstraintRequestType = "deleteConstraintTransaction"

type DeleteConstraintAction struct {
	ID      dssmodels.ID
	Manager dssmodels.Manager
	OVN     scdmodels.OVN
}

// NewDeleteConstraintAction decodes a DeleteConstraintAction from its
// replicated payload bytes, as produced by Payload.
func NewDeleteConstraintAction(data []byte) (*DeleteConstraintAction, error) {
	var action DeleteConstraintAction
	if err := json.Unmarshal(data, &action); err != nil {
		return nil, stacktrace.Propagate(err, "failed to unmarshal delete constraint action")
	}
	return &action, nil
}

func (a *DeleteConstraintAction) RequestType() string { return DeleteConstraintRequestType }

func (a *DeleteConstraintAction) Payload() any { return a }

func (a *DeleteConstraintAction) Run(ctx context.Context, r repos.Repository) (any, error) {
	old, err := r.GetConstraint(ctx, a.ID)
	switch {
	case err == pgx.ErrNoRows:
		return nil, stacktrace.NewErrorWithCode(dsserr.NotFound, "Constraint %s not found", a.ID)
	case err != nil:
		return nil, stacktrace.Propagate(err, "Unable to get Constraint from repo")
	case old == nil:
		return nil, stacktrace.NewErrorWithCode(dsserr.NotFound, "Constraint %s not found", a.ID)
	case old.Manager != a.Manager:
		return nil, stacktrace.NewErrorWithCode(dsserr.PermissionDenied,
			"Constraint owned by %s, but %s attempted to delete", old.Manager, a.Manager)
	case old.OVN != a.OVN:
		return nil, stacktrace.NewErrorWithCode(dsserr.VersionMismatch,
			"Current version is %s but client specified version %s", old.OVN, a.OVN)
	}

	if err := r.DeleteConstraint(ctx, a.ID); err != nil {
		return nil, stacktrace.Propagate(err, "Unable to delete Constraint from repo")
	}
	subs, err := r.IncrementNotificationIndicesForConstraints(ctx, &dssmodels.Volume4D{
		StartTime: old.StartTime,
		EndTime:   old.EndTime,
		SpatialVolume: &dssmodels.Volume3D{
			AltitudeHi: old.AltitudeUpper,
			AltitudeLo: old.AltitudeLower,
			Footprint: dssmodels.GeometryFunc(func() (s2.CellUnion, error) {
				return old.Cells, nil
			}),
		}})
	if err != nil {
		return nil, stacktrace.Propagate(err, "Unable to increment notification indices")
	}

	return &restapi.ChangeConstraintReferenceResponse{
		ConstraintReference: *old.ToRest(),
		Subscribers:         repos.MakeSubscribersToNotify(subs),
	}, nil
}

func (a *DeleteConstraintAction) IsReadOnly() bool { return false }
