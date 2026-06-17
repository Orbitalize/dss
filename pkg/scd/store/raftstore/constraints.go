package raftstore

import (
	"context"

	dssmodels "github.com/interuss/dss/pkg/models"
	scdmodels "github.com/interuss/dss/pkg/scd/models"
	"github.com/interuss/stacktrace"
)

func (r *repo) SearchConstraints(ctx context.Context, v4d *dssmodels.Volume4D) ([]*scdmodels.Constraint, error) {
	result, err := r.propose(ctx, searchConstraints, v4d)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to propose searchConstraints")
	}

	constraints, ok := result.([]*scdmodels.Constraint)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return constraints, nil
}

func (r *repo) GetConstraint(ctx context.Context, id dssmodels.ID) (*scdmodels.Constraint, error) {
	result, err := r.propose(ctx, getConstraint, id)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to propose getConstraint")
	}

	constraint, ok := result.(*scdmodels.Constraint)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return constraint, nil
}

func (r *repo) UpsertConstraint(ctx context.Context, constraint *scdmodels.Constraint) (*scdmodels.Constraint, error) {
	result, err := r.propose(ctx, upsertConstraint, constraint)
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to propose upsertConstraint")
	}

	constraint, ok := result.(*scdmodels.Constraint)
	if !ok {
		return nil, stacktrace.NewError("invalid result type: %T", result)
	}

	return constraint, nil
}

func (r *repo) DeleteConstraint(ctx context.Context, id dssmodels.ID) error {
	_, err := r.propose(ctx, deleteConstraint, id)
	return err
}

func (r *repo) CountConstraints(ctx context.Context) (int64, error) {
	result, err := r.propose(ctx, countConstraints, nil)
	if err != nil {
		return 0, stacktrace.Propagate(err, "failed to propose countConstraints")
	}

	count, ok := result.(int64)
	if !ok {
		return 0, stacktrace.NewError("invalid result type: %T", result)
	}

	return count, nil
}
