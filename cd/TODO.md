# CD Program: Missing from Go port

Compared against `~/dev/defang-mvp/pulumi/index.ts`.

## Refactoring

- [ ] **Migrate `BuildResult` to `pulumix` types** — `common.BuildResult.LoadBalancerDNS` uses `pulumi.StringPtrOutput`; switching to `pulumix.Output[*string]` would simplify nil-output initialization (e.g. `pulumix.Val[*string](nil)` instead of `.Untyped()` cast)
- [ ] **Support `${VAR:-default}` with soft-fail config lookup** — `ConfigProvider.GetConfigValue` errors on missing keys, so `pulumi.All()` fails before compose-go's `:-` default can kick in. Needs a `TryGetConfigValue` (or similar) that returns empty instead of erroring. Would also enable `${VAR:?msg}` to produce custom error messages for missing keys. Expressions that already work: `$VAR`/`${VAR}`, `${VAR:+repl}`, `${VAR+repl}`, `$$` escaping.

## Missing config keys (set on stack in TS)

- [x] `aws:defaultTags` — `{defang:org, defang:project, defang:stack, defang:version}`
- [ ] `buildRepo` — `${projectName}/${repoPrefix}-build`
- [x] `allowManaged` — always `"true"` - deprecated
- [x] `createApexDnsRecord` — always `"true"`
- [x] `skipNetworkAcl` — always `"true"` (allow SMTP in BYOC) - deprecated
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

