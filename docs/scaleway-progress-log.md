# Scaleway Progress Log

This draft-only log tracks cleanup, implementation work, live validation, failures, and follow-up notes for the Scaleway PRs:

- `DefangLabs/defang#2105`
- `DefangLabs/pulumi-defang#234`

It should be kept current during SAM-driven development and removed or converted into user-facing docs before the PR is marked ready for merge.

## 2026-05-11

### Starting State

- CLI branch checked out locally: `feat/scaleway-byoc`
- CLI PR head: `fb1994ffb784064508d556389fc692929e6c9c69`
- Pulumi branch checked out locally: `sam/httpsgithubcomdefanglabspulumi-defang-deep-dive-01kr1k`
- Pulumi PR head: `a4aa8ed0c84cbe817487dd0821acc864ef1d0afc`
- Both PRs are mergeable drafts.
- Verified the actual GitHub PR diff for CLI #2105 is Scaleway-scoped; earlier concern about Azure/shared noise came from comparing against the pinned submodule commit instead of the PR merge base.

### Scaleway Account Cleanup

Bootstrapped standard env from SAM-provided variables:

- `SCALEWAY_DEV_API_KEY` -> `SCW_SECRET_KEY`
- `SCALEWAY_DEV_PROJECT_ID` -> `SCW_DEFAULT_PROJECT_ID`
- `SCALEWAY_DEV_ORG_ID` -> `SCW_DEFAULT_ORGANIZATION_ID`
- `DEFANG_TOKEN` -> `DEFANG_ACCESS_TOKEN`
- Derived `SCW_ACCESS_KEY` from the Scaleway IAM API using the secret key as `X-Auth-Token`.

Initial resource inventory found:

- 4 stale `defang-cd` Serverless Job definitions.
- `mastra-extended` Serverless Container namespace with `app` and `worker` containers.
- `funcscwmastraextended...` registry namespace with app/worker/cache images.
- One managed Postgres instance named `postgres`.
- One private network named `mastra-extended`.
- Two Cockpit tokens: `debug-logs-*` and `defang-cd-logs`.

Cleanup actions completed:

- Deleted `app` and `worker` Serverless Containers.
- Deleted `mastra-extended` Serverless Container namespace.
- Deleted app/worker/cache registry images and app registry namespace.
- Submitted delete request for managed Postgres instance.
- Deleted Cockpit tokens.
- Deleted four stale `defang-cd` job definitions.

Pending:

- Retry private network deletion after managed Postgres deletion fully drains. Initial attempt returned `resource_still_in_use`.

Preserved intentionally for now:

- `defang-cd` registry namespace and images. It currently contains the CD/Kaniko images needed to rebuild and test. This should be replaced/re-tagged as part of the CD image work rather than left stale.

### Implementation Pass 1

Scaleway API research:

- Current Serverless Jobs API is `v1alpha2`.
- `command` is deprecated in favor of `startup_command` and `args`.
- Job secret injection is now supported through Serverless Jobs secret references backed by Secret Manager. The reference is created after the job definition and points either to an env var name or a file path.

Code changes made:

- CLI `projects/defang`:
  - Switched the Scaleway Jobs client from `v1alpha1` to `v1alpha2`.
  - Updated job run parsing for the `{"job_runs":[...]}` start response.
  - Added Secret Manager-backed job secret references for CD job env values:
    - `AWS_SECRET_ACCESS_KEY`
    - `SCW_SECRET_KEY`
    - `PULUMI_CONFIG_PASSPHRASE`
  - Recreate the CD job definition during setup when a stale one already exists, so the image/env/secret refs match the current run.
  - Auto-create Defang config `OPENAI_API_KEY` from the Scaleway API key when a Compose model service has been fixed up to Scaleway Managed Inference and the config is missing.
  - Expanded CD teardown to delete CD job definitions, CD Secret Manager entries, registry images/namespace, and the CD S3 bucket contents/bucket.
- Pulumi provider `projects/pulumi-defang`:
  - Switched Kaniko build jobs to `v1alpha2` `startup_command`/`args`.
  - Moved Kaniko secrets (`AWS_SECRET_ACCESS_KEY`, `DOCKER_CONFIG_JSON`) into Secret Manager-backed job secret references.
  - Added cleanup for temporary build secrets.
  - Added the requested TODO beside the runtime-dependent worker health shim.

