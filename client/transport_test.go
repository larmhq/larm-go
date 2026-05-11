package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockResponse is a recipe for the response a mockTransport should produce on
// the Nth call. Storing recipes (not *http.Response) avoids tripping bodyclose
// on long-lived response literals.
type mockResponse struct {
	status     int
	retryAfter string // optional; sets Retry-After header on this response
	body       string // defaults to "{}"
}

type mockTransport struct {
	mu       sync.Mutex
	recipes  []mockResponse
	calls    int
	respHook func(*http.Response) // optional: called for each response before returning
}

func (m *mockTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx := m.calls
	if idx >= len(m.recipes) {
		idx = len(m.recipes) - 1
	}
	m.calls++

	r := m.recipes[idx]
	body := r.body
	if body == "" {
		body = "{}"
	}
	resp := &http.Response{
		StatusCode: r.status,
		Header:     http.Header{},
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
	}
	if r.retryAfter != "" {
		resp.Header.Set("Retry-After", r.retryAfter)
	}
	if m.respHook != nil {
		m.respHook(resp)
	}
	return resp, nil
}

func (m *mockTransport) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

// recipes is a small constructor helper that takes a list of status codes.
func recipes(statuses ...int) []mockResponse {
	out := make([]mockResponse, len(statuses))
	for i, s := range statuses {
		out[i] = mockResponse{status: s}
	}
	return out
}

