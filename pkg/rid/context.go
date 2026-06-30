package rid

import "context"

type allowHTTPBaseUrlsKey struct{}

// WithAllowHTTPBaseUrls returns a new context carrying the AllowHTTPBaseUrls deployment config.
// It is not part of any REST request, so (like locality.WithRequestLocality) it travels via
// context instead of being a field on a generated Action type.
func WithAllowHTTPBaseUrls(ctx context.Context, allow bool) context.Context {
	return context.WithValue(ctx, allowHTTPBaseUrlsKey{}, allow)
}

// AllowHTTPBaseUrlsFromContext returns the AllowHTTPBaseUrls deployment config carried by ctx.
func AllowHTTPBaseUrlsFromContext(ctx context.Context) bool {
	allow, _ := ctx.Value(allowHTTPBaseUrlsKey{}).(bool)
	return allow
}
