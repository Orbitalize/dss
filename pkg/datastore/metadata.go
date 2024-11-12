package datastore

import (
	"fmt"
	"github.com/coreos/go-semver/semver"
)

const COCKROACHDB = "cockroach"
const YUGABYTE = "yugabyte"

type Metadata struct {
	version *semver.Version
	dbType  string
}

func (m *Metadata) Version() *semver.Version {
	return m.version
}

func (m *Metadata) DbType() string {
	return m.dbType
}

func NewMetadata(version *semver.Version, dbType string) *Metadata {
	return &Metadata{version, dbType}
}

func (m *Metadata) IsCockroachDB() bool {
	if m.dbType == COCKROACHDB {
		return true
	}
	return false
}

func (m *Metadata) IsYugabyte() bool {
	if m.dbType == YUGABYTE {
		return true
	}
	return false
}

func (m *Metadata) String() string {
	return fmt.Sprintf("%s@%s", m.dbType, m.version.String())
}
