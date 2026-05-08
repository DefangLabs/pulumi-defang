# Scaleway Provider Iteration Notes

## 2026-05-08: Privacy, Endpoints, and Multi-Service Support

### Changes Made

1. **Container privacy based on ingress ports** (`provider/defangscaleway/scaleway/containers.go`)
   - Added `containerPrivacy()` function that returns `"public"` when the service has ingress ports, `"private"` otherwise
   - Previously all containers were hardcoded as `"public"`
   - Private containers (background workers, consumers with no ports) now deploy as Scaleway private containers

2. **Endpoint format includes `https://` prefix** (`provider/defangscaleway/scaleway/containers.go`)
   - Public service endpoints now return `https://<domainName>` instead of just the domain name
   - Private service endpoints return the raw domain name (reachable only from within the private network)

3. **Postgres nil version panic fix** (`provider/defangscaleway/scaleway/postgres.go`)
   - Fixed nil pointer dereference when `pg.Version` is nil (no image specified for postgres service)
   - Now defaults to PostgreSQL-17 when no version is specified

4. **Multi-service project test** (`provider/defangscaleway/project_test.go`)
   - Added `TestBuildProjectMultiServiceWithPostgres` covering: public web service, private worker, and managed PostgreSQL
   - Verifies shared infra (namespace, private network), postgres instance, and correct privacy per service

5. **Private service unit tests** (`provider/defangscaleway/scaleway/containers_test.go`)
   - Added `TestContainerPrivacy` for privacy function
   - Added `TestCreateContainerServicePrivateService` verifying private container creation

6. **End-to-end example** (`examples/scaleway-yaml/Pulumi.yaml`, `examples/scaleway-e2e-demo/`)
   - Updated from a static nginx sample to a real request chain: public web service -> public API service -> managed PostgreSQL
   - Uses pre-built API/web images because Scaleway build-from-source is not implemented in this PR
   - Documents the current phased deployment path because standalone `Service.environment` inputs are plain strings and cannot yet directly consume component outputs like `${db.connectionUrl}` or `${api.endpoint}`

### Test Results

All 29 tests passing:
- `provider/defangscaleway/` — 3 tests (project-level)
- `provider/defangscaleway/scaleway/` — 17 tests (unit-level)
- `tests/scaleway/` — 9 tests (integration-level)

### What Works (Mock/Preview Level)

- Public HTTP service → Scaleway Serverless Container with `privacy: public`
- Private background worker → Scaleway Serverless Container with `privacy: private`
- Managed PostgreSQL → Scaleway Database Instance + Database + Privilege
- Managed Redis → Scaleway Redis Cluster
- Private network shared across all services
- Health check mapping from Compose to Scaleway HTTP health checks
- Validation of Scaleway-specific limits (CPU, memory, max scale, reserved ports/env vars)
- Dependency ordering via `common.TopologicalSort`

### Critical Finding: Private Network is Egress-Only

From Scaleway documentation (serverless-containers/reference-content/containers-private-networks):

- **Egress only**: Containers can reach databases/Redis on the private network
- **No inbound private traffic**: Container-to-container communication via private network is NOT supported
- VPC integration is now always enabled for namespaces in the Scaleway Pulumi provider; `activateVpcIntegration` is deprecated and should not be set explicitly
- **Only one Private Network per container**
- Container-to-container communication must use public HTTPS endpoints
- DNS resolution uses VPC DNS server (169.254.169.254) for `*.internal` records

**Implications for Defang:**
- The `privacy: private` setting for background workers is correct (no public endpoint needed)
- **Serverless Container to Serverless Container internal networking is not available on Scaleway.** Defang cannot implement a private service mesh between Scaleway Serverless Containers with the current platform APIs.
- Inter-service communication between Serverless Containers must go through public HTTPS URLs.
- Managed databases and Redis on the private network ARE reachable from containers (egress)
- The private network primarily serves database/Redis connectivity, not inter-service mesh
- This means the Scaleway provider cannot currently preserve Docker Compose service-name networking for container-to-container calls (for example, `http://api:8080`).

**Changes applied:**
- Removed explicit `ActivateVpcIntegration`; the Scaleway provider now treats VPC integration as always enabled
- All container endpoints use `https://` public URLs (even private containers get a domain)

### Database Connectivity and Local Connection Strings

