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
