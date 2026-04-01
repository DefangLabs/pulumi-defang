# CD Program: Missing from Go port

Compared against `~/dev/defang-mvp/pulumi/index.ts`.

## Bugs in Go code

- [ ] `main.go:218` — `stackConfig()` called with no args but signature requires `composePath string`
- [ ] `envs.go:18` — `awsRegion` references `region` (line 37) which is `""` at init time; `REGION` fallback never works

## Missing commands (from TS)

- [ ] `list`/`ls` — enumerate stacks from S3 backend (`listObjectsV2`, filters by `.json` size > 700 bytes)

## Missing config keys (set on stack in TS)

- [x] `aws:defaultTags` — `{defang:org, defang:project, defang:stack, defang:version}`
- [ ] `buildRepo` — `${projectName}/${repoPrefix}-build`
- [ ] `allowManaged` — always `"true"`
- [x] `createApexDnsRecord` — always `"true"`
- [ ] `skipNetworkAcl` — always `"true"` (allow SMTP in BYOC)
- [ ] `internalLogGroup` — `"true"` when debug is enabled
- [ ] `serviceDomain` — same as `domain`
- [ ] `dockerHubUsername` — from `DOCKERHUB_USERNAME`
- [ ] `dockerHubAccessToken` — from `DOCKERHUB_ACCESS_TOKEN`

## Missing env vars

- [ ] `DEFANG_BUILD_REPO`
- [ ] `DEFANG_PREVIEW` — preview-only mode for destroy/refresh
- [ ] `DOCKERHUB_USERNAME`
- [ ] `DOCKERHUB_ACCESS_TOKEN`

## Missing behavior

- [ ] **Hook up `delegationSetId` for Route53 zone**
- [ ] **Refresh on `up` failure** — TS catches `up` errors and does a forced refresh before rethrowing (skipped if abort signal or `ConcurrentUpdateError`); old GCP code did the same in `up.go`
- [ ] **Upload method** — TS uses `PUT`, Go uses `POST`; old GCP code used `PUT` with `retryablehttp`; verify which the portal expects
- [ ] **Retryable HTTP uploads** — old GCP code used `retryablehttp.NewClient()` with 30s timeout; new uses bare `http.Post` with no retries or timeout
- [ ] **Deployment timeout** — TS has 58-min soft timeout (`SIGXCPU`) + 60-min hard kill (`process.exit(137)`)
- [ ] **Output filtering** — TS suppresses Pulumi progress bars (`suppressProgress: true`) and filters noisy log lines containing: `` `node ``, `` `pulumi ``, `press ^C`, `grpc: the client connection is closing`, `will not show as a stack output`; old GCP code also used `SuppressProgress()` + `ErrorProgressStreams()`
- [ ] **Error normalization** — TS rewrites Pulumi lock error messages to suggest `defang cd cancel` instead of `pulumi cancel`; strips ANSI and stack traces; old GCP code had `normalizeError()` that stripped after first newline
- [ ] **`previewOnly` on destroy/refresh** — TS passes `previewOnly` flag from `DEFANG_PREVIEW` env var
- [ ] **Upload in `finally`** — TS always uploads events+state in a `finally` block (even on error); old GCP used deferred uploads in `RunCD()`; new Go only uploads on certain commands and not on error paths
- [ ] **`down` uploads events+state** — TS uploads via `finally`; Go's `down` does upload but only after destroy, not if refresh fails
- [ ] **Stack select vs upsert** — old GCP code used `selectStack()` (no program) for destroy/refresh/cancel, and `upsertStack()` (with program) only for up/preview; new always uses `UpsertStackInlineSource` for all commands
- [ ] **Preview always shows diffs** — old GCP code had `optpreview.Diff()` unconditionally; new makes it conditional on `DEFANG_PULUMI_DIFF`
- [ ] **Hook up `privateDomain`** - new AWS code ignores `privateDomain` config key
- [ ] **Only `UpsertStackInlineSource` for up/preview** — old code used `selectStack()` (no program) for destroy/refresh/cancel, and `upsertStack()` (with program) only for up/preview; new always uses `UpsertStackInlineSource` for all commands

## Missing Pulumi options on commands

- [ ] **`TargetDependents` on destroy** — old GCP `getDestroyOptions()` included `optdestroy.TargetDependents()` and `optdestroy.Target(pulumiTargets)`; new destroy has neither
- [ ] **`Target(pulumiTargets)` on refresh** — old GCP `getRefreshOptions()` passed targets; new refresh doesn't
- [ ] **`UserAgent` on destroy/refresh** — old GCP set `UserAgent("defang/"+version)` on all commands; new only sets it on up/preview

## Missing commands (from old GCP code)

- [ ] `cleanup` — old GCP had `Cleanup()` + `scheduleCleanUp()` that created a Cloud Build cron job to clean VPC resources after destroy

## Missing GCP-specific config

- [ ] `gcp:defaultLabels` — old GCP `upsertStack()` set `{defang-org, defang-project, defang-stack, defang-version}` labels

## Missing env vars (from old GCP code)

- [ ] `DEFANG_CD_IMAGE` — declared in `envs.go` but unused; old GCP used it (or extracted from protobuf) to schedule cleanup jobs