The live E2E demo proved application-level DB interaction, and a follow-up
Project-component hostname test clarified what hostname forms work on Scaleway:

- The standalone E2E demo used `Postgres` plus standalone `Service` resources.
- Because those standalone resources did not share a provider-created
  `PrivateNetwork`, the API connected to the managed database through the
  managed database endpoint output, not through a Compose-style hostname like
  `db`.
- The successful standalone E2E connection string used the provider output shape
  `postgres://defang:<password>@<resolved-host>:<resolved-port>/defang?sslmode=require`.
- In the Project component path, the provider creates shared private network
  infrastructure and attaches managed Postgres and Serverless Containers to it.
- A live Project-component test deployed two public API containers against the
  same managed Postgres instance:
  - `DATABASE_URL=postgres://defang:<password>@db:5432/defang?sslmode=require`
    returned HTTP 500 with `lookup db on 169.254.169.254:53: no such host`.
  - `DATABASE_URL=postgres://defang:<password>@db.demo.internal:5432/defang?sslmode=require`
    returned HTTP 200 and inserted/read a row from Postgres.

**Conclusion:** Scaleway does not provide bare Docker Compose-style service-name
DNS for managed database resources. The working private DNS form is the Scaleway
VPC form `<resource-name>.<private-network-name>.internal` (or the equivalent
UUID-based forms). For Defang's local/cloud connection-string value prop, the
Scaleway provider should preserve the application-facing environment variable
shape by rewriting/injecting a cloud connection string that points at Scaleway's
private DB hostname/IP. It should not expect the literal local hostname `db` to
resolve in Scaleway Serverless Containers.

### Live Testing

- Actual Scaleway deployment with full credentials succeeded on 2026-05-08 using the filesystem Pulumi backend:
  - Created namespace, private network, managed PostgreSQL instance, database, privilege, public web container, and private worker container
  - Verified the public web endpoint returned HTTP 200
  - Destroyed all 12 temporary resources after validation
- End-to-end demo deployment also succeeded on 2026-05-08:
  - Published temporary public API/web images with `ko` because Docker was unavailable in the workspace
  - Created standalone managed PostgreSQL plus standalone API and web Serverless Containers
  - Deployed in phases: DB first, then API with `DATABASE_URL`, then web with `API_URL`
  - API used the managed Postgres endpoint from `db.connectionUrl`; it did not use hostname `db`
  - Verified direct API response inserted/read a Postgres hit count
  - Verified public web response called API, API wrote to Postgres, and the rendered page showed `API status: 200` plus the incrementing database hit count
  - Destroyed all 11 temporary resources and removed the local Pulumi stack metadata
- Project-component Postgres hostname validation also succeeded on 2026-05-08:
  - Deployed managed Postgres plus two public API containers on the same
    provider-created private network
  - Confirmed bare Compose-style hostname `db` does **not** resolve from
    Scaleway Serverless Containers
  - Confirmed Scaleway internal DNS hostname `db.demo.internal` resolves and
    reaches the managed database over the private network
  - Destroyed all 12 temporary resources and removed the local Pulumi stack metadata
- Redis private network connectivity from a container
- Container scaling behavior (minScale 0 → cold start latency)
- Custom domain attachment and DNS verification

### Known Limitations

- Build-from-source not supported (requires pre-built images)
- LLM/Managed Inference not implemented
- No CD state upload (like GCP's GCS state URL)
- DNS/load-balancer abstraction not comparable to GCP
- **Container-to-container private networking not supported** (Scaleway limitation, egress-only PN)
- Cold starts may be slightly longer when private network is attached (IP booking overhead)

### Credentials Status

- `SCALEWAY_DEV_API_KEY` available (secret key / UUID format, authenticates against Scaleway API)
- `SCALEWAY_DEV_PROJECT_ID` and `SCALEWAY_DEV_ORG_ID` available in the workspace
- `SCW_ACCESS_KEY` was discoverable as non-secret metadata via Scaleway IAM API key listing
- Live Pulumi runs used:
  - `SCW_ACCESS_KEY`
  - `SCW_SECRET_KEY=$SCALEWAY_DEV_API_KEY`
  - `SCW_DEFAULT_PROJECT_ID=$SCALEWAY_DEV_PROJECT_ID`
  - `SCW_DEFAULT_ORGANIZATION_ID=$SCALEWAY_DEV_ORG_ID`
