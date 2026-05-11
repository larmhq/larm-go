package client

import (
	"bytes"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// defaultMaxRetries is the retry count used by client.New when WithRetries is
// not provided. It is intentionally not the zero value, so direct construction
// of RetryTransport via &RetryTransport{} yields zero retries (single attempt).
const defaultMaxRetries = 3

// RetryTransport wraps an http.RoundTripper with retry logic for transient failures.
//
// It retries on HTTP 429 and 5xx responses with exponential backoff. The
// Retry-After header is honored on 429 responses (interpreted as seconds; an
// unparseable value falls back to exponential backoff). Request bodies are
// buffered so they can be replayed on each attempt.
//
// If Base is nil, http.DefaultTransport is used.
//
// A single RetryTransport is safe for concurrent use by multiple goroutines and
// can serve multiple http.Client instances. Like all RoundTrippers, it may
// mutate the request — callers should not reuse a *http.Request across
// goroutines while it is in flight.
type RetryTransport struct {
	// Base is the underlying transport. If nil, http.DefaultTransport is used.
	Base http.RoundTripper

	// MaxRetries is the maximum number of retry attempts after the initial
	// request. MaxRetries=0 means a single attempt with no retries. Negative
	// values are treated as 0. When constructing RetryTransport directly, this
	// is the zero value; for typical usage via client.New, see WithRetries.
	MaxRetries int
}

// RoundTrip implements http.RoundTripper.
func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}

	maxRetries := t.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}

	// Buffer the request body so we can replay it on retries.
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := retryDelay(attempt, resp)
			timer := time.NewTimer(backoff)
			select {
			case <-timer.C:
			case <-req.Context().Done():
				timer.Stop()
				return nil, req.Context().Err()
			}
		}

		// Drain and close the previous response body to allow connection reuse.
		if resp != nil && resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}

		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			req.ContentLength = int64(len(bodyBytes))
		}

		resp, err = base.RoundTrip(req)
		if err != nil {
			continue
		}

		if !shouldRetry(resp.StatusCode) {
			return resp, nil
		}
	}

	return resp, err
}

func shouldRetry(statusCode int) bool {
	return statusCode == 429 || statusCode >= 500
}

func retryDelay(attempt int, resp *http.Response) time.Duration {
	if resp != nil && resp.StatusCode == 429 {
		if after := resp.Header.Get("Retry-After"); after != "" {
			if seconds, err := strconv.Atoi(after); err == nil {
				return time.Duration(seconds) * time.Second
			}
		}
	}

	// Equal jitter: 50%-100% of the deterministic exponential backoff. Spreads
	// retries across a window to avoid thundering herd when many clients hit a
	// transient outage simultaneously. Weak RNG is correct here — jitter only
	// needs unpredictability across clients, not cryptographic randomness.
	base := time.Duration(1<<(attempt-1)) * time.Second
	return base/2 + time.Duration(rand.Int64N(int64(base/2))) //nolint:gosec // G404: weak RNG is appropriate for backoff jitter
}
