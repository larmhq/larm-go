package client

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// APIError represents a structured error from the Larm API.
//
// The Larm API returns errors as a JSON envelope of the form
// `{"error": {"type": "...", "message": "...", "errors": {...}}}`, where
// `errors` holds field-level validation errors. ParseAPIError decodes that
// envelope; callers should use errors.As to extract.
type APIError struct {
	StatusCode int    `json:"-"`
	Type       string `json:"error"`
	Message    string `json:"message"`
	// Errors maps field names to their validation messages.
	Errors map[string][]string `json:"errors,omitempty"`
}

func (e *APIError) Error() string {
	parts := make([]string, 0, 1+len(e.Errors))
	if e.Message != "" {
		parts = append(parts, e.Message)
	}
	for _, field := range slices.Sorted(maps.Keys(e.Errors)) {
		parts = append(parts, fmt.Sprintf("%s: %s", field, strings.Join(e.Errors[field], ", ")))
	}
	return fmt.Sprintf("%s: %s", e.Type, strings.Join(parts, "; "))
}

// ParseAPIError parses an API error response body. If the body does not
// match the expected envelope, the returned APIError uses a generic
// `http_<status>` type and the raw body as the message.
func ParseAPIError(statusCode int, body []byte) *APIError {
	var envelope struct {
		Error struct {
			Type    string              `json:"type"`
			Message string              `json:"message"`
			Errors  map[string][]string `json:"errors"`
		} `json:"error"`
	}

	apiErr := &APIError{StatusCode: statusCode}

	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Error.Type != "" {
		apiErr.Type = envelope.Error.Type
		apiErr.Message = envelope.Error.Message
		apiErr.Errors = envelope.Error.Errors
	} else {
		apiErr.Type = fmt.Sprintf("http_%d", statusCode)
		apiErr.Message = string(body)
	}
	return apiErr
}
