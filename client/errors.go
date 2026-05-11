package client

import (
	"encoding/json"
	"fmt"
)

// APIError represents a structured error from the Larm API.
//
// The Larm API returns errors as a JSON envelope of the form
// `{"error": {"type": "...", "message": "..."}}`. ParseAPIError
// decodes that envelope; callers should use errors.As to extract.
type APIError struct {
	StatusCode int    `json:"-"`
	Type       string `json:"error"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// ParseAPIError parses an API error response body. If the body does not
// match the expected envelope, the returned APIError uses a generic
// `http_<status>` type and the raw body as the message.
func ParseAPIError(statusCode int, body []byte) *APIError {
	var envelope struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	apiErr := &APIError{StatusCode: statusCode}

	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Error.Type != "" {
		apiErr.Type = envelope.Error.Type
		apiErr.Message = envelope.Error.Message
	} else {
		apiErr.Type = fmt.Sprintf("http_%d", statusCode)
		apiErr.Message = string(body)
	}
	return apiErr
}
