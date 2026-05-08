# Scaleway end-to-end demo

This demo validates a real request chain on Scaleway:

```text
public web service -> public api service -> managed PostgreSQL
```

The API service creates a `demo_hits` table if needed, inserts one row for each
request, and returns the current row count. The web service calls the API and
renders the API response.

## Why this is not a Compose build example

The Scaleway provider currently requires pre-built images. Build-from-source is
not implemented for Scaleway Serverless Containers in this PR, so publish the
two images first and pass their image references to the Pulumi example.

## Publish images without Docker

The live validation used `ko`, which can build and publish Go images without a
local Docker daemon:

```bash
go install github.com/google/ko@latest

cd examples/scaleway-e2e-demo

KO_DOCKER_REPO=<registry>/<api-image-repo> ko publish ./cmd/api --bare
KO_DOCKER_REPO=<registry>/<web-image-repo> ko publish ./cmd/web --bare
```

Use a registry that Scaleway can pull from. The live test used temporary public
`ttl.sh` image references.

## Deploy

The Pulumi YAML program in `examples/scaleway-yaml` defines:

- `db` is a standalone `defang-scaleway:index:Postgres`
- `api` reads `DATABASE_URL` from Pulumi config
- `web` reads `API_URL` from Pulumi config

Scaleway Serverless Container Private Networks are egress-only. Containers can
reach managed databases on a private network, but Serverless Containers cannot
receive inbound private traffic from other Serverless Containers. That means
container-to-container traffic must use public HTTPS endpoints, which is why
`web` calls the API service's public endpoint.

The live E2E validation proved that the API can write to and read from managed
Postgres. It used the `db.connectionUrl` provider output, not a local-style
hostname such as `db`.

A separate live Project-component hostname test validated Scaleway private DNS
behavior. Bare Compose-style hostname `db` did not resolve from a Serverless
Container (`lookup db on 169.254.169.254:53: no such host`). The Scaleway
internal DNS form `db.demo.internal` did resolve and the API returned HTTP 200
after writing to Postgres. For Scaleway, the provider should inject/rewrite the
cloud connection string to use the provider-generated private DB endpoint rather
than expecting the literal local hostname `db` to work in the cloud.

Do not implement this by trying to force `db` DNS inside Scaleway Serverless
Containers. Scaleway does not expose a Compose-style DNS alias mechanism for
managed database resources, and container-to-container private networking is not
available for a private DNS/proxy workaround. Provider-managed connection string
translation is the intended compatibility layer.

Current `Service.environment` inputs are plain strings, so this demo is deployed
in phases: create DB, set `DATABASE_URL`, create API, set `API_URL`, then create
web. The live validation followed this flow.

```bash
cd examples/scaleway-yaml

pulumi login file:///tmp/pulumi-scaleway-state
pulumi stack init scaleway-e2e

pulumi config set --secret POSTGRES_PASSWORD 'Secret123!'
pulumi config set API_IMAGE '<api image ref>'
pulumi config set WEB_IMAGE '<web image ref>'

pulumi up

pulumi config set --secret DATABASE_URL "$(pulumi stack output databaseUrl --show-secrets)"
pulumi up

pulumi config set API_URL "$(pulumi stack output apiEndpoint)"
pulumi up
```

After deployment, open the `webEndpoint` output. A successful response includes:

- `API status: 200`
- `Database hit count: <n>`
- JSON from the API saying it wrote a row to Postgres and read the count

## Cleanup

```bash
pulumi destroy
pulumi stack rm scaleway-e2e
```
