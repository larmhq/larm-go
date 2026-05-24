.PHONY: test test-race lint fmt clean generate check-generate sync-spec vuln integration-test verify

# Override LARM_BACKEND_REPO to point at a different checkout.
LARM_BACKEND_REPO ?= $(HOME)/workspace/larm

# Tests run with the race detector on by default. Library, fast suite — race is
# correct-by-default and CI catches concurrent-misuse regressions.
test:
	go test -race ./...

# Explicit alias for clarity.
test-race: test

lint:
	golangci-lint run

fmt:
	go fmt ./...
	goimports -w .

clean:
	rm -rf dist/ coverage.out

# Generate the typed API client. The generator and its heavy build deps live
# in the tools/ submodule so they don't bleed into the consumer go.mod.
generate:
	cd tools && go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen \
		-config ../client/oapi_codegen_config.yml ../api/openapi.yaml

# Copy the OpenAPI spec from the backend and regenerate the client.
# Run this after backend API changes; backend is the source of truth for openapi.yaml.
sync-spec:
	cp $(LARM_BACKEND_REPO)/apps/backend/priv/openapi.yaml api/openapi.yaml
	$(MAKE) generate

# CI check: generated code matches committed code.
check-generate: generate
	@if [ -n "$$(git diff --name-only)" ]; then \
		echo "Generated code is out of date. Run 'make generate' and commit."; \
		git diff; \
		exit 1; \
	fi

# Vulnerability scan. govulncheck is installed via mise (see mise.toml).
vuln:
	govulncheck ./...

# Run integration tests against a real Larm API. Requires LARM_API_KEY.
# Override the base URL with LARM_API_URL (defaults to production).
integration-test:
	go test -tags=integration -race ./client/...

# Verify dep checksums match the cached versions.
verify:
	go mod verify
