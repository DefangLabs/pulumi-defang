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

**Decision for this PR:** do not try to make the literal hostname `db` work on
Scaleway. There is no platform-level knob for adding Docker Compose-style DNS
aliases to Serverless Containers, and container-to-container private networking
is unavailable. Workarounds such as mutating `/etc/hosts`, running a private
DNS/proxy service, or depending on user image entrypoints would be brittle and
would not match Defang's provider-level portability goal. The correct Scaleway
implementation path is provider-managed connection string translation: keep the
same env var contract for the application, but set the cloud value to Scaleway's
private database endpoint.

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

## 2026-05-10: Health Shim, Redis 8.4.0, and LLM End-to-End

### Changes Made

1. **Redis version updated to 8.4.0** (`provider/defangscaleway/scaleway/redis.go`)
   - Scaleway removed Redis 6.2.7 and 7.2.5; only 8.4.0 is available as of 2026-05
   - `redisVersionFromImage()` now always returns `"8.4.0"` regardless of compose image tag
   - Tests updated accordingly

2. **Health shim for portless background workers** (`provider/defangscaleway/scaleway/containers.go`)
   - Scaleway Serverless Containers **require** a listening HTTP port (documented requirement: "It must expose a webserver port and be listening on `0.0.0.0`")
   - Background workers (queue consumers, cron jobs) that don't expose ports previously failed with "Container is unable to start OR is not listening on port 8080"
   - Added automatic health shim injection: when a service has no ports but has a command defined, the provider wraps the command with a shell script that:
     1. Starts a tiny HTTP health responder on `$PORT` in the background
     2. Tries runtimes in order: `node`, `python3`, `python`, `nc` (shell+netcat fallback)
     3. Execs the original command as PID 1
   - Port is set to 8080 (Scaleway's default)
   - `Commands` becomes `["/bin/sh", "-c"]`, `Args` becomes the shim script
   - Helper functions: `needsHealthShim()`, `healthShimScript()`, `shellJoin()`
   - Full test coverage: `TestHealthShim`, `TestHealthShimInjectedInContainer`, `TestShellJoin`

### Shim Script Shape

For a service with `command: ["npm", "run", "worker"]` and no ports:

```sh
/bin/sh -c '(node -e "require('"'"'http'"'"').createServer((_,r)=>{...}).listen(process.env.PORT||8080)" 2>/dev/null || python3 -c "..." 2>/dev/null || ...) & exec npm run worker'
```

The shim uses a cascade of common runtimes so it works across Node.js, Python, Alpine, and other base images. The `exec` ensures the real command replaces the shell as PID 1 for proper signal handling.

### Critical: min_scale=1 for Portless Workers

Scaleway Serverless Containers only wake scaled-to-zero instances on **inbound HTTP requests**. Background workers that poll Redis/databases internally (like BullMQ consumers) never receive HTTP traffic, so they stay at zero instances permanently after scaling down. This was discovered when the mastra-extended worker container stopped processing queue jobs after scaling to zero.

**Fix:** `containerMinScale()` now returns 1 for services that need the health shim (no ports). This ensures they always have at least one running instance. Services with ports can still scale to zero since HTTP traffic will wake them.

### Mastra Extended End-to-End Validation

Deployed `samples/mastra-extended` (6-service compose) on Scaleway with full success:

| Service | Type | Status | Notes |
|---------|------|--------|-------|
| **app** | Serverless Container | ready | Next.js UI + Mastra agent API, port 3000 |
| **worker** | Serverless Container | ready | BullMQ queue consumer, health shim on port 8080 |
| **postgres** | Managed Database | ready | Scaleway Managed PostgreSQL |
| **redis** | Managed Redis | ready | Scaleway Managed Redis 8.4.0 |
| **chat** | Scaleway Generative API | N/A | `llama-3.3-70b-instruct` (CLI resolves, no container) |
| **embedding** | Scaleway Generative API | N/A | `bge-multilingual-gemma2` (CLI resolves, no container) |

**Validated flows:**
- App serves Next.js UI (HTTP 200)
- Data seeding: 20 items generated, all 20 classified via embedding model, 0 failures
- BullMQ queue: worker processes jobs from Redis queue correctly
- Chat API: LLM calls tools (`getTasks`, `getEvents`), returns structured responses
- Streaming: NDJSON stream with tool calls and text deltas working

### LLM Architecture on Scaleway

Unlike AWS (Bedrock + LiteLLM sidecar) and GCP (Vertex AI + LiteLLM sidecar), Scaleway uses **direct API access**:

- CLI strips `provider: type: model` services during compose fixup
- Dependent services receive env vars pointing to `https://api.scaleway.ai/v1/`
- No sidecar container deployed; authentication via `OPENAI_API_KEY` (user's Scaleway secret key)
- Model resolution: `chat-default` → `llama-3.3-70b-instruct`, `embedding-default` → `bge-multilingual-gemma2`

### Known Limitations

- Build-from-source not supported (requires pre-built images)
- No CD state upload (like GCP's GCS state URL)
- DNS/load-balancer abstraction not comparable to GCP
- **Container-to-container private networking not supported** (Scaleway limitation, egress-only PN)
- Cold starts may be slightly longer when private network is attached (IP booking overhead)
- Health shim for portless services requires at least `sh` in the container image; if the service has no explicit command AND the image ENTRYPOINT is unknown, the shim cannot be injected
- Scaleway Generative API model selection is hardcoded (no dynamic model discovery like Azure)

### Credentials Status

- `SCALEWAY_DEV_API_KEY` available (secret key / UUID format, authenticates against Scaleway API)
- `SCALEWAY_DEV_PROJECT_ID` and `SCALEWAY_DEV_ORG_ID` available in the workspace
- `SCW_ACCESS_KEY` was discoverable as non-secret metadata via Scaleway IAM API key listing
- Live Pulumi runs used:
  - `SCW_ACCESS_KEY`
  - `SCW_SECRET_KEY=$SCALEWAY_DEV_API_KEY`
  - `SCW_DEFAULT_PROJECT_ID=$SCALEWAY_DEV_PROJECT_ID`
  - `SCW_DEFAULT_ORGANIZATION_ID=$SCALEWAY_DEV_ORG_ID`
