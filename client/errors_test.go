package client

import (
	"errors"
	"testing"
)

func TestParseAPIError(t *testing.T) {
	t.Parallel()
	body := `{"error":{"type":"not_found","message":"Monitor not found"}}`
	err := ParseAPIError(404, []byte(body))

	if err.Type != "not_found" {
		t.Errorf("expected type not_found, got: %s", err.Type)
	}
	if err.Message != "Monitor not found" {
		t.Errorf("expected message 'Monitor not found', got: %s", err.Message)
	}
	if err.StatusCode != 404 {
		t.Errorf("expected status 404, got: %d", err.StatusCode)
	}
}

func TestParseAPIErrorUnknownBody(t *testing.T) {
	t.Parallel()
	err := ParseAPIError(500, []byte("Internal Server Error"))

	if err.Type != "http_500" {
		t.Errorf("expected http_500, got: %s", err.Type)
	}
	if err.Message != "Internal Server Error" {
		t.Errorf("expected raw body as message, got: %s", err.Message)
	}
}

func TestParseAPIErrorEmptyBody(t *testing.T) {
	t.Parallel()
	err := ParseAPIError(500, nil)

	if err.Type != "http_500" {
		t.Errorf("expected http_500, got: %s", err.Type)
	}
	if err.Message != "" {
		t.Errorf("expected empty message, got: %q", err.Message)
	}
	if err.StatusCode != 500 {
		t.Errorf("expected status 500, got: %d", err.StatusCode)
	}
}

func TestParseAPIErrorMalformedJSON(t *testing.T) {
	t.Parallel()
	body := `{not valid json`
	err := ParseAPIError(502, []byte(body))

	if err.Type != "http_502" {
		t.Errorf("expected http_502 for malformed JSON, got: %s", err.Type)
	}
	if err.Message != body {
		t.Errorf("expected raw body as message, got: %s", err.Message)
	}
}

func TestAPIErrorAs(t *testing.T) {
	t.Parallel()
	err := error(ParseAPIError(401, []byte(`{"error":{"type":"unauthorized","message":"bad token"}}`)))

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected errors.As to extract *APIError")
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("expected status 401, got: %d", apiErr.StatusCode)
	}
}

func TestAPIErrorMessageFormat(t *testing.T) {
	t.Parallel()
	err := &APIError{Type: "validation", Message: "name required", StatusCode: 422}
	want := "validation: name required"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
