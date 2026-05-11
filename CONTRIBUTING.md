# Contributing

Thanks for considering a contribution. This SDK is small but tightly versioned — please read before opening a PR.

## Setup

```sh
mise install   # installs Go, goimports, golangci-lint, govulncheck at pinned versions
```

## Local dev loop

```sh
make generate       # regenerate client.gen.go from api/openapi.yaml
make fmt
make test           # tests run with -race by default
make lint           # all 13 linters must pass
make vuln           # zero known CVEs
make check-generate # generated client must match committed
```

Run `make test lint vuln check-generate` before pushing.

## Spec changes

The OpenAPI spec (`api/openapi.yaml`) is synced manually from the source-of-truth in the `larm` backend repo (`apps/backend/priv/openapi.yaml`). Spec changes must originate there and propagate forward — do not edit `api/openapi.yaml` in this repo as a primary source.

After a spec sync, run `make generate` and commit the regenerated `client/client.gen.go` alongside the updated spec.

## PR conventions

- Keep changes scoped. Multiple unrelated changes belong in separate PRs.
- Add tests for new public-API behavior. Wire-up tests should assert the actual HTTP request that goes out (via captured `http.RoundTripper`), not just compile.
- Public API additions update the godoc and the README options list.
- Breaking changes require a `CHANGELOG.md` entry and (after 1.0) a major version bump.

## Releases

Releases are cut by tagging a commit on `main`:

```sh
git tag v0.1.x
git push --tags
```

The `release.yml` workflow re-runs the full quality gauntlet on the tagged commit and creates a GitHub Release with auto-generated notes.
