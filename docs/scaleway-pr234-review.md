# PR #234 Scaleway Provider Review

Date: 2026-05-08

## Scope

Reviewed the Scaleway provider implementation against the existing GCP provider shape, Scaleway public documentation, and the test suite added in PR #234.

Primary Scaleway references:

- Serverless Containers limitations: https://www.scaleway.com/en/docs/serverless-containers/reference-content/containers-limitations/
- Serverless Containers concepts: https://www.scaleway.com/en/docs/serverless-containers/concepts/
- Serverless Containers port parameter: https://www.scaleway.com/en/docs/serverless-containers/reference-content/port-parameter-variable/
- Managed Database for PostgreSQL and MySQL docs: https://www.scaleway.com/en/docs/managed-databases-for-postgresql-and-mysql/

## GCP Provider Map

The GCP provider has a mature `Project` path:

- `Project.Construct` registers the component, builds shared infra, then topologically deploys services.
- `BuildGlobalConfig` creates VPC/subnet/public IP/private DNS, optional wildcard DNS/cert, optional build infra, external registry mirrors, and VPC peering for managed data services.
- `EnableGcpAPIs` explicitly enables required GCP APIs.
- Service dispatch chooses Cloud Run vs Compute Engine. Cloud Run gets service accounts, secret references, startup probes, VPC access, ingress, and max-instance scaling. Compute Engine handles host-port style services.
- Managed Postgres maps to Cloud SQL with private service access, backup config, edition, availability, DB/user handling, and deletion protection options.
- Managed Redis maps to Memorystore with private service access, tier/version/memory mapping, and encryption options.
- CD integration creates an explicit GCP provider and uploads the updated `ProjectUpdate` protobuf to GCS when `DEFANG_STATE_URL` is set.

The Scaleway provider is intentionally smaller:

- `Project` creates one Serverless Containers namespace, one Private Network, and one resource per Compose service.
- App services map to Scaleway Serverless Containers.
- Postgres maps to Scaleway Managed Database for PostgreSQL.
- Redis maps to Scaleway Managed Redis.
- Build-from-source is not implemented; services need pre-built images.
- There is no Scaleway equivalent yet for GCP's DNS/load-balancer layer, project state upload, LLM support, or build pipeline.

## Findings And Fixes

- [x] **Unsupported Compose inputs were silently ignored.** Scaleway accepted fields that GCP either implements or routes elsewhere, including LLM config, host-mode ports, and multiple exposed ports. Serverless Containers expose only one port and have no host-port equivalent. Fixed by failing fast with explicit errors instead of producing a misleading deployment.

- [x] **Scaleway-reserved ports and environment variables were not protected.** Scaleway reserves `PORT`, `SCW_*`, and several internal ports. The previous implementation would pass those through to the provider and let apply-time behavior fail or behave unpredictably. Fixed with provider-side validation.

- [x] **Documented Serverless Container resource limits were not enforced.** The implementation converted Compose CPU/memory/replica inputs directly to Scaleway arguments without rejecting values outside documented Serverless Container ranges. Fixed by validating CPU, memory, and max scale before creating the container.

- [x] **Image platform constraints were accepted but ignored.** Scaleway documents `linux/amd64` as the required image architecture for Serverless Containers. The provider accepted `platform` but did not enforce it. Fixed by rejecting non-`linux/amd64` platform inputs before resource creation.

- [x] **Health checks were accepted by the schema but dropped.** GCP maps Compose health checks into a startup probe. Scaleway's SDK supports HTTP health checks, so silently ignoring the input was not good enough. Fixed by emitting a Scaleway HTTP health check using Compose retry and interval values.

- [x] **Test quality was too happy-path heavy.** The initial tests proved that basic resources were emitted, but they did not assert rejection of unsupported Compose semantics, provider-documentation constraints, or health-check mapping. Added focused unit tests for unsupported inputs, invalid limits, reserved names/ports, and health-check resource shape.

## Remaining GCP Parity Gaps

These are not regressions from the patch above, but they remain real limitations compared with GCP:

- Build-from-source for Scaleway is still missing. GCP provisions build infra and resolves images; Scaleway requires pre-built images.
- Scaleway CD does not upload a post-deploy `ProjectUpdate` artifact like GCP does for `gs://` state URLs.
- There is no Scaleway LLM resource mapping.
- There is no Scaleway DNS/load-balancer abstraction comparable to GCP's wildcard certificate and load-balancer path.
- Postgres and Redis still need real Scaleway preview/apply validation for private-network endpoint shape, service-to-database connectivity, and custom domain behavior.

## Verification

- `make test_unit` passed before the fixes.
- `go test ./provider/defangscaleway/... ./tests/scaleway` passed after the fixes.
