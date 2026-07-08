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
make ensure            # go mod tidy for every go.mod in the tree (root, tests/, cd/)
```

## Testing

```bash
make test              # Run all three suites: test_unit + test_provider + test_cd
make test_unit         # Unit tests only: go test ./provider/...
make test_provider     # Provider integration tests: cd tests && go test -short ./...
make test_cd           # cd/ module tests with -race
COVER=1 make test      # Bust the test cache and write coverage_*.out
make coverage          # COVER=1 test + merged HTML report (opens in browser)
```

Run a single test:
```bash
go test -v -run TestName ./provider/compose/...           # unit test
cd tests && go test -v -run TestName -short ./aws/...     # provider test
cd cd && go test -v -run TestName -race                   # cd test
```

Provider tests live in a separate `tests/` module with their own `go.mod`. They use a mock Pulumi server from `tests/testutil/`. The `cd/` module is also a separate Go module (see Architecture).

## Git Hooks

Run `make install-git-hooks` to set up:
- **Pre-commit / pre-merge-commit:** `make pre-commit` → `lint-staged` (golangci-lint + `go test` for affected providers only). The pre-merge-commit hook additionally fails if `make ensure` would dirty `go.mod`/`go.sum`.
- **Pre-push:** `make -j4 pre-push` → `provider test image_all`, then fails if `sdk/v2/` has uncommitted or staged changes.

The lint-staged config (`.lintstagedrc.js`) is smart: changes to `provider/compose` or `provider/common` trigger all provider tests since all three import them.

**Generated SDKs must be committed alongside the change that regenerated them.** The pre-push hook enforces a clean `sdk/v2/` tree — when provider/schema changes regenerate SDKs, stage and commit the resulting `sdk/v2/` diff in the same push.

**Pre-push hang gotcha.** `make -j4 pre-push` reaches `pulumi package gen-sdk` via `image_%: go_sdk` and runs the aws/gcp/azure variants in parallel. They all share the invoking terminal, so if `pulumi` needs an interactive prompt (e.g. expired login) the push hangs silently with no visible question. Run `pulumi whoami` before pushing; if a push appears stuck, look for sleeping `pulumi package gen-sdk` children holding `/dev/tty` (`lsof -p <pid> | grep /dev/tty`).

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

Components: **Project** (orchestrator), **Service** (container), **Postgres** (managed DB), **Redis** (managed cache), **Build** (Pulumi *resource*, not component — registered by all three providers: CodeBuild on AWS, Cloud Build on GCP, ACR Task on Azure).

### Component Scopes: Project vs. Standalone

Each managed-resource component has **one implementation**, invoked two ways:

- **Standalone** — user instantiates `Service` / `Postgres` / `Redis` directly via the generated SDK. The component's `Construct` method runs with no shared project infrastructure by default. All three providers now use a type named `SharedInfra`; how the standalone path obtains one differs:
    - **AWS**: passes a nil `*SharedInfra` to the worker.
    - **GCP**: builds a minimal `*SharedInfra` via `NewStandaloneGlobalConfig(ctx)` (the function name predates the `GlobalConfig` → `SharedInfra` type rename and was kept for back-compat).
    - **Azure**: standalone `Service` builds a minimal `*SharedInfra` (resource group + managed environment, no VNet / DNS / Log Analytics / Key Vault) via `newStandaloneInfra`. Standalone `Postgres` / `Redis` create their own resource group inline.

  Callers in Go code may pass a richer `Infra` (`*SharedInfra` in all three providers) — these fields carry resource pointers and Pulumi Outputs and are **not** part of the SDK schema.
- **Project-owned** — `Project.Construct` runs `buildProject` / `newService`, which provisions shared infra (VPC, NAT, Artifact Registry, ALB/LB, DNS zones), then dispatches each service by registering the typed `*Outputs` component and calling the same internal worker (`createPostgres` / `createRedis` / `createECSService` / `createService`). Each component must be registered under the exact type token defined by the `<Component>ComponentType` constant so the SDK schema matches the runtime registration.

**Build-from-source on a `Service` is a Project-only concern.** Although each provider registers a standalone `Build` *resource* (CodeBuild / Cloud Build / ACR Task), `ServiceInputs` deliberately does not carry a `Build` field on any provider: build-from-source wiring requires the project-scope build pipeline (ECR + CodeBuild on AWS, Artifact Registry + Cloud Build on GCP, ACR + ACR Task on Azure). Don't re-add `Build` to standalone Service inputs.

**VPC-dependent features require infra** in GCP: VpcAccess on Cloud Run, and LLM bindings that cross VPC boundaries. Standalone Cloud Run skips `VpcAccess` when `infra.PublicIP == nil`; standalone routing forces Cloud Run regardless of port configuration (legacy behavior, kept for schema stability) — *except* when the service uses CE-only features (sidecars, volumes, volumes_from), in which case a standalone Compute Engine MIG is created on the GCP **default network**. Sidecars/volumes on a single-ingress-port service (which must be Cloud Run) are an error.

**Dependency handles on outputs.** AWS and GCP `*Outputs` structs carry an internal, untagged handle used by the project dispatcher for ordering and load-balancer wiring:
- AWS: `PostgresOutputs.Dependency`, `RedisOutputs.Dependency`, `ServiceOutputs.Dependency` — all `pulumi.Resource` (either the backing instance or a private-zone CNAME record). AWS `PostgresOutputs` also exports `InstanceIdentifier` (the RDS `DBInstanceIdentifier`) as a *schema* field for downstream consumers.
- GCP: `PostgresOutputs.Instance`, `RedisOutputs.Instance` (typed), `ServiceOutputs.LBEntry` (built inside the worker).
- Azure: standalone `AzurePostgresOutputs` / `AzureRedisOutputs` / `ServiceOutputs` currently expose only `Endpoint` — no equivalent untagged dependency handle. Project-scope wiring on Azure flows through the shared `ManagedEnvironment` instead of per-component handles.

Untagged fields are ignored by the Pulumi infer framework (`introspect.ParseTag` treats them as `Internal: true`), so they don't need `pulumi:"-"`.

### Compose-shape parity across providers

`compose.ServiceConfig` carries a growing set of compose-shaped fields — `containerName`, `workingDir`, `stopGracePeriod`, `volumes`, `volumesFrom`, `dependsOn`, `autoscaling`, per-port `listener` — plus the standalone-Service extras `sidecars`, `secrets`, `taskRoleArn` / `serviceAccountEmail`, `securityGroupIds`, `triggers`, `waitForSteadyState`. AWS `ServiceInputs` and GCP `ServiceInputs` expose these; the underlying workers (`CreateECSService`, `CreateComputeEngine`) implement them.

**Azure is intentionally behind on parity.** `provider/defangazure/service.go` `ServiceInputs` does *not* expose these fields yet, and the Container Apps worker does not honor them. Rationale: no current Defang customer stack drives the Azure Service standalone path with these features, so the added surface area (multi-container Container App revisions, Managed Identity reuse, Container Apps native secret refs, etc.) is deferred until there is a concrete consumer. **TODO(azure-parity)** — when adding this, mirror the AWS/GCP inputs 1:1 for schema stability, and follow the same design as GCP for sidecars (a `sidecars: map[string]ServiceConfig` alongside the main container). Keep any implementation-gap surface behind explicit `unsupported on Azure` errors rather than silent no-ops.

### Pulumi inputs and outputs

- Pulumi input types should use native types or `pulumi.*Input` types, no `pulumi.*Output`!
- Pulumi output types should use `pulumi.*Output` types, no `pulumi.*Input`!

### Shared Code

- `provider/compose/` — Docker Compose types (`ServiceConfig`, `BuildConfig`, `DeployConfig`, etc.) shared across all providers
- `provider/common/` — Cross-provider utilities (health checks, ingress detection, DNS, topology sorting)

### `cd/` — the deploy-driver

`cd/` is a separate Go module that compiles to the binary the Defang CLI runs inside the customer's cloud. It reads `compose.yaml` from S3/GCS/Azure Blob, then invokes the Pulumi program built from this repo's providers. It has its own `go.mod` and its own test suite (`make test_cd`, run with `-race`).

### SDK Generation Flow

Done by `make sdks`:

1. Build provider binary → `./bin/pulumi-resource-defang-{aws,gcp,azure}`
2. Extract schema → `provider/cmd/*/schema.json`
3. `pulumi package gen-sdk` generates each language SDK into `sdk/`

Per-provider build logic is in `defang-{aws,gcp,azure}.mk`.

### Generated files: README and SDKs

- `README.md`'s code blocks are regenerated from `examples/*/` by `scripts/check-readme-examples.sh --update` (run via `make README.md`). Don't hand-edit the embedded snippets — edit the example files and regenerate.
- `sdk/v2/**` is generated from provider schemas (see SDK Generation Flow). Treat it as build output that must be committed alongside the source change that produced it.

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

