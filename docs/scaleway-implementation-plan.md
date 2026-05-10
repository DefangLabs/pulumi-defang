# Scaleway Provider Implementation Plan

This document plans a fourth Defang Pulumi provider, tentatively named `defang-scaleway`, by comparing the existing AWS, GCP, and Azure providers and mapping the same Defang abstractions onto Scaleway resources.

## Current Provider Shape

All three providers follow the same top-level contract:

- A Pulumi provider package built with `pulumi-go-provider/infer`.
- One custom `Build` resource that performs an imperative cloud-native image build and waits for completion.
- Four component resources exposed to users and generated SDKs: `Project`, `Service`, `Postgres`, and `Redis`.
- `Project` accepts Compose-derived `services`, optional `networks`, and an `etag`, creates shared cloud infrastructure, then creates per-service child components.
- `Project` outputs `endpoints` and `loadBalancerDns` for CD/CLI compatibility.
- Per-service creation branches on `svc.Postgres`, `svc.Redis`, `svc.LLM`, or regular container service.
- Service ordering is derived from Compose dependencies, either by topological sort or by explicitly creating managed services before app services.
- Standalone `Service` components are image-only in the current providers; build-from-source is handled by `Project`.
- Shared helpers live in `provider/common` and `provider/compose`; provider-specific cloud glue lives in `provider/defang<cloud>/<cloud>`.
- Shared schema inputs come from `provider/compose`: services, networks, build, ports, deploy resources, environment, health checks, `x-defang-postgres`, `x-defang-redis`, and `x-defang-llm`.

The implementation pattern is visible in:

- `provider/defangaws/provider.go` and `provider/defangaws/project.go`
- `provider/defanggcp/provider.go` and `provider/defanggcp/project.go`
- `provider/defangazure/provider.go` and `provider/defangazure/project.go`
- `provider/compose/types.go`

## Existing Cloud Mappings

| Defang concept | AWS | GCP | Azure |
| --- | --- | --- | --- |
| Provider package | `defang-aws` | `defang-gcp` | `defang-azure` |
| Main component | `defang-aws:index:Project` | `defang-gcp:index:Project` | `defang-azure:index:Project` |
| App runtime | ECS Fargate service | Cloud Run, with Compute Engine fallback for unsupported port shapes | Azure Container Apps |
| Image registry | ECR | Artifact Registry | Azure Container Registry |
| Image build | CodeBuild custom resource | Cloud Build custom resource | ACR Task custom resource |
| PostgreSQL | RDS | Cloud SQL | Azure Database for PostgreSQL Flexible Server |
| Redis | ElastiCache | Memorystore | Azure Managed Redis / Redis Enterprise |
| Network | Existing/default VPC plus SGs/subnets/private hosted zone | VPC/subnet/private DNS/service networking/NAT as needed | Resource group, VNet/subnets/private DNS when managed services exist |
| Public ingress | ALB plus optional ACM/Route 53 | Cloud Run direct URL plus global LB for host ports/domain | Container App FQDN; no explicit LB |
| Config/secrets | SSM Parameter Store | Secret Manager | Key Vault |
| LLM | Bedrock policy and env wiring | Vertex AI API/IAM and env wiring | Azure OpenAI deployment |
| Resource tagging | AWS tags where supported | labels/tags where supported | cascading `Tags` transformation |

## Scaleway Pulumi Provider

There is an existing Pulumi Scaleway provider maintained by Pulumiverse:

- Pulumi registry package: <https://www.pulumi.com/registry/packages/scaleway/>
- Install/config docs: <https://www.pulumi.com/registry/packages/scaleway/installation-configuration/>
- Repository: <https://github.com/pulumiverse/pulumi-scaleway>
- Go SDK import: `github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway`
- Node package: `@pulumiverse/scaleway`
- Python package: `pulumiverse_scaleway`
- .NET package: `Pulumiverse.Scaleway`

As of the current Pulumi registry page, the provider is at `v1.48.0`, published April 29, 2026. The provider is based on the official Scaleway Terraform provider.

Required provider configuration:

- `SCW_ACCESS_KEY` / `scaleway:access_key`
- `SCW_SECRET_KEY` / `scaleway:secret_key`
- `SCW_ORGANIZATION_ID` / `scaleway:organization_id`
- `SCW_DEFAULT_PROJECT_ID` / `scaleway:project_id`
- Optional `SCW_DEFAULT_REGION` / `scaleway:region`, defaulting to `fr-par`
- Optional `SCW_DEFAULT_ZONE` / `scaleway:zone`, defaulting to `fr-par-1`

