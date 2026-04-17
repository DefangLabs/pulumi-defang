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
- `provider/defang{aws,gcp,azure}/build.go` — image build component
- `provider/defang{aws,gcp,azure}/service.go` — individual container Service component (requires image)
- `provider/defang{aws,gcp,azure}/postgres.go` — Postgres-compatible managed database component
- `provider/defang{aws,gcp,azure}/redis.go` — Redis-compatible managed cache component
- `provider/defang{aws,gcp,azure}/{aws,gcp,azure}/` — cloud-specific resource creation

Components: **Project** (orchestrator), **Service** (container), **Postgres** (managed DB), **Redis** (managed cache), **Build** (AWS only, resource not component).

### Component Scopes: Project vs. Standalone

Each managed-resource component has **one implementation**, invoked two ways:

- **Standalone** — user instantiates `Service` / `Postgres` / `Redis` directly via the generated SDK. The component's `Construct` method runs with no shared project infrastructure by default. For GCP it builds a minimal `GlobalConfig` (region + project only) via `NewStandaloneGlobalConfig`; for AWS a nil `SharedInfra` is used. Callers in Go code may pass a richer `Infra` (AWS: `*SharedInfra`, GCP: `*GlobalConfig`) — these fields carry resource pointers and are **not** part of the SDK schema.
- **Project-owned** — `Project.Construct` runs `buildProject` / `newService`, which provisions shared infra (VPC, NAT, Artifact Registry, ALB/LB, DNS zones), then dispatches each service by registering the typed `*Outputs` component and calling the same internal worker (`createPostgres` / `createRedis` / `createECSService` / `createService`). Each component must be registered under the exact type token defined by the `<Component>ComponentType` constant so the SDK schema matches the runtime registration.

**Build is a Project-only concern.** `Service` standalone is image-only — `ServiceInputs` does not carry a `Build` field. Build-from-source requires project-scope `BuildInfra` (Artifact Registry + Cloud Build on GCP, ECR + CodeBuild on AWS). Don't re-add `Build` to standalone Service inputs.

**VPC-dependent features require infra** in GCP: Compute Engine (CE), VpcAccess on Cloud Run, and LLM bindings that cross VPC boundaries. Standalone Cloud Run skips `VpcAccess` when `infra.PublicIP == nil`; standalone routing forces Cloud Run regardless of port configuration since CE cannot run without a VPC.

**Dependency handles on outputs.** Each managed `*Outputs` struct carries an internal, untagged handle used by the project dispatcher for ordering and load-balancer wiring:
- AWS: `PostgresOutputs.Dependency`, `RedisOutputs.Dependency`, `ServiceOutputs.Dependency` — all `pulumi.Resource` (either the backing instance or a private-zone CNAME record).
- GCP: `PostgresOutputs.Instance`, `RedisOutputs.Instance` (typed), `ServiceOutputs.LBEntry` (built inside the worker).

Untagged fields are ignored by the Pulumi infer framework (`introspect.ParseTag` treats them as `Internal: true`), so they don't need `pulumi:"-"`.

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

---

## SAM Workflows

When running inside SAM (detected by `$SAM_WORKSPACE_ID` being set), follow these additional guidelines.

### Ephemeral Environment

SAM VMs are ephemeral — **unpushed work is lost** when the VM shuts down. Push frequently, especially after:
- Schema changes (`make schema`)
- SDK generation (`make sdks`)
- Any provider code changes that pass tests

### Progress Reporting

Use `update_task_status` at key milestones:
- Schema generated successfully
- Provider binaries built
- Tests passing
- SDKs generated


### Coordination with Other Repos

Pulumi provider changes often require corresponding updates in the CLI or the Fabric backend. Use `dispatch_task` to coordinate cross-repo work rather than trying to make changes across multiple repositories directly.

### Context and Ideas

- Use `search_tasks` to find prior context and decisions related to this repo
- Use `create_idea` for improvements discovered but out of scope for the current task
- Use `search_messages` to find context from prior conversations

### Knowledge Graph

SAM maintains a persistent knowledge graph across sessions. Use it to preserve non-obvious context:

- **`add_knowledge`** — Store observations about:
  - User preferences and work style (entityType: `preference`)
  - Code conventions not captured in CLAUDE.md (entityType: `style`)
  - Architecture decisions and their rationale (entityType: `context`)
  - Project context: ongoing initiatives, deadlines, blockers (entityType: `context`)
- **`search_knowledge`** — Query before key decisions (e.g., search "Architecture" before changing provider structure, search "AzureSupport" before touching Azure provider code)
- **`update_knowledge`** / **`remove_knowledge`** — Fix stale or incorrect observations
- **`confirm_knowledge`** — When you verify an existing observation is still accurate

Do NOT store: code patterns derivable from the codebase, git history, ephemeral task details, or anything already in CLAUDE.md.

### Subprocess Restriction

Do NOT launch `claude` as a subprocess — use SAM's `dispatch_task` instead.

