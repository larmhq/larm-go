// Package client is the typed Go SDK for the Larm public API.
//
// # Construction
//
// Use [New] to construct a client. The returned [*ClientWithResponses] is the
// generated typed client; every API operation has a corresponding method.
//
//	c, err := client.New(
//	    "https://app.larm.dev/api/v1",
//	    client.WithToken("larm_api_..."),
//	)
//	resp, err := c.ListMonitorsWithResponse(ctx, nil)
//
// The base URL passed to [New] must include the API version path. Trailing
// slashes are trimmed.
//
// # Authentication
//
// Authentication is supplied through a [TokenSource] — most callers use the
// convenience [WithToken] option for a static bearer token. For credentials
// that need to be refreshed (OAuth, etc.) implement [TokenSource] or use the
// [TokenSourceFunc] adapter. A token source that returns the empty string
// (with nil error) results in the Authorization header being omitted, which
// is useful for unauthenticated public endpoints.
//
// # Retries and timeouts
//
// Requests are wrapped in [RetryTransport], which retries on HTTP 429 and 5xx
// responses with exponential backoff. The Retry-After header is honored on
// 429 responses. Configure the retry count with [WithRetries] (default 3) and
// the per-request HTTP timeout with [WithTimeout] (default 30 seconds).
//
// # Errors
//
// API errors are returned as [*APIError] with the parsed envelope. Callers
// should use errors.As to inspect:
//
//	var apiErr *client.APIError
//	if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
//	    // ...
//	}
//
// # Stability
//
// While the SDK is at 0.x, breaking changes may occur on minor version bumps
// as the contract is shaken down by the CLI and the Terraform provider. After
// 1.0, breaking changes will require a major version bump and a new module
// path.
package client