## Scaleway Target Architecture

The most natural Scaleway implementation is:

| Defang concept | Scaleway target | Pulumi module/resource |
| --- | --- | --- |
| Provider package | `defang-scaleway` | new inferred provider package |
| Main component | `defang-scaleway:index:Project` | local component |
| App runtime | Serverless Containers | `scaleway.containers.Namespace`, `scaleway.containers.Container`, optional `Domain`, `Cron`, `Trigger` |
| Image registry | Scaleway Container Registry | `scaleway.registry.Namespace` or the registry endpoint attached to a Serverless Containers namespace |
| Image build | Decision needed: external/local build push, Scaleway Jobs, or CI-side build | No direct managed build equivalent found in Pulumi provider |
| PostgreSQL | Managed Database for PostgreSQL | `scaleway.databases.Instance`, `Database`, `User`, `Acl`, possibly `Privilege` |
| Redis | Managed Redis | `scaleway.redis.Cluster` |
| Network | VPC + regional Private Network | `scaleway.network.Vpc`, `scaleway.network.PrivateNetwork` |
| Private connectivity | Attach Serverless Containers, PostgreSQL, and Redis to the Private Network | resource-specific private network arguments |
| Public ingress | Serverless Container generated endpoint and optional custom domain | `scaleway.containers.Container.domainName`, `scaleway.containers.Domain`; DNS CNAME outside provider unless Scaleway domain APIs are used |
| Config/secrets | Serverless Container secret env vars and/or Secret Manager | `scaleway.secrets.Secret`, `scaleway.secrets.Version`, plus container `secretEnvironmentVariables` |
| LLM | Scaleway Managed Inference | `scaleway.inference.Deployment`, `Model`, `getModel` |
| Scheduled/background jobs | Serverless Jobs | `scaleway.job.Definition` |
| IAM for private containers | IAM application/API key/policy | `scaleway.iam.Application`, `ApiKey`, `Policy` |

Relevant Scaleway docs:

- Serverless Containers concepts: <https://www.scaleway.com/en/docs/serverless-containers/concepts/>
- Serverless Containers and Private Networks: <https://www.scaleway.com/en/docs/serverless-containers/reference-content/containers-private-networks/>
- Container Registry API/docs: <https://www.scaleway.com/en/developers/api/registry/>
- Serverless Jobs concepts: <https://www.scaleway.com/en/docs/serverless-jobs/concepts/>

Relevant Pulumi API docs:

- Scaleway API modules: <https://www.pulumi.com/registry/packages/scaleway/api-docs/>
- Containers module: <https://www.pulumi.com/registry/packages/scaleway/api-docs/containers/>
- Container resource: <https://www.pulumi.com/registry/packages/scaleway/api-docs/containers/container/>
- Container namespace: <https://www.pulumi.com/registry/packages/scaleway/api-docs/containers/namespace/>
- Registry namespace: <https://www.pulumi.com/registry/packages/scaleway/api-docs/registry/>
- Registry namespace resource: <https://www.pulumi.com/registry/packages/scaleway/api-docs/registry/namespace/>
- Databases module: <https://www.pulumi.com/registry/packages/scaleway/api-docs/databases/>
- Database instance: <https://www.pulumi.com/registry/packages/scaleway/api-docs/databases/instance/>
- Redis module: <https://www.pulumi.com/registry/packages/scaleway/api-docs/redis/>
- Redis cluster: <https://www.pulumi.com/registry/packages/scaleway/api-docs/redis/cluster/>
- VPC/private network: <https://www.pulumi.com/registry/packages/scaleway/api-docs/network/privatenetwork/>
- Secrets module: <https://www.pulumi.com/registry/packages/scaleway/api-docs/secrets/>
- Secret version: <https://www.pulumi.com/registry/packages/scaleway/api-docs/secrets/version/>
- Jobs module: <https://www.pulumi.com/registry/packages/scaleway/api-docs/job/>
- Job definition: <https://www.pulumi.com/registry/packages/scaleway/api-docs/job/definition/>
- Inference module: <https://www.pulumi.com/registry/packages/scaleway/api-docs/inference/>
- Container image build/push guide: <https://www.scaleway.com/en/docs/serverless-containers/how-to/build-push-container-image>
- Scaleway CLI container deploy reference: <https://cli.scaleway.com/container/>

## Proposed Package Layout

Mirror the current providers:

