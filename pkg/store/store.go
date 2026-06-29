package store

import (
	"context"
	"io"

	"github.com/interuss/stacktrace"
)

type Action[R any] interface {
	IsReadOnly() bool
	RequestType() string
	Payload() any
	Run(ctx context.Context, r R) (any, error)
}

// store.Store is the generic means to access and interact with any type of data backing the DSS
// may ever use, by obtaining a means to perform R-specific (repo type) operations.
type Store[R any] interface {
	io.Closer
	// Obtain a Repo (repo type R) that doesn't need transactional guarantees (for instance,
	// read-only).
	Interact(context.Context) (R, error)
	// Transact attempts to apply action atomically
	Transact(ctx context.Context, action Action[R]) (any, error)
}

const (
	CodeRetryable = stacktrace.ErrorCode(1)
)
