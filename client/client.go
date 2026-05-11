package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// defaultTimeout is the per-request timeout when WithTimeout is not provided.
const defaultTimeout = 30 * time.Second

// Option configures a Client created by New.
type Option func(*config)

type config struct {
	tokenSource   TokenSource
	userAgent     string
	maxRetries    int
	timeout       time.Duration
	baseTransport http.RoundTripper
}

// WithToken sets a static bearer token used for the Authorization header.
//
// An empty token is treated as "no authentication" — the Authorization header
// is omitted entirely. For credentials that need to be refreshed, use
// WithTokenSource instead.
func WithToken(token string) Option {
	return func(c *config) { c.tokenSource = StaticToken(token) }
}

// WithTokenSource sets a TokenSource used to mint the bearer token for each request.
//
// The TokenSource is consulted on every API call, so implementations should
// cache short-lived tokens to avoid latency. A TokenSource that returns an
// empty string (with nil error) results in the Authorization header being
// omitted; an error from the TokenSource fails the request.
func WithTokenSource(ts TokenSource) Option {
	return func(c *config) { c.tokenSource = ts }
}

// WithRetries overrides the default retry count for transient HTTP failures.
// The default is 3. Setting n=0 disables retries (single attempt). Negative
// values are clamped to 0. Retries fire on 429 and 5xx; Retry-After is honored
// on 429.
func WithRetries(n int) Option {
	return func(c *config) { c.maxRetries = n }
}

// WithUserAgent appends a token to the default User-Agent. The final header
// is "larm-go/<version> <ua>" — useful for callers that want to identify
// themselves alongside the SDK (e.g. "terraform-provider-larm/0.1").
func WithUserAgent(ua string) Option {
	return func(c *config) { c.userAgent = ua }
}

// WithBaseTransport sets the http.RoundTripper that the SDK's RetryTransport
// wraps. Use this to layer middleware (logging, tracing, custom TLS) below
// the retry logic. If nil, http.DefaultTransport is used.
//
// To replace the entire HTTP client (skipping retries and our request editor),
// use NewClientWithResponses directly with the generated WithHTTPClient.
func WithBaseTransport(rt http.RoundTripper) Option {
	return func(c *config) { c.baseTransport = rt }
}

// WithTimeout overrides the default per-request HTTP timeout (30 seconds).
// Per-call cancellation should still be done via context.Context passed to
// the typed methods.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// New constructs a typed Larm API client.
//
// baseURL must include the API version path, e.g. "https://app.larm.dev/api/v1".
// Trailing slashes are trimmed. Without a token option, requests are sent
// unauthenticated and will receive 401 from the server.
func New(baseURL string, opts ...Option) (*ClientWithResponses, error) {
	cfg := &config{
		maxRetries: defaultMaxRetries,
		timeout:    defaultTimeout,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	httpClient := &http.Client{
		Timeout: cfg.timeout,
		Transport: &RetryTransport{
			Base:       cfg.baseTransport,
			MaxRetries: cfg.maxRetries,
		},
	}

	ua := "larm-go/" + Version
	if cfg.userAgent != "" {
		ua = ua + " " + cfg.userAgent
	}

	editor := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("User-Agent", ua)
		if cfg.tokenSource != nil {
			token, err := cfg.tokenSource.Token(ctx)
			if err != nil {
				return fmt.Errorf("larm-go: token source: %w", err)
			}
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}
		}
		return nil
	}

	return NewClientWithResponses(
		strings.TrimRight(baseURL, "/"),
		WithHTTPClient(httpClient),
		WithRequestEditorFn(editor),
	)
}
