# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - TBD

### Added

- Initial release: typed Go client generated from the Larm public OpenAPI spec.
- `client.New` constructor with options:
  - `WithToken(token string)`
  - `WithTokenSource(ts TokenSource)`
  - `WithRetries(n int)`
  - `WithUserAgent(ua string)`
  - `WithBaseTransport(rt http.RoundTripper)`
  - `WithTimeout(d time.Duration)`
- `TokenSource` interface with `StaticToken` and `TokenSourceFunc` adapters.
- `RetryTransport` exported for callers that bypass `New` — retries 429 + 5xx with exponential backoff and `Retry-After` honored.
- `APIError` type for parsed `{"error": {"type", "message"}}` envelopes.
- Race-tested, `govulncheck`-clean, lint-clean across 13 enabled linters.
- Integration test gated on `LARM_API_KEY`.
- Build-time tooling isolated in a `tools/` submodule so consumers' dep graphs stay minimal.