// newReq builds a request with a background context.
func newReq(t *testing.T, method, url string, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

// drain closes the response body and discards any remaining bytes.
func drain(t *testing.T, resp *http.Response) {
	t.Helper()
	if resp != nil && resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
}

func TestNoRetryOn200(t *testing.T) {
	t.Parallel()
	mock := &mockTransport{recipes: recipes(200)}
	rt := &RetryTransport{Base: mock, MaxRetries: 3}

	req := newReq(t, http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer drain(t, resp)

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if mock.callCount() != 1 {
		t.Errorf("expected 1 call, got %d", mock.callCount())
	}
}

func TestNoRetryOn4xx(t *testing.T) {
	t.Parallel()
	mock := &mockTransport{recipes: recipes(404)}
	rt := &RetryTransport{Base: mock, MaxRetries: 3}

	req := newReq(t, http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer drain(t, resp)

	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	if mock.callCount() != 1 {
		t.Errorf("expected 1 call, got %d", mock.callCount())
	}
}

func TestRetryOn5xx(t *testing.T) {
	t.Parallel()
	mock := &mockTransport{recipes: recipes(500, 500, 200)}
	rt := &RetryTransport{Base: mock, MaxRetries: 3}

	req := newReq(t, http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer drain(t, resp)

	if resp.StatusCode != 200 {
		t.Errorf("expected 200 after retries, got %d", resp.StatusCode)
	}
	if mock.callCount() != 3 {
		t.Errorf("expected 3 calls, got %d", mock.callCount())
	}
}

func TestRetryOn429(t *testing.T) {
	t.Parallel()
	mock := &mockTransport{
		recipes: []mockResponse{
			{status: 429, retryAfter: "0"},
			{status: 200},
		},
	}
	rt := &RetryTransport{Base: mock, MaxRetries: 3}

	req := newReq(t, http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer drain(t, resp)

	if resp.StatusCode != 200 {
		t.Errorf("expected 200 after retry, got %d", resp.StatusCode)
	}
	if mock.callCount() != 2 {
		t.Errorf("expected 2 calls, got %d", mock.callCount())
	}
}

func TestRequestBodyPreservedOnRetry(t *testing.T) {
	t.Parallel()
	var bodies []string
	mock := &mockTransport{recipes: recipes(500, 200)}

	captureTransport := &bodyCapture{
		base:   mock,
		bodies: &bodies,
	}

	rt := &RetryTransport{Base: captureTransport, MaxRetries: 3}

	body := `{"name":"test"}`
	req := newReq(t, http.MethodPost, "http://example.com", bytes.NewReader([]byte(body)))
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer drain(t, resp)

	if len(bodies) != 2 {
		t.Fatalf("expected 2 attempts, got %d", len(bodies))
	}
	if bodies[0] != body {
		t.Errorf("first attempt body: %q, want %q", bodies[0], body)
	}
	if bodies[1] != body {
		t.Errorf("second attempt body: %q, want %q", bodies[1], body)
	}
}

type bodyCapture struct {
	base   http.RoundTripper
	bodies *[]string
}

func (b *bodyCapture) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		data, _ := io.ReadAll(req.Body)
		*b.bodies = append(*b.bodies, string(data))
		req.Body = io.NopCloser(bytes.NewReader(data))
	}
	return b.base.RoundTrip(req)
}

func TestRetryTransportNilBaseUsesDefault(t *testing.T) {
	t.Parallel()
	// Setting Base = nil should fall back to http.DefaultTransport. We verify
	// the nil-base branch doesn't panic by issuing a request with an already-
	// canceled context; http.DefaultTransport returns the ctx error promptly.
	rt := &RetryTransport{Base: nil, MaxRetries: 0}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:1/", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, gotErr := rt.RoundTrip(req)
	defer drain(t, resp)

	if gotErr == nil {
		t.Fatal("expected error from canceled ctx, got nil")
	}
	if !errors.Is(gotErr, context.Canceled) && !strings.Contains(gotErr.Error(), "context canceled") {
		t.Logf("error: %v (acceptable as long as RoundTrip didn't panic)", gotErr)
	}
}

func TestRetryTransportZeroMaxNoRetries(t *testing.T) {
	t.Parallel()
	mock := &mockTransport{recipes: recipes(500)}
	rt := &RetryTransport{Base: mock, MaxRetries: 0}

	req := newReq(t, http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer drain(t, resp)

	if resp.StatusCode != 500 {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
	if mock.callCount() != 1 {
		t.Errorf("expected 1 attempt (no retries), got %d", mock.callCount())
	}
}

func TestRetryTransportNegativeMaxClampedToZero(t *testing.T) {
	t.Parallel()
	mock := &mockTransport{recipes: recipes(500)}
	rt := &RetryTransport{Base: mock, MaxRetries: -5}

	req := newReq(t, http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer drain(t, resp)

	if resp.StatusCode != 500 {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
	if mock.callCount() != 1 {
		t.Errorf("expected 1 attempt (negative clamped to 0), got %d", mock.callCount())
	}
}

func TestRetryTransportContextCancelDuringBackoff(t *testing.T) {
	t.Parallel()
	mock := &mockTransport{recipes: recipes(500, 200)}
	rt := &RetryTransport{Base: mock, MaxRetries: 3}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	resp, gotErr := rt.RoundTrip(req)
	defer drain(t, resp)

	if !errors.Is(gotErr, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", gotErr)
	}
	if mock.callCount() != 1 {
		t.Errorf("expected 1 attempt before cancel, got %d", mock.callCount())
	}
}

func TestRetryTransportInvalidRetryAfterFallsBack(t *testing.T) {
	t.Parallel()
	mock := &mockTransport{
		recipes: []mockResponse{
			{status: 429, retryAfter: "not-a-number"},
			{status: 200},
		},
	}
	rt := &RetryTransport{Base: mock, MaxRetries: 3}

	start := time.Now()
	req := newReq(t, http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	defer drain(t, resp)

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	// Equal jitter at attempt=1: backoff range [0.5s, 1.0s]. Lower bound 400ms
	// allows for small scheduler slop.
	if elapsed < 400*time.Millisecond {
		t.Errorf("expected backoff fallback to apply (>=400ms), got %v", elapsed)
	}
}

func TestRetryTransportConcurrentSafe(t *testing.T) {
	t.Parallel()
	rt := &RetryTransport{
		Base: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewReader([]byte("{}"))),
			}, nil
		}),
		MaxRetries: 3,
	}

	const goroutines = 10
	const perG = 5
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perG; j++ {
				req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
				if err != nil {
					t.Errorf("NewRequestWithContext: %v", err)
					return
				}
				resp, err := rt.RoundTrip(req)
				if err != nil {
					t.Errorf("RoundTrip: %v", err)
					return
				}
				if resp != nil && resp.Body != nil {
					_ = resp.Body.Close()
				}
			}
		}()
	}
	wg.Wait()
}

func TestRetryTransportExhaustedReturnsLastResponse(t *testing.T) {
	t.Parallel()
	mock := &mockTransport{recipes: recipes(500, 500, 500, 500)}
	rt := &RetryTransport{Base: mock, MaxRetries: 3}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("expected nil error on exhausted retries, got %v", err)
	}
	defer drain(t, resp)

	if resp == nil || resp.StatusCode != 500 {
		t.Errorf("expected final 500 response, got %+v", resp)
	}
	if mock.callCount() != 4 {
		t.Errorf("expected 4 attempts, got %d", mock.callCount())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
