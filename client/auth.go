package client

import "context"

// TokenSource produces a bearer token for authenticating API requests.
//
// Implementations may return a static token, fetch one from a credential store,
// or refresh an OAuth token on demand. The context is propagated from the API
// call, so implementations can honor cancellation and deadlines.
//
// Returning ("", nil) is valid and means "no Authorization header for this
// request" — callers can use this to disable auth on a per-request basis.
// Returning a non-nil error fails the request.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// StaticToken is a TokenSource that always returns the same bearer token.
type StaticToken string

// Token implements TokenSource.
func (s StaticToken) Token(_ context.Context) (string, error) {
	return string(s), nil
}

// TokenSourceFunc is an adapter that lets ordinary functions satisfy TokenSource.
// Mirrors the http.HandlerFunc pattern.
type TokenSourceFunc func(ctx context.Context) (string, error)

// Token implements TokenSource by calling f(ctx).
func (f TokenSourceFunc) Token(ctx context.Context) (string, error) {
	return f(ctx)
}

var (
	_ TokenSource = StaticToken("")
	_ TokenSource = TokenSourceFunc(nil)
)
