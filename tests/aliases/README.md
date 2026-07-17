# Alias-migration harness

Offline test scenario for evaluating strategies to add `pulumi.Aliases` to the
providers so that stacks deployed by the **old defang-mvp CD** keep their cloud
resources — instead of delete+create — when redeployed with **this repo's
providers**. Relates to the `x-defang-aliases` extension
(`provider/compose/aliases.go`) and the MVP-parity tracking issue (#81).

## What changes between old and new URNs

| | old (defang-mvp `pulumi/cd`) | new (this repo) |
|---|---|---|
| Pulumi project | fixed `cd` (`Pulumi.yaml`) | the compose project name (`cd/cd_main.go`) |
| Topology | bootstrap under a `defang-mvp:shared/ecs/defang:Defang` component; per-service resources flat at the stack root | everything nested under `defang-aws:index:Project` / `…:Service` / `…:Redis` components |
| Logical names | `safe_namings` truncation | mostly stable names + `pulumi:autonaming` for physical names |

Every URN therefore changes at the *project* segment, and most also change
their *parent-type* chain. Without aliases the first `up` after migrating
plans a full delete+create of the world (verified below).

## How the harness works

`scenario_test.go` runs the **real Pulumi engine** against a throwaway
`file://` backend, with `random.RandomString` standing in for cloud resources
(alias resolution is provider-independent), so it needs no credentials:

1. Deploy a model of the **old** stack shape (project `cd`): a `Defang`
   component owning `vpc`/`cluster`, plus four flat per-service resources.
2. For each alias strategy, select the **same stack from the same backend**
   under the new project name and preview a model of the **new** stack shape
   (project = compose name, nested components), tallying the planned ops.

The backend is pinned to the legacy (pre-project-scoped) DIY layout via
`PULUMI_DIY_BACKEND_LEGACY_LAYOUT`, matching the defang CD buckets: the stack
file path has no project segment (`.pulumi/stacks/<stack>.json`), so the new
program reads the old project's state **in place** — the project rename does
not move the state file. (A backend created with the modern project-scoped
layout would shelve the old state under `stacks/cd/` and need a copy step;
that's not our situation.)

Run it from this directory (needs the `pulumi` CLI; skipped under `-short`,
so CI's provider-test run is unaffected):

```sh
go test -v .
```

## Strategies and results

7 old resources; the new shape has 9 (the `Service`/`Redis` components are new
and virtual, so 2 creates is the floor). Ops exclude the stack and provider
pseudo-resources.

| mode | aliases attached | result |
|---|---|---|
| `none` | — | create 9, **delete 7** — the world is replaced |
| `urn` | explicit old URN per resource (generalized `x-defang-aliases`) | **same 7**, create 2, delete 0 |
| `spec` | `pulumi.Alias{Name/Type/Project/ParentURN/NoParent}` computed from the old naming convention | **same 7**, create 2, delete 0 |
| `parent-inherit` | alias only on the components; children rely on SDK inherited aliases | same 1 (the component), **children all recreated** |
| `parent+project` | like `parent-inherit` plus `Alias{Project: "cd"}` on children | same 1, **children all recreated** |

## Findings

1. **The state file is found in place.** With the legacy DIY layout the CD
   buckets use, the stack file path doesn't include the project name, so the
   renamed project reads the old state directly — no relocation or import
   step. Aliases are the only thing needed. (Keep the legacy layout pinned in
   the cd driver; a layout upgrade would move stack files under per-project
   directories.)
2. **Per-resource aliases work — both flavors.** Explicit URNs (`urn`) and
   convention-computed specs (`spec`) both achieve zero deletes. `spec` is the
   interesting one for automation: `Alias{Project: "cd", NoParent: true}` for
   previously-flat resources, plus `Type`/`Name`/`ParentURN` when those
   changed, can be emitted mechanically by provider code that knows the old
   layout — no user-supplied URN inventory needed.
3. **Component-level aliases don't propagate usefully across a project
   rename.** The SDK's inherited-alias computation builds child URNs with the
   *current* project name, so aliasing only the parent component leaves every
   child unmatched. Adding a bare `Alias{Project}` on children doesn't compose
   either — the child's alias still resolves against its *new* parent's URN.
   Conclusion: each child needs its own complete alias; there is no cheap
   parent-only shortcut.
4. **Aliases must be attached inside the provider.** Our components are remote
   (MLC) components: resource options passed by the calling program do not
   propagate to children registered in the provider process (already noted in
   `provider/compose/aliases.go`). So whichever flavor is chosen, the wiring
   lives in the `Create*` workers — as `x-defang-aliases` does today for the
   Redis child kinds.

## Suggested direction

Attach convention-computed `pulumi.Alias` specs (finding 2) **by default** in
the `Create*` workers — but scoped to **stateful resources only**. Full
zero-replacement fidelity isn't the goal: stateless resources (task
definitions, ECS/Cloud Run services, listeners, target groups, log groups,
certs, IAM, builds) may recreate freely on migration. Aliases resolve purely
in the engine against current state: inert when no old URN is present,
permanent no-ops once the first post-migration `up` rewrites the URNs, and a
wrong alias fails safe (no match → the replacement you'd have had anyway).
The explicit `x-defang-aliases` URN map stays as the per-resource override,
and a recipe kill-switch covers debugging.

Starter set of stateful resources to alias:

- **AWS**: VPC + subnets, NAT EIPs (customers allowlist these — the old CD
  exported `publicNatIps`), RDS instance + DB subnet group, ElastiCache
  replication group / MemoryDB cluster + subnet group + parameter group,
  Route53 zones (the public zone's NS delegation is externally pinned),
  the ALB (its DNS name changes on recreation and customers CNAME to it),
  ECR repositories and S3 buckets (access logs, build context — deleting
  either **fails outright while non-empty**, stranding the update mid-way).
- **GCP**: network + subnet, the global static IP (the LB address customers
  point DNS at), Cloud SQL instance + service-networking connection/reserved
  range, Memorystore instance, the public DNS managed zone, Artifact Registry
  repositories and GCS buckets (same non-empty-delete failure class as
  ECR/S3).

Two subtleties the stateful-only cut must respect:

1. **ForceNew attachments ride along.** A kept database pins whatever its
   replace-forcing inputs reference — e.g. an ElastiCache/RDS instance whose
   subnet group changes gets replaced anyway, and a subnet group referencing
   recreated subnets drags the database with it. So the stateful set is
   closed under "referenced by a ForceNew property": DB → subnet group →
   subnets → VPC. (This is exactly the shape of the existing
   `x-defang-aliases` kinds: cluster / subnet-group / parameter-group /
   security-group.)
2. **An unmatched stateful resource isn't just "recreated".** Three failure
   classes, in decreasing severity: databases come back **empty** (the engine
   creates the new one and deletes the old in the same update — silent data
   loss, or a deletion-protection error mid-deploy); ECR/Artifact Registry
   repositories and S3/GCS buckets **fail to delete while non-empty**,
   stranding the update half-migrated; and pinned addresses/hostnames (NAT
   EIPs, the global IP, the ALB DNS name, zone NS records) break whatever
   customers configured externally. Databases are the hard core of the set;
   the rest decides whether migration is one clean `up` or a support ticket.
