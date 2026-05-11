//go:build tools

// Package tools pins the version of build-time tooling used by the SDK.
// It lives in its own Go module so that heavyweight tool dependencies do
// not leak into the consumer-facing module (github.com/larmhq/larm-go).
//
// To regenerate the typed client, run `make generate` from the repo root.
package tools

import (
	_ "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"
)
