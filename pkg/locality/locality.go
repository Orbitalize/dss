package locality

import (
	"context"

	"github.com/interuss/stacktrace"
)

type localityKey struct{}

// RequestLocalityFromContext returns the locality of the DSS instance that handled the request,
// or an error if the value is not present. The locality is set by the handler when a query is
// received then (on the receiver side) by the Raftstore when the query is applied, mirroring how
// pkg/timestamp threads the request timestamp, so that data depending on which DSS instance
// processed a request (e.g. Subscription.Writer) is replicated deterministically rather than
// re-derived locally by whichever node happens to be applying the entry.
func RequestLocalityFromContext(ctx context.Context) (string, error) {
	locality, ok := ctx.Value(localityKey{}).(string)
	if !ok || locality == "" {
		return "", stacktrace.NewError("locality not found in context")
	}
	return locality, nil
}

// WithRequestLocality returns a new context carrying the given locality.
func WithRequestLocality(ctx context.Context, locality string) context.Context {
	return context.WithValue(ctx, localityKey{}, locality)
}