```text
provider/defangscaleway/
  provider.go
  project.go
  service.go
  postgres.go
  redis.go
  build.go
  scaleway/
    scaleway.go
    config.go
    naming.go
    image.go
    registry.go
    containers.go
    postgres.go
    redis.go
    network.go
    secrets.go
    iam.go
    inference.go
    jobs.go
    recipe.go
tests/scaleway/
examples/scaleway-{yaml,go,nodejs,python,dotnet}/
defang-scaleway.mk
```

Use `defangscaleway` as the Go package name to match the current package naming convention (`defangaws`, `defanggcp`, `defangazure`).

## SharedInfra Proposal

`provider/defangscaleway/scaleway.SharedInfra` should contain:

- `Region string`
- `Zone string`
- `ProjectID string`
- `Vpc *network.Vpc`
- `PrivateNetwork *network.PrivateNetwork`
- `ContainerNamespace *containers.Namespace`
- `RegistryNamespace *registry.Namespace` if separate registry namespace is required
- `BuildInfra *BuildInfra` if a managed build path is implemented
- `ConfigProvider compose.ConfigProvider`
- `SecretPrefix string`
- `Etag string`
- `LLMInfra *LLMInfra` if Managed Inference support is implemented in the first pass

Create shared infra lazily:

- Always create a Serverless Containers namespace for app services.
- Create registry resources only when a service needs a build or when image mirroring is required.
- Create a VPC/private network when any service uses Postgres, Redis, private networking, or private service discovery.
- Create IAM resources only when a private container endpoint is required.
- Create LLM infra only when `common.IsProjectUsingLLM` is true.

## Implementation Decisions

### Image Build

AWS, GCP, and Azure all have a custom `Build` resource that invokes a managed cloud build service. I did not find an equivalent Scaleway managed image build product exposed by the Pulumi provider. Options:

1. Use Defang CD/CLI to build and push before Pulumi, then make `GetServiceImage` return the already-pushed registry image.
2. Use Scaleway Serverless Jobs to run a build container and push to Container Registry.
3. Use the current CD sandbox build path and push directly to Scaleway Container Registry from the CD environment.

**Implemented:** Option 2 — Scaleway Serverless Jobs runs a patched Kaniko executor. See `provider/defangscaleway/build.go` for the `Build` custom resource and `provider/defangscaleway/scaleway/image.go` for the `GetServiceImage` dispatch. The Kaniko image is patched for gVisor sandbox compatibility (chown, setgroups, apt sandbox). Source context is fetched from Scaleway Object Storage (S3-compatible), and built images are pushed to Scaleway Container Registry.

### Runtime

Use `scaleway.containers.Container` for regular Compose services:

- `registryImage`: built or provided image URI
- `port`: first public HTTP/gRPC port where applicable
- `protocol`: `http1` by default, `h2c` for gRPC/HTTP2
- `cpuLimit` and `memoryLimit`: map from Compose resources with Scaleway-supported pairs
- `minScale`/`maxScale`: map from Defang deploy settings or defaults
- `environmentVariables` and `secretEnvironmentVariables`: built from Compose env/config/secrets
- `privacy`: public for externally exposed HTTP services, private for internal-only services
- `deploy`: true, with `registrySha256` or equivalent replacement trigger if available
- `privateNetworkId`: when managed services or private networking are used

Scaleway Serverless Containers are stateless, autoscaled, expose generated endpoints, support custom domains, support secret env vars, and support Private Networks. Private Network support is suitable for egress from a container to databases/Redis on the same private network; validate current support for inbound private container-to-container service discovery before relying on it.

### PostgreSQL

Use Scaleway Managed Databases:

- Create a `databases.Instance` with PostgreSQL engine, node type, HA/backups/storage from Defang config.
- Create `databases.Database` for the application database.
- Create `databases.User` for application credentials.
- Create `databases.Privilege` if required by provider semantics.
- Attach the database to the Private Network.
- Prefer private endpoint connection strings for app containers.
- Store/generated secrets should use Pulumi secrets and avoid exposing passwords in plain outputs.

### Redis

Use `scaleway.redis.Cluster`:

- Map Defang Redis size to Scaleway node type and cluster settings.
- Attach to the Private Network.
- Prefer `passwordWo`/write-only password handling where the Pulumi SDK exposes it.
- Return `redis://` or `rediss://` connection URL as the managed endpoint.
- Confirm TLS/auth fields exposed by the Pulumi resource before final env URL formatting.

### Secrets and Config

Existing providers read Defang config from provider-specific secret stores. Scaleway has both Secret Manager resources and Serverless Container secret env vars.

Recommended first implementation:

