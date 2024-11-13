package datastore

import (
	"context"
	"fmt"
	"github.com/coreos/go-semver/semver"
	"github.com/interuss/stacktrace"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Datastore struct {
	Version *Version
	DB      Database
	Pool    *pgxpool.Pool
}

type Database interface {
	GetSchemaVersion(ctx context.Context, dbName string) (*semver.Version, error)
}

var UnknownVersion = &semver.Version{}

func NewDatastore(ctx context.Context, pool *pgxpool.Pool) (*Datastore, error) {
	version, err := fetchVersion(ctx, pool)
	if err != nil {
		return nil, err
	}

	if version.IsCockroachDB() {
		return &Datastore{Version: version, Pool: pool, DB: &Cockroach{Pool: pool}}, nil
	}
	//if version.IsYugabyte() {
	//return &Datastore{Version: version, Pool: pool, DB: &Yugabyte{Pool: pool}}, nil
	//}
	return nil, fmt.Errorf("%s is not implemented yet", version.dsName)
}

func Dial(ctx context.Context, connParams ConnectParameters) (*Datastore, error) {
	dsn, err := connParams.BuildDSN()
	if err != nil {
		return nil, stacktrace.Propagate(err, "Failed to create connection config for pgx")
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Failed to parse connection config for pgx")
	}

	if connParams.SSL.Mode == "enable" {
		config.ConnConfig.TLSConfig.ServerName = connParams.Host
	}
	config.MaxConns = int32(connParams.MaxOpenConns)
	config.MaxConnIdleTime = (time.Duration(connParams.MaxConnIdleSeconds) * time.Second)
	config.HealthCheckPeriod = (1 * time.Second)
	config.MinConns = 1

	dbPool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	ds, err := NewDatastore(ctx, dbPool)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Failed to connect to datastore")
	}
	return ds, nil
}

func fetchVersion(ctx context.Context, pool *pgxpool.Pool) (*Version, error) {
	const versionDbQuery = `
      SELECT version();
    `
	var fullVersion string
	err := pool.QueryRow(ctx, versionDbQuery).Scan(&fullVersion)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error querying datastore version")
	}

	return VersionFromString(fullVersion)
}
