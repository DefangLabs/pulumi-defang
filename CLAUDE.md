# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A Pulumi multi-cloud provider that deploys Docker Compose applications to AWS, GCP, and Azure. Written in Go, it generates SDKs for TypeScript, Python, Go, and .NET from provider schemas.

## Build Commands

```bash
make provider          # Build all three provider binaries (to ./bin/)
make schema            # Generate OpenAPI schemas from provider binaries
make go_sdk            # Generate Go SDKs (also checks for InputInput type bugs)
make sdks              # Generate all language SDKs (go, nodejs, python, dotnet)
make build             # Full build: provider + schema + sdks
make install           # Install provider binaries to $GOPATH/bin
make lint              # golangci-lint with --fix
make ensure            # go mod tidy for both root and tests/
```

## Testing

```bash
make test              # Run both unit and provider tests
make test_unit         # Unit tests only: go test ./provider/...
make test_provider     # Provider integration tests: cd tests && go test -short ./...
```

Run a single test:
```bash
go test -v -run TestName ./provider/compose/...           # unit test
cd tests && go test -v -run TestName -short ./aws/...     # provider test
```

Provider tests live in a separate `tests/` module with their own `go.mod`. They use a mock Pulumi server from `tests/testutil/`.

## Git Hooks

Run `make install-git-hooks` to set up:
- **Pre-commit:** `lint-staged` — runs golangci-lint and tests only for affected providers
- **Pre-push:** `make provider test go_sdk` — full build + test

The lint-staged config (`.lintstagedrc.js`) is smart: changes to `provider/compose` or `provider/common` trigger all provider tests since all three import them.

## Architecture

### Provider Pattern

Each cloud (AWS/GCP/Azure) follows the same structure:
- `provider/defang{aws,gcp,azure}/provider.go` — registers components via `pulumi-go-provider`'s `infer.Provider()`
- `provider/defang{aws,gcp,azure}/project.go` — top-level Project component
- `provider/defang{aws,gcp,azure}/service.go` — individual Service component
- `provider/defang{aws,gcp,azure}/postgres.go` — Postgres-compatible managed database component
- `provider/defang{aws,gcp,azure}/redis.go` — Redis-compatible managed cache component
- `provider/defang{aws,gcp,azure}/{aws,gcp,azure}/` — cloud-specific resource creation

Components: **Project** (orchestrator), **Service** (container), **Postgres** (managed DB), **Redis** (AWS only), **Build** (AWS only, resource not component).

### Pulumi inputs and outputs

- Pulumi input types should use native types or `pulumi.*Input` types, no `pulumi.*Output`!
- Pulumi output types should use `pulumi.*Output` types, no `pulumi.*Input`!

### Shared Code

- `provider/compose/` — Docker Compose types (`ServiceConfig`, `BuildConfig`, `DeployConfig`, etc.) shared across all providers
- `provider/common/` — Cross-provider utilities (health checks, ingress detection, DNS, topology sorting)

### SDK Generation Flow

Done by `make sdks`:

1. Build provider binary → `./bin/pulumi-resource-defang-{aws,gcp,azure}`
2. Extract schema → `provider/cmd/*/schema.json`
3. `pulumi package gen-sdk` generates each language SDK into `sdk/`

Per-provider build logic is in `defang-{aws,gcp,azure}.mk`.

### Tooling

Tools are managed by `flake.nix`, which imports `shell.nix`, loaded by DirEnv's `.envrc`.

