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
2. `pulumi stack export` it, then **import the state into a fresh stack under
   the new project name** — modeling the state relocation the CLI must do.
3. For each alias strategy, preview a model of the **new** stack shape
   (project = compose name, nested components) and tally the planned ops.

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

1. **State relocation is a prerequisite, not optional.** Aliases only remap
   URNs *within* one stack's state. Because the Pulumi project name changed
   (`cd` → compose name), the DIY-backend stack file lives at a different path;
   the CLI (or a migration step) must copy/rename the old stack's state into
   the new project's stack before the first `up`. The harness models this with
   export/import.
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

Extend the existing `x-defang-aliases` mechanism with a convention mode: when
migration from the old CD is detected (or requested via recipe/extension),
have each `Create*` worker attach a computed `pulumi.Alias` spec (finding 2)
for the resources the old CD created, keyed off the old naming rules. The
explicit URN map stays as the escape hatch for resources whose old names are
not derivable.