- Implement `scaleway.ConfigProvider` behind `compose.ConfigProvider`.
- Use Scaleway Secret Manager for project-level `defang config set` values where the CLI/control plane can provision them.
- Prefer `secrets.Version` write-only payload fields, where available, to avoid storing secret values in Pulumi state.
- When creating containers, split non-sensitive env into `environmentVariables` and sensitive values into `secretEnvironmentVariables`.
- Treat missing config as a deployment error, matching current project preference for fail-fast configuration.

### LLM

Scaleway Managed Inference can map to the existing `svc.LLM` path:

- Look up existing model with `scaleway.inference.getModel`.
- Create `scaleway.inference.Deployment` with `acceptEula` where needed.
- Export the deployment endpoint/base URL into dependent service env vars.

**Implemented differently:** Scaleway LLM support works via CLI-level compose fixup rather than Pulumi resources. The CLI strips `provider: type: model` services and injects env vars pointing to Scaleway's Generative API (`https://api.scaleway.ai/v1/`), which is OpenAI-compatible. No sidecar or Managed Inference deployment needed. See `src/pkg/cli/compose/fixup.go` in the CLI repo.

## Detailed Checklist

### Repository Wiring

- [x] Add `defang-scaleway` to `PACKS` in `Makefile`.
- [x] Add `defang-scaleway.mk` mirroring the other provider makefiles.
- [x] Add provider package at `provider/defangscaleway`.
- [x] Add helper package at `provider/defangscaleway/scaleway`.
- [x] Add Scaleway provider SDK dependency to root `go.mod`.
- [x] Add Scaleway CD program path in `cd/program`.
- [x] Update `cd/program/program.go` accepted providers from `aws/gcp/azure` to include `scaleway`.
- [x] Update Dockerfile targets and image build matrix for `scaleway`.
- [ ] Update release/goreleaser config for a fourth provider binary/package.
- [ ] Update `.github/workflows/test.yml` provider, SDK, example, schema, and Go SDK drift matrices.
- [ ] Update `.github/workflows/release.yml` SDK/image matrices, Scaleway registry auth, and provider version pinning.
- [x] Add generated schema path `provider/cmd/pulumi-resource-defang-scaleway/schema.json`.
- [x] Add generated Go SDK path `sdk/v2/go/defang-scaleway`.
- [x] Update README/docs installation pages with `defang-scaleway`.

### Provider Schema

- [x] Implement `provider/defangscaleway/provider.go`.
- [x] Register `Build` resource if a build path is included.
- [x] Register `Project`, `Service`, `Postgres`, and `Redis` components.
- [x] Set metadata: description, keywords, homepage, repo, publisher, logo, license, plugin download URL.
- [x] Set language package names, likely `@defang-io/pulumi-defang-scaleway` and matching Go/Python/.NET names.
- [x] Add schema tests for `Project`, `Service`, `Postgres`, and `Redis`.

### Project Component

- [x] Define `ProjectInputs` with `services`, `networks`, optional Scaleway config, and `etag`.
- [x] Define `ProjectOutputs` with `endpoints` and `loadBalancerDns`.
- [x] Register component and outputs exactly like existing providers.
- [x] Build shared Scaleway infra.
- [x] Create a dry-run config provider path.
- [x] Create managed services before dependent app services or use `common.TopologicalSort` with explicit dependencies.
- [ ] Keep all child resources parented under the project or service component; existing tests enforce this hierarchy.
- [x] Avoid creating resources inside `ApplyT`; use explicit outputs and dependencies instead.
- [x] Return per-service endpoints.
- [x] Decide whether `loadBalancerDns` should be empty, nil, or a generated container domain. Current prototype returns an empty string for CD/CLI compatibility because Scaleway Serverless Containers expose per-service endpoints.

### Shared Infrastructure

- [x] Read region/zone/project ID from Scaleway provider config or explicit project inputs.
- [ ] Create deterministic Defang names with length/character constraints.
- [x] Create Serverless Containers namespace.
- [x] Create or reuse Container Registry namespace.
- [x] Create VPC/private network when needed.
- [x] Add common tags to every Scaleway resource where supported.
- [ ] Implement IAM application/API key/policy for private containers if required.
- [ ] Implement Secret Manager integration or document the CLI-owned secret path.

### Image Handling

- [x] Implement `GetServiceImage` for pre-built images.
- [x] Implement `GetServiceImage` for `build:` services once the build decision is made.
- [x] Keep standalone `Service` image-only unless all other providers are changed consistently.
- [x] If building in CD, push to Scaleway registry and feed the immutable image URI into Pulumi.
- [x] If building in Pulumi, implement `Build` custom resource, polling, timeout, dry-run behavior, replacement triggers, and image output.
- [ ] Add unit tests for image URI parsing and build trigger hashing.

