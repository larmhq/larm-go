package client_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/larmhq/larm-go/client"
)

// Construct a client with a static API key and call a typed method.
func Example() {
	c, err := client.New(
		"https://app.larm.dev/api/v1",
		client.WithToken("larm_api_..."),
	)
	if err != nil {
		panic(err)
	}

	resp, err := c.ListMonitorsWithResponse(context.Background(), nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.StatusCode())
}

// Use a TokenSourceFunc to mint a fresh token per request, e.g. when wrapping
// an OAuth refresh flow.
func ExampleTokenSourceFunc() {
	refresher := client.TokenSourceFunc(func(_ context.Context) (string, error) {
		// In a real implementation, fetch or refresh the token here.
		return "fresh-token", nil
	})

	_, _ = client.New(
		"https://app.larm.dev/api/v1",
		client.WithTokenSource(refresher),
	)
}

// Inspect API errors with errors.As to act on the parsed envelope.
func ExampleAPIError() {
	c, _ := client.New("https://app.larm.dev/api/v1", client.WithToken("..."))

	id := uuid.MustParse("00000000-0000-0000-0000-000000000000")
	_, err := c.GetMonitorWithResponse(context.Background(), id)

	var apiErr *client.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
		fmt.Println("not found")
	}
}

// Layer custom middleware (logging, tracing, custom TLS) below the SDK's
// retry transport by passing a base http.RoundTripper.
func ExampleWithBaseTransport() {
	// Replace with otelhttp.NewTransport, a logging RoundTripper, etc.
	base := http.DefaultTransport

	_, _ = client.New(
		"https://app.larm.dev/api/v1",
		client.WithToken("larm_api_..."),
		client.WithBaseTransport(base),
	)
}
