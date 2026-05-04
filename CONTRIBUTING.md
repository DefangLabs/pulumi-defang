# Contributing

## What this is

Pulumi providers that deploy Docker Compose applications to AWS, GCP, and Azure. The repo produces three separate provider binaries (`pulumi-resource-defang-aws`, `pulumi-resource-defang-gcp`, `pulumi-resource-defang-azure`) and generates SDKs for Go, Node.js, Python, and .NET from each provider's schema.

## Build & test

Common targets (see `make help` for the full list):

```bash
make provider          # Build all three provider binaries into ./bin/
make schema            # Generate OpenAPI schemas from the provider binaries
make sdks              # Generate Go, Node.js, Python, and .NET SDKs into ./sdk/
make build             # Full build: provider + schema + sdks
make install           # Install provider binaries to $GOPATH/bin
make examples          # Generate language examples from per-provider YAML sources
make test              # Run all tests (unit + provider + cd)
make test_unit         # Unit tests only
make test_provider     # Provider integration tests
make lint              # golangci-lint with --fix
make ensure            # go mod tidy for both root and tests/
```

`make examples` regenerates language examples in `examples/{aws,gcp,azure}-{dotnet,go,nodejs,python}/` from the corresponding `examples/{aws,gcp,azure}-yaml/` sources.

## Git hooks

```bash
make install-git-hooks
```

Sets up:
- **Pre-commit:** `lint-staged` — runs golangci-lint and tests only for affected providers.
- **Pre-push:** `make provider test go_sdk` — full build + test.

The lint-staged config (`.lintstagedrc.js`) is provider-aware: changes to `provider/compose` or `provider/common` trigger all provider tests since all three providers depend on them.

## Repository layout

- `provider/cmd/pulumi-resource-defang-{aws,gcp,azure}/` — entry points for each provider binary; each emits its own `schema.json`.
- `provider/defang{aws,gcp,azure}/` — provider implementations (Project, Service, Postgres, Redis, plus AWS-only Build).
- `provider/compose/` — shared Docker Compose types.
- `provider/common/` — cross-provider utilities (health checks, ingress detection, DNS, topology sorting).
- `sdk/` — generated language SDKs.
- `examples/` — per-provider example programs in YAML, Go, Node.js, Python, and .NET.
- `tests/` — provider integration tests (separate Go module, uses a mock Pulumi server in `tests/testutil/`).

See [CLAUDE.md](CLAUDE.md) for a deeper architecture overview, including the Project-vs-standalone component scoping model.

## Additional details

This repository depends on the [pulumi-go-provider](https://github.com/pulumi/pulumi-go-provider) library.

## References

- [Pulumi Command provider](https://github.com/pulumi/pulumi-command/blob/master/provider/pkg/provider/provider.go)
- [Pulumi Go Provider repository](https://github.com/pulumi/pulumi-go-provider)