Validation status:

- Installed Go tooling in the workspace because it was missing from the base image.
- Used Go 1.25.9 for tests to match `pulumi-defang/go.mod`.
- Passed: `projects/defang/src`: `go1.25.9 test ./pkg/clouds/scaleway ./pkg/cli/compose ./pkg/cli/client/byoc/scaleway`
- Passed: `projects/pulumi-defang`: `go1.25.9 test ./provider/defangscaleway/...`

### Live Validation and Follow-up Fixes

Additional cleanup:

- Deleted a stale Redis cluster that was holding the private network through IPAM.
- Private network deletion still fails because Scaleway IPAM reports a phantom `serverless_container` attachment with an all-zero resource ID. No containers or namespaces remain for that attachment; this appears to require Scaleway-side cleanup.

CD image work:

- Installed Docker in the workspace and built/pushed test CD images to `rg.fr-par.scw.cloud/defang-cd/cd`.
- `pulumi plugin install resource defang-scaleway -f ...` rejects prerelease provider versions such as `2.0.0-beta.5`, so the Scaleway CD image now lays out the local provider plugin manually under `/root/.pulumi/plugins/resource-defang-scaleway-v${PROVIDER_VERSION}`.
- The Scaleway CD job runtime now uses `PULUMI_HOME=/root/.pulumi` so Pulumi sees the bundled plugins in the scratch CD image.
- Latest validated test image: `rg.fr-par.scw.cloud/defang-cd/cd:sam-20260511d`, digest `sha256:f515f0387b57b001b5b91e08d4fbf854fac7847327efa6d9f08913c35b5f14aa`.

CLI fixes driven by live testing:

- Treat first deploy as missing state when the CD bucket or `project.pb` key is absent.
- Treat corrupt `project.pb` as missing state so a redeploy can recover from partial or malformed state writes.
- Add required `local_storage_capacity` to `v1alpha2` CD job definitions.
- Delete all existing `defang-cd` job definitions before creating the current one. Scaleway permits duplicate job names, and stale definitions caused runs to use old images/env.
- Fetch the real Cockpit logs data-source URL from the Cockpit API instead of guessing the hostname.
- Create Cockpit query tokens with `token_scopes: ["read_only_logs"]`; the older boolean `scopes` object creates empty-scope tokens on the current API.

CD/provider fixes driven by live testing:

- Added Scaleway `project.pb` upload after successful deployment using a Pulumi-managed `ObjectItem`, gated on the project resource. This restores `compose ps` parity with AWS/GCP/Azure readback.
- The first attempt used `contentBase64`; Scaleway stored data in a form the CLI could not unmarshal. The final version writes a temporary binary file and passes `file` plus a SHA-256 `hash`.

Live deploy result:

- Deployed a one-service Python app with a Compose `models:` entry backed by Scaleway Managed Inference.
- Verified first-class LLM auth: the CLI auto-created `OPENAI_API_KEY` config from the Scaleway API key, the container received it, and `/llm` successfully queried Scaleway's OpenAI-compatible `/models` endpoint.
- Verified native Scaleway endpoint:
  - `/` returned `{"ok": true, "model": "llama-3.3-70b-instruct"}`
  - `/llm` returned `{"ok": true, "count": 15}`
- Verified `compose ps` after `project.pb` fix:
  - service `app`
  - state `DEPLOYMENT_COMPLETED`
  - health `healthy`
- Verified `compose logs` no longer fails on Cockpit endpoint/auth errors. The test app did not emit useful log lines in the short query window, but the query path returned successfully.
- Verified `compose down` removed the live Serverless Container and namespace. A follow-up native API check showed no remaining Serverless Containers or namespaces for the test app.

Known limitations observed:

- Defang delegated domain DNS was not created because this Scaleway credential cannot create the `defang.app` DNS zone (`HTTP 403`). The native Scaleway serverless container domain worked.
- Explicit `mode: host` ports are rejected by the Scaleway provider because Serverless Containers expose HTTP ingress only. The live test used `mode: ingress`.
