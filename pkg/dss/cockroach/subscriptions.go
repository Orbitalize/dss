package cockroach

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang/geo/s2"
	"github.com/lib/pq"
	"github.com/steeling/InterUSS-Platform/pkg/dss/models"
	"go.uber.org/multierr"
)

type crSubscriptionStore struct {
	*sql.DB
}

type queryable interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func (c *crSubscriptionStore) fetch(ctx context.Context, q queryable, query string, args ...interface{}) ([]*models.Subscription, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payload []*models.Subscription
	for rows.Next() {
		s := new(models.Subscription)

		err := rows.Scan(
			&s.ID,
			&s.Owner,
			&s.Url,
			&s.NotificationIndex,
			&s.StartTime,
			&s.EndTime,
		)
		if err != nil {
			return nil, err
		}
		payload = append(payload, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return payload, nil
}

func (c *crSubscriptionStore) fetchByID(ctx context.Context, q queryable, id string) (*models.Subscription, error) {
	// TODO(steeling) don't fetch by *
	const query = `
		SELECT * FROM
			subscriptions
		WHERE
			id = $1`
	s := new(models.Subscription)

	err := q.QueryRowContext(ctx, query, id).Scan(
		&s.ID,
		&s.Owner,
		&s.Url,
		&s.NotificationIndex,
		&s.StartTime,
		&s.EndTime,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (c *crSubscriptionStore) push(ctx context.Context, q queryable, s *models.Subscription) error {
	const (
		upsertQuery = `
		UPSERT INTO
		  subscriptions
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, transaction_timestamp())`
		subscriptionCellQuery = `
		UPSERT INTO
			cells_subscriptions
		VALUES
			($1, $2, $3, transaction_timestamp())
		`
	)
	if _, err := q.ExecContext(
		ctx,
		upsertQuery,
		s.ID,
		s.Owner,
		s.Url,
		s.NotificationIndex,
		s.StartTime,
		s.EndTime,
	); err != nil {
		return err
	}

	// TODO(steeling) we also need to delete any leftover cells.
	for _, cell := range s.Cells {
		if _, err := q.ExecContext(ctx, subscriptionCellQuery, cell, cell.Level(), s.ID); err != nil {
			return err
		}
	}
	return nil
}

// Get returns the subscription identified by "id".
func (c *crSubscriptionStore) Get(ctx context.Context, id string) (*models.Subscription, error) {
	return c.fetchByID(ctx, c.DB, id)
}

// Insert inserts subscription into the store and returns
// the resulting subscription including its ID.
func (c *crSubscriptionStore) Insert(ctx context.Context, s *models.Subscription) (*models.Subscription, error) {

	tx, err := c.Begin()
	if err != nil {
		return nil, err
	}
	_, err = c.fetchByID(ctx, tx, s.ID)
	if err == nil {
		// TODO(steeling) fix errors
		return nil, errors.New("already exists")
	}
	if err != sql.ErrNoRows {
		return nil, multierr.Combine(err, tx.Rollback())
	}

	if err := c.push(ctx, tx, s); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s, nil
}

// updatesSubscription updates the subscription  and returns
// the resulting subscription including its ID.
func (c *crSubscriptionStore) Update(ctx context.Context, s *models.Subscription, cells s2.CellUnion) (*models.Subscription, error) {
	const (
		getQuery = `
		SELECT * FROM
			subscriptions
		WHERE
			id = $1`
	)

	tx, err := c.Begin()
	if err != nil {
		return nil, err
	}

	old, err := c.fetchByID(ctx, tx, s.ID)
	switch {
	case err == sql.ErrNoRows: // Return a 404 here.
		return nil, multierr.Combine(err, tx.Rollback())
	case err != nil:
		return nil, multierr.Combine(err, tx.Rollback())
	case s.Version() != "" && s.Version() != old.Version():
		err := fmt.Errorf("version mismatch for subscription %s", s.ID)
		return nil, multierr.Combine(err, tx.Rollback())
	}

	if err := c.push(ctx, tx, old.Apply(s)); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s, nil
}

// DeleteSubscription deletes the subscription identified by "id" and
// returns the deleted subscription.
func (c *crSubscriptionStore) Delete(ctx context.Context, id, version string) (*models.Subscription, error) {
	const (
		query = `
		DELETE FROM
			subscriptions
		WHERE
			id = $1`
	)

	tx, err := c.Begin()
	if err != nil {
		return nil, err
	}

	// We fetch to know whether to return a concurrency error, or a not found error
	old, err := c.fetchByID(ctx, tx, id)
	switch {
	case err == sql.ErrNoRows: // Return a 404 here.
		return nil, multierr.Combine(err, tx.Rollback())
	case err != nil:
		return nil, multierr.Combine(err, tx.Rollback())
	case version != "" && version != old.Version():
		err := fmt.Errorf("version mismatch for subscription %s", id)
		return nil, multierr.Combine(err, tx.Rollback())
	}

	if _, err := tx.ExecContext(ctx, query, id); err != nil {
		return nil, multierr.Combine(err, tx.Rollback())
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return old, nil
}

// SearchSubscriptions returns all subscriptions in "cells".
func (c *crSubscriptionStore) Search(ctx context.Context, cells s2.CellUnion, owner string) ([]*models.Subscription, error) {
	const (
		query = `
			SELECT
				subscriptions.*
			FROM
				subscriptions
			LEFT JOIN 
				(SELECT DISTINCT cells_subscriptions.subscription_id FROM cells_subscriptions WHERE cells_subscriptions.cell_id = ANY($1))
			AS
				unique_subscription_ids
			ON
				subscriptions.id = unique_subscription_ids.subscription_id
			WHERE
				subscriptions.owner = $2`
	)

	if len(cells) == 0 {
		return nil, errors.New("missing cell IDs for query")
	}

	tx, err := c.Begin()
	if err != nil {
		return nil, err
	}

	subscriptions, err := c.fetch(ctx, tx, query, pq.Array(cells), owner)
	if err != nil {
		return nil, multierr.Combine(err, tx.Rollback())
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return subscriptions, nil
}
