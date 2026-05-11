//go:build integration

package client_test

import (
	"context"
	"os"
	"testing"

	"github.com/larmhq/larm-go/client"
)

func TestIntegrationListMonitors(t *testing.T) {
	token := os.Getenv("LARM_API_KEY")
	if token == "" {
		t.Skip("LARM_API_KEY not set; skipping integration test")
	}

	baseURL := os.Getenv("LARM_API_URL")
	if baseURL == "" {
		baseURL = "https://app.larm.dev/api/v1"
	}

	c, err := client.New(baseURL, client.WithToken(token))
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}

	resp, err := c.ListMonitorsWithResponse(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListMonitors: %v", err)
	}
	if resp.StatusCode() != 200 {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}