### Service Component

- [x] Define `ServiceInputs`/`ServiceOutputs` matching existing providers.
- [x] Create `scaleway.containers.Container` for regular services.
- [x] Map Compose ports to Scaleway `port`, `protocol`, privacy, and endpoint shape.
- [x] Map Compose resources to valid Scaleway CPU/memory combinations.
- [x] Map env/config/secrets to regular and secret env vars.
- [x] Attach Private Network when private dependencies exist.
- [ ] Generate endpoint as `https://` plus container `domainName`.
- [x] Add custom domain support with `containers.Domain` when `domainName` is configured.
- [ ] Decide how to handle non-HTTP ports, UDP, multiple public ports, and service-to-service internal ports.
- [x] Add service unit tests and at least one integration-style Pulumi test.

### Postgres Component

- [x] Define `PostgresInputs`/`PostgresOutputs`.
- [x] Create Managed Database instance.
- [x] Create database/user/password.
- [x] Attach Private Network.
- [x] Build `DATABASE_URL`/endpoint output for dependent services.
- [ ] Apply vector extension support if Scaleway PostgreSQL supports it and Defang `postgres` options require it.
- [x] Add tests for schema registration and connection string formatting.

### Redis Component

- [x] Define `RedisInputs`/`RedisOutputs`.
- [x] Create Managed Redis cluster.
- [x] Attach Private Network.
- [x] Build Redis endpoint/connection URL.
- [x] Add tests for schema registration and URL formatting.

### CD Program and Examples

- [x] Add `deployScaleway` in `cd/program/scaleway.go`.
- [x] Add provider config validation and clear unsupported-provider errors.
- [x] Add `examples/scaleway-yaml/Pulumi.yaml`.
- [x] Generate Go, Node.js, Python, and .NET examples.
- [ ] Add Scaleway to README example generation if desired.
- [ ] Add `compare` support if the comparison tool assumes exactly three providers.

### Tests and Verification

- [x] Add `tests/testutil.MakeScalewayTestServer`.
- [x] Add Scaleway URN/parent hierarchy helpers if needed.
- [x] Add project construct tests.
- [ ] Add parent hierarchy tests matching the existing AWS/Azure coverage.
- [x] Run `go test ./provider/...`.
- [x] Run `make provider_defang-scaleway`.
- [x] Run `make schema_defang-scaleway`.
- [x] Run `go test ./tests/scaleway -short`.
- [x] Run `make test_unit`.
- [x] Run `make test_provider`.
- [ ] Run `make test_cd`.
- [ ] Run `make build` once SDK generation is wired.
- [ ] Add CI matrix entries for Scaleway.

## Suggested Work Split

1. Provider skeleton and build/release wiring: `Makefile`, `defang-scaleway.mk`, provider registration, schema tests, SDK naming.
2. Scaleway shared infra and naming: config, tags, VPC/private network, namespace, registry, config provider.
3. Service runtime: Serverless Containers resource mapping, env/secrets, endpoints, domains, resource sizing.
4. Managed data services: PostgreSQL and Redis components plus connection URL/env integration.
5. CD/examples/tests: `cd/program/scaleway.go`, examples, Dockerfile/release/CI updates, final test pass.

## Risks and Open Questions

- ~~Scaleway does not appear to expose a managed build service equivalent to CodeBuild, Cloud Build, or ACR Tasks in the Pulumi provider.~~ **Resolved:** Build implemented via Kaniko in Scaleway Serverless Jobs. Requires a patched Kaniko image for gVisor sandbox compatibility.
- Serverless Containers support Private Network egress to private resources, but private inbound service-to-service discovery needs validation before replacing current provider behavior for internal services.
- Serverless Containers have one primary port. Multiple ports, UDP, and arbitrary TCP service exposure may require a fallback runtime or an explicit unsupported error.
- Custom domains require a DNS CNAME to the generated container endpoint. Scaleway `containers.Domain` can attach the domain, but DNS automation depends on whether the zone is managed by Scaleway or external.
- CPU/memory limits must be normalized to Scaleway-supported combinations, similar to AWS Fargate resource normalization.
- Secret Manager and Serverless Container secret environment variables are separate mechanisms; decide which is the source of truth for Defang project config.
- Managed Inference model availability, EULA behavior, and endpoint auth should be validated before claiming LLM parity.
