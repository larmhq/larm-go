package client

import (
	"context"
	"errors"
	"testing"
)

func TestStaticTokenReturnsValue(t *testing.T) {
	t.Parallel()

	tok, err := StaticToken("hello").Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if tok != "hello" {
		t.Errorf("got %q, want %q", tok, "hello")
	}
}

func TestStaticTokenIgnoresContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tok, err := StaticToken("hello").Token(ctx)
	if err != nil {
		t.Fatalf("expected no error from canceled ctx, got %v", err)
	}
	if tok != "hello" {
		t.Errorf("got %q, want %q", tok, "hello")
	}
}

func TestTokenSourceFuncCallsFunction(t *testing.T) {
	t.Parallel()

	var called int
	ts := TokenSourceFunc(func(_ context.Context) (string, error) {
		called++
		return "fn-token", nil
	})

	tok, err := ts.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if tok != "fn-token" {
		t.Errorf("got %q, want %q", tok, "fn-token")
	}
	if called != 1 {
		t.Errorf("expected 1 call, got %d", called)
	}
}

func TestTokenSourceFuncPropagatesContext(t *testing.T) {
	t.Parallel()

	type ctxKey struct{}
	wantCtx := context.WithValue(context.Background(), ctxKey{}, "marker")

	ts := TokenSourceFunc(func(ctx context.Context) (string, error) {
		if v, _ := ctx.Value(ctxKey{}).(string); v != "marker" {
			return "", errors.New("ctx not propagated")
		}
		return "ok", nil
	})

	if _, err := ts.Token(wantCtx); err != nil {
		t.Fatal(err)
	}
}