- [ ] **Hook up `delegationSetId` for AWS Route53 zone** — `cd/config.go` now plumbs `DELEGATION_SET_ID` → `defang-aws:delegationSetId` config, but the AWS provider's `AWSConfig` (`provider/defangaws/aws/config.go`) doesn't read it yet, so the zone still ignores it.
- [ ] **Refresh on `up` failure** — TS catches `up` errors and does a forced refresh before rethrowing (skipped if abort signal or `ConcurrentUpdateError`); old GCP code did the same in `up.go`. Still a `TODO` at `cd_main.go:166`.
- [x] **Upload method** — TS uses `PUT`, Go uses `POST`; old GCP code used `PUT` with `retryablehttp`; verify which the portal expects. Done: `doUpload` uses `http.MethodPut`.
- [x] **Retryable HTTP uploads** — `upload.go` `doUpload` uses `retryablehttp.NewClient()` with a 30s per-attempt timeout.
- [x] **Deployment timeout** — `main.go` sets a 60-min `context.WithTimeoutCause` that fires `SIGXCPU` (exit 152). Note: single-stage only; the TS 58-min soft + 60-min hard-kill split was not reproduced.
- [x] **Output filtering** — all commands pass `SuppressProgress()` + `ProgressStreams`/`ErrorProgressStreams`; `log.go` filters noisy lines (`` `pulumi ``, `press ^C`, `grpc: the client connection is closing`, `will not show as a stack output`). Note: the TS `` `node `` filter was dropped.
- [ ] **Error normalization** — done: `errors.go` `pulumiErr` strips the auto SDK code/stdout/stderr dump down to the `error:` block (and recovers the nested exit code). Remaining: TS also rewrites Pulumi lock errors to suggest `defang cd cancel` instead of `pulumi cancel` — `pulumiErr` currently passes the lock message through verbatim (see `errors_test.go`).
- [ ] **`previewOnly` on destroy/refresh** — TS passes `previewOnly` flag from `DEFANG_PREVIEW` env var. Not implemented.
- [x] **Upload in `finally`** — `cd_main.go` calls `uploadState` unconditionally before the error check on up/destroy/refresh, and events upload via `defer waitEvents()`, so both upload on error paths too.
- [x] **`down` uploads events+state** — `down`/`destroy` share a path that uploads state at `cd_main.go:215` plus deferred events.
- [x] **Stack select vs upsert** — `cd_main.go` uses `UpsertStackInlineSource` only for up/preview and `SelectStackInlineSource` for destroy/down/refresh/cancel/outputs/list.
- [x] **Preview always shows diffs** — old GCP code had `optpreview.Diff()` unconditionally; new makes it conditional on `DEFANG_PULUMI_DIFF`
- [ ] **Hook up `privateDomain`** — `cd/config.go` plumbs `PRIVATE_DOMAIN` → `defang-aws:privateDomain`, but `AWSConfig` doesn't read it; the provider still computes its own `<project>.internal` (`networking.go:28`).
- [ ] **Azure Key Vault client scoping** — `fetchFromKeyVault` in `provider/defangazure/azure/userconfig.go` uses `azidentity.DefaultAzureCredential` + string-built vault URL, ignoring Pulumi's configured `azure-native:tenantId`/`subscriptionId`. Silent divergence if the vault lives in a different subscription than the Pulumi provider. Fix: pass `TenantID` from Pulumi config to the credential, and derive the vault URL via `keyvault.LookupVault` (parented) so mismatches fail loudly.
- [x] **Only `UpsertStackInlineSource` for up/preview** — old code used `selectStack()` (no program) for destroy/refresh/cancel, and `upsertStack()` (with program) only for up/preview; new always uses `UpsertStackInlineSource` for all commands

## Missing Pulumi options on commands

- [x] **`TargetDependents` on destroy** — done: destroy passes `optdestroy.TargetDependents()` + `optdestroy.Target(pulumiTargets)`; a targeted destroy also skips `optdestroy.Remove()` so partial state isn't orphaned
- [x] **`Target(pulumiTargets)` on refresh** — done: refresh passes `optrefresh.Target(pulumiTargets)`
- [x] **`UserAgent` on destroy/refresh** — old GCP set `UserAgent("defang/"+version)` on all commands; new only sets it on up/preview

## Missing commands (from old GCP code)

- [ ] `cleanup` — old GCP had `Cleanup()` + `scheduleCleanUp()` that created a Cloud Build cron job to clean VPC resources after destroy

## Missing GCP-specific config

- [x] `gcp:defaultLabels` — old GCP `upsertStack()` set `{defang-org, defang-project, defang-stack, defang-version}` labels

## Missing env vars (from old GCP code)

- [x] `DEFANG_CD_IMAGE` — no longer unused: `config.go` reads it into `defang:cdImage`. The consumer (GCP cleanup cron) is still missing — tracked under `cleanup` above.

---

# Inline TODO/FIXME markers in code

Captured from `grep -rn -E "(TODO|FIXME):"` across `cd/` and `provider/`. These are in-code design notes, not part of the TS port comparison above; listed here so they're tracked in one place.

## `cd/`

- [ ] `cd_main.go:77` — what to do when Compose project name doesn't match `PROJECT` env var
- [ ] `cd_main.go:106` — `USER`-in-lock-file hack for debugging doesn't work on Linux
- [ ] `cd_main.go:166` — run a refresh on `up` failure to capture partial changes (also tracked above)
- [ ] `cd_main.go:215` — `uploadState` after destroy prints a warning even on success
- [ ] `upload.go:74` — skip gzip compression for small bodies
- [ ] `config.go:126,134,137` — sanitize project name in GCP autonaming patterns
- [ ] `config.go:194` — configure label-logger (GCP)
- [ ] `program/aws.go:26`, `program/gcp.go:24` — `defang:org`/`defang-org` label doesn't work with DIY backends
- [ ] `program/gcp.go:27` — `defang-version` label can't contain dots
- [ ] `program/azure.go:74` — support multiple endpoints per service
- [ ] `program/azure.go:76` — support private FQDNs

## `provider/compose`, `provider/common`

- [ ] `compose/types.go:24` — `MapOrList` could be `string[]` but normalized to map by compose-go
- [ ] `compose/types.go:377` — `POSTGRES_PASSWORD` should not default to `""`
- [ ] `compose/helpers_test.go:138` — make `${VAR:?err}` work (#159)
- [ ] `common/build.go:75` — get actual etag from URL, not path (strip sig query param)

## `provider/defangaws`

- [ ] `project.go:195` — detect sidecar services (`network_mode: "service:<name>"`, `volumes_from`)
- [ ] `aws/ecs.go:430` — index is `[1]` if there's no loggroup
- [ ] `aws/ecs.go:520` — which service should listen on the project domain?
- [ ] `aws/codebuild.go:56` — revisit defaulting build arch to x86_64
- [ ] `aws/codebuild.go:188` — separate logGroup for builds
- [ ] `aws/parameters.go:28` — customizable prefix
- [ ] `aws/cert.go:71` — filter out `RetainOnDelete` from opts
- [ ] `aws/infra.go:103` — look up `publicZoneId`?
- [ ] `aws/infra.go:113` — only pick CAs that we need (CAA issuers)
- [ ] `aws/iam.go:96` — restrict IAM resource `*` to project-scoped ARNs
- [ ] `aws/elasticache.go:329` — allow user to set `authToken` via config
- [ ] `aws/networking.go:30` — missing type checking on NAT gateway strategy
- [ ] `aws/networking.go:104` — make NAT gateway optional to save cost
- [ ] `aws/route53.go:102` — support `iodef`, etc. in CAA records
- [ ] `aws/lb.go:230` — path-based routing
- [ ] `aws/ecr.go:57` — hashTrim/truncate prefix smartly

## `provider/defanggcp`

- [ ] `project.go:103` — create dependency between `NewService` and services that need the API
- [ ] `project.go:262` — add dependency to the member resource
- [ ] `build.go:43` — use ETAG from object metadata as digest in `Diff`
- [ ] `build.go:112` — implement secrets with global `availableSecrets` + per-step `secretEnv`
- [ ] `build.go:114` — inline secret for env vars to build steps
- [ ] `build.go:126` — support NPM/Python packages via `Artifacts` field
- [ ] `gcp/parameters.go:26` — customizable prefix
- [ ] `gcp/alb.go:248,359` — can the LB-backend source range be stricter than `0.0.0.0/0`?
- [ ] `gcp/alb.go:529` — internal ALB only supports HTTP traffic
- [ ] `gcp/compute.go:90` — `BaseInstanceName` resource doesn't support autonaming
- [ ] `gcp/compute.go:374` — pass all env/secrets as env instead of flattening into the command line
- [ ] `gcp/gcp.go:282` — avoid creating Pulumi resources within `ApplyT`
- [ ] `gcp/image.go:47` — 63-char name truncation could collide
- [ ] `gcp/algo.go:9` — could depend on deployment mode + number of services

## `provider/defangazure`

- [ ] `project.go:383` — don't set `KeyVaultURL` if the vault doesn't exist
- [ ] `project.go:516` — no LB in Azure (`loadBalancerDNS` hardcoded `""`)
- [ ] `azure/azure.go:46` — `defang-org` tag doesn't work with DIY backends
- [ ] `azure/containerapp.go:252` — need top-level networks to decide whether `default` is internal
- [ ] `azure/containerapp.go:447` — support more than one ingress port
- [ ] `azure/image.go:228` — handle image push when `svc.Image` is specified alongside build
