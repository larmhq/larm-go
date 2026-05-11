# larm-go

Official Go SDK for the [Larm](https://larm.dev) public API. Consumed by [`larm-cli`](https://github.com/larmhq/larm-cli) and the Terraform provider.

## Status

Pre-1.0. Breaking changes are expected while the SDK contract is shaken down by its first consumers. Pin to a specific version.

## Install

```sh
go get github.com/larmhq/larm-go
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/larmhq/larm-go/client"
)

func main() {
	c, err := client.New(
		"https://app.larm.dev/api/v1",
		client.WithToken("larm_api_..."),
	)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := c.ListMonitorsWithResponse(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.Status())
}
```

### Options

- `WithToken(token string)` — bearer token for `Authorization` header. Empty tokens drop the header (matches `golang.org/x/oauth2`).
- `WithTokenSource(ts TokenSource)` — dynamic token source (for refreshable credentials).
- `WithRetries(n int)` — override default retry count (default: 3, retries on 429 + 5xx, respects `Retry-After`).
- `WithUserAgent(ua string)` — appended to the SDK's default `User-Agent`.
- `WithBaseTransport(rt http.RoundTripper)` — layer middleware (logging, tracing) below the retry transport.
- `WithTimeout(d time.Duration)` — per-request HTTP timeout (default: 30s). Per-call cancellation should still use `context.Context`.

If you need a fully custom `*http.Client` (e.g. to swap the entire transport chain), use `NewClientWithResponses` directly with the generated `WithHTTPClient` option.

### Errors

API errors are surfaced as `*client.APIError`:

```go
var apiErr *client.APIError
if errors.As(err, &apiErr) {
    fmt.Printf("API returned %d %s: %s\n", apiErr.StatusCode, apiErr.Type, apiErr.Message)
}
```

## Versioning

Strict semver. While the SDK is `0.x`, minor versions may break the public API as the contract is shaken down by `larm-cli` and `terraform-provider-larm`. Once `1.0.0` is released, breaking changes require a major version bump and a new module path (`github.com/larmhq/larm-go/v2`).

## Compatibility

Requires Go 1.24 or later. The SDK targets the latest patch of each supported Go minor since 1.24.

## Stability

The exported surface of the `client` package is part of the public API contract (within the semver guarantees above). The generated `client.gen.go` is regenerated when `api/openapi.yaml` changes; renames or removed fields there are breaking changes that require a major version bump.

## Development

Tool versions are pinned in `mise.toml`. Build-time codegen tools live in the `tools/` submodule so they don't bleed into consumers' dependency graphs.

```sh
mise install        # install Go, goimports, golangci-lint, govulncheck
make generate       # regenerate client.gen.go from api/openapi.yaml
make test           # run tests with -race
make lint           # golangci-lint with the project's strict ruleset
make vuln           # govulncheck against the dep graph + stdlib
make verify         # go mod verify
make check-generate # CI check that committed client matches the spec
make integration-test  # gated on LARM_API_KEY; hits prod by default (override with LARM_API_URL)
```

The OpenAPI spec at `api/openapi.yaml` is currently synced manually from the Larm backend repo. Automated sync via CI is planned.

## Reporting security issues

See [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE).
