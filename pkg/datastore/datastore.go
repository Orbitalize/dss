package datastore

import (
	"context"
	"github.com/coreos/go-semver/semver"
)

var UnknownVersion = &semver.Version{}

type Datastore interface {
	Dial(ctx context.Context, connParams ConnectParameters) (*Datastore, error)
	GetVersion(ctx context.Context, dbName string) (*semver.Version, error)
	GetServerVersion() (*Metadata, error)
}
