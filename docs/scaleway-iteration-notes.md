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

6. **Enhanced example** (`examples/scaleway-yaml/Pulumi.yaml`)
   - Updated from single nginx service to multi-service: public web, private worker, managed postgres
   - Demonstrates all three deployment types in one Pulumi program

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
- But inter-service communication between Serverless Containers must go through public URLs
- Managed databases and Redis on the private network ARE reachable from containers (egress)
- The private network primarily serves database/Redis connectivity, not inter-service mesh

**Changes applied:**
- Removed explicit `ActivateVpcIntegration`; the Scaleway provider now treats VPC integration as always enabled
- All container endpoints use `https://` public URLs (even private containers get a domain)

### Live Testing

- Actual Scaleway deployment with full credentials succeeded on 2026-05-08 using the filesystem Pulumi backend:
  - Created namespace, private network, managed PostgreSQL instance, database, privilege, public web container, and private worker container
  - Verified the public web endpoint returned HTTP 200
  - Destroyed all 12 temporary resources after validation
- PostgreSQL private network endpoint resolution from a container
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
- Missing: `SCW_ACCESS_KEY`, `SCW_DEFAULT_PROJECT_ID`, `SCW_DEFAULT_ORGANIZATION_ID`
- Requested human input for full credential set
