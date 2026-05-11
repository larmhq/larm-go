package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// recordingTransport captures each request it sees and returns a configurable status.
type recordingTransport struct {
	mu       sync.Mutex
	requests []*http.Request
	status   int // 0 → 200
}

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	clone := req.Clone(req.Context())
	r.requests = append(r.requests, clone)

	status := r.status
	if status == 0 {
		status = 200
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
		Request:    req,
	}, nil
}

func (r *recordingTransport) lastRequest(t *testing.T) *http.Request {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.requests) == 0 {
		t.Fatal("no requests captured")
	}
	return r.requests[len(r.requests)-1]
}

func (r *recordingTransport) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.requests)
}

// newRecordingClient builds a client whose Transport is the recordingTransport,
// short-circuiting the RetryTransport so test calls don't sleep on retries.
func newRecordingClient(t *testing.T, rt *recordingTransport, opts ...Option) *ClientWithResponses {
	t.Helper()
	full := append([]Option{WithBaseTransport(rt)}, opts...)
	c, err := New("https://example.test/api/v1", full...)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNewSetsDefaultUserAgent(t *testing.T) {
	t.Parallel()
	rt := &recordingTransport{}
	c := newRecordingClient(t, rt)

	if _, err := c.ListMonitorsWithResponse(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	got := rt.lastRequest(t).Header.Get("User-Agent")
	want := "larm-go/" + Version
	if got != want {
		t.Errorf("User-Agent = %q, want %q", got, want)
	}
}

func TestNewAppendsCustomUserAgent(t *testing.T) {
	t.Parallel()
	rt := &recordingTransport{}
	c := newRecordingClient(t, rt, WithUserAgent("foo/1.0"))

	if _, err := c.ListMonitorsWithResponse(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	got := rt.lastRequest(t).Header.Get("User-Agent")
	want := "larm-go/" + Version + " foo/1.0"
	if got != want {
		t.Errorf("User-Agent = %q, want %q", got, want)
	}
}

func TestNewSetsAuthorizationFromToken(t *testing.T) {
	t.Parallel()
	rt := &recordingTransport{}
	c := newRecordingClient(t, rt, WithToken("abc"))

	if _, err := c.ListMonitorsWithResponse(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	got := rt.lastRequest(t).Header.Get("Authorization")
	if got != "Bearer abc" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer abc")
	}
}

func TestNewWithEmptyTokenOmitsAuthorization(t *testing.T) {
	t.Parallel()
	rt := &recordingTransport{}
	c := newRecordingClient(t, rt, WithToken(""))

	if _, err := c.ListMonitorsWithResponse(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	if got := rt.lastRequest(t).Header.Get("Authorization"); got != "" {
		t.Errorf("Authorization = %q, want empty", got)
	}
}

func TestNewWithoutTokenOmitsAuthorization(t *testing.T) {
	t.Parallel()
	rt := &recordingTransport{}
	c := newRecordingClient(t, rt)

	if _, err := c.ListMonitorsWithResponse(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	if got := rt.lastRequest(t).Header.Get("Authorization"); got != "" {
		t.Errorf("Authorization = %q, want empty", got)
	}
}

func TestNewSetsAuthorizationFromTokenSource(t *testing.T) {
	t.Parallel()
	rt := &recordingTransport{}
	c := newRecordingClient(t, rt, WithTokenSource(StaticToken("xyz")))

	if _, err := c.ListMonitorsWithResponse(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	got := rt.lastRequest(t).Header.Get("Authorization")
	if got != "Bearer xyz" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer xyz")
	}
}

func TestNewTokenSourceCalledPerRequest(t *testing.T) {
	t.Parallel()
	rt := &recordingTransport{}
	var called int
	c := newRecordingClient(t, rt, WithTokenSource(TokenSourceFunc(func(_ context.Context) (string, error) {
		called++
		return "tok", nil
	})))

	for i := 0; i < 3; i++ {
		if _, err := c.ListMonitorsWithResponse(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if called != 3 {
		t.Errorf("TokenSource calls = %d, want 3", called)
	}
}

func TestNewTokenSourceErrorPropagates(t *testing.T) {
	t.Parallel()
	rt := &recordingTransport{}
	sentinel := errors.New("token boom")
	c := newRecordingClient(t, rt, WithTokenSource(TokenSourceFunc(func(_ context.Context) (string, error) {
		return "", sentinel
	})))

	_, err := c.ListMonitorsWithResponse(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error from token source, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected wrapped sentinel, got %v", err)
	}
	if !strings.Contains(err.Error(), "larm-go: token source:") {
		t.Errorf("expected error prefix 'larm-go: token source:', got %q", err.Error())
	}
}

func TestNewTokenSourceContextPropagates(t *testing.T) {
	t.Parallel()
	rt := &recordingTransport{}
	type ctxKey struct{}
	var seen string
	c := newRecordingClient(t, rt, WithTokenSource(TokenSourceFunc(func(ctx context.Context) (string, error) {
		if v, ok := ctx.Value(ctxKey{}).(string); ok {
			seen = v
		}
		return "tok", nil
	})))

	ctx := context.WithValue(context.Background(), ctxKey{}, "marker")
	if _, err := c.ListMonitorsWithResponse(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if seen != "marker" {
		t.Errorf("TokenSource ctx value = %q, want %q", seen, "marker")
	}
}

func TestNewTokenSourceEmptyTokenOmitsAuthorization(t *testing.T) {
	t.Parallel()
	rt := &recordingTransport{}
	c := newRecordingClient(t, rt, WithTokenSource(TokenSourceFunc(func(_ context.Context) (string, error) {
		return "", nil
	})))

	if _, err := c.ListMonitorsWithResponse(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if got := rt.lastRequest(t).Header.Get("Authorization"); got != "" {
		t.Errorf("Authorization = %q, want empty", got)
	}
}

func TestNewWrapsBaseTransportInsideRetry(t *testing.T) {
	t.Parallel()
	rt := &recordingTransport{}
	c := newRecordingClient(t, rt, WithToken("t"))

	if _, err := c.ListMonitorsWithResponse(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if rt.count() != 1 {
		t.Errorf("base transport calls = %d, want 1", rt.count())
	}
	if got := rt.lastRequest(t).Header.Get("Authorization"); got != "Bearer t" {
		t.Errorf("Authorization not threaded through: %q", got)
	}
}

func TestNewRetriesConfigurable(t *testing.T) {
	t.Parallel()
	rt := &recordingTransport{status: 500}
	c := newRecordingClient(t, rt,
		WithRetries(2),
	)

	// Use a context that cancels backoff quickly so the test stays fast.
	// 2 retries × 1s backoff = 3s base; cancel after 5s as a safety bound.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _ = c.ListMonitorsWithResponse(ctx, nil)

	// Initial + 2 retries = 3 total attempts.
	if got := rt.count(); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
}

func TestNewWithRetriesZero(t *testing.T) {
	t.Parallel()
	rt := &recordingTransport{status: 500}
	c := newRecordingClient(t, rt, WithRetries(0))

	resp, err := c.ListMonitorsWithResponse(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode() != 500 {
		t.Errorf("status = %d, want 500", resp.StatusCode())
	}
	if got := rt.count(); got != 1 {
		t.Errorf("attempts = %d, want 1 (no retries)", got)
	}
}

func TestNewWithRetriesNegative(t *testing.T) {
	t.Parallel()
	rt := &recordingTransport{status: 500}
	c := newRecordingClient(t, rt, WithRetries(-1))

	resp, err := c.ListMonitorsWithResponse(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response (locked: no more nil/nil bug)")
	}
	if resp.StatusCode() != 500 {
		t.Errorf("status = %d, want 500", resp.StatusCode())
	}
	if got := rt.count(); got != 1 {
		t.Errorf("attempts = %d, want 1 (negative clamped to 0)", got)
	}
}

func TestNewBaseURLTrailingSlashTrimmed(t *testing.T) {
	t.Parallel()

	cases := []string{
		"https://example.test/api/v1",
		"https://example.test/api/v1/",
		"https://example.test/api/v1//",
	}
	paths := make([]string, 0, len(cases))
	for _, baseURL := range cases {
		rt := &recordingTransport{}
		c, err := New(baseURL, WithBaseTransport(rt))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.ListMonitorsWithResponse(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, rt.lastRequest(t).URL.String())
	}

	for i := 1; i < len(paths); i++ {
		if paths[i] != paths[0] {
			t.Errorf("URL with trailing slash differed: %q vs %q", paths[0], paths[i])
		}
	}
}

func TestNewWithTimeoutConfigurable(t *testing.T) {
	t.Parallel()

	// Slow transport: hangs longer than the configured timeout, then returns.
	slow := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		select {
		case <-time.After(500 * time.Millisecond):
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{}")), Request: req}, nil
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	})

	c, err := New("https://example.test/api/v1",
		WithBaseTransport(slow),
		WithTimeout(50*time.Millisecond),
	)
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	_, err = c.ListMonitorsWithResponse(context.Background(), nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if elapsed > 400*time.Millisecond {
		t.Errorf("timeout not honored, took %v", elapsed)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
