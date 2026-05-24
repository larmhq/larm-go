# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - TBD

### Added

- `ReplaceStatusPageStructure` — atomic full-tree replace for a status page's groups, components, and monitor links. Takes the same shape `GetStatusPage` returns; nodes with an `id` are updated, nodes without are created, nodes missing from the payload are deleted.
- Component group CRUD: `CreateComponentGroup`, `GetComponentGroup`, `UpdateComponentGroup`, `DeleteComponentGroup` under `/status-pages/{id}/component-groups`.
- `make sync-spec` target — copies `apps/backend/priv/openapi.yaml` from the backend repo (override path with `LARM_BACKEND_REPO=...`) and regenerates `client.gen.go`. Was previously a manual `cp` + `make generate`.

### Changed

- **Breaking**: `StatusPage.Components` is now a polymorphic ordered tree (`[]StatusPageTreeEntry` discriminated by `type = "group" | "component"`) instead of a flat list of `StatusPageComponentSummary`. Groups appear inline alongside ungrouped components, in display order; components inside a group are nested under the group entry's own `Components` field. Use the generated `AsStatusPageGroupEntry`/`AsStatusPageComponentEntry` helpers to discriminate.
- `StatusPageSummary` is the new lighter type returned by `ListStatusPages`. It contains everything the previous `StatusPage` exposed except the components tree.

### Removed

- **Breaking**: `ListComponents` and `ListComponentGroups`. The components and groups for a status page are now returned in the tree on `GetStatusPage`; fetch the page instead of listing children separately.
- **Breaking**: `StatusPageComponentSummary` model. Superseded by `StatusPageComponentEntry` inside the tree.

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
