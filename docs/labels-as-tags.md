# Compose labels → cloud tags/labels

This document describes how Docker Compose `labels` are propagated to the
cloud resources each service/network produces, and the design decisions and
gotchas behind the implementation. It is a contributor/design reference, not
user-facing provider documentation.

## Goal

A Compose project can attach `labels` to a service or a network:

```yaml
services:
  web:
    image: nginx
    labels:
      com.acme.team: core
      com.acme.env: prod
networks:
  default:
    labels:
      com.acme.cost-center: "1234"
```

These labels must flow **transitively** onto every cloud resource that the
service (or the default network's shared infrastructure) creates:

- **AWS** → resource `Tags` (a `map[string]string`)
- **Azure** → resource `Tags`
- **GCP** → resource `Labels` (with sanitization — see below)

`ServiceConfig.Labels` and `NetworkConfig.Labels` are both
`compose.MapOrList[string]` (compose-go normalizes the list form
`["k=v"]` to a map). See `provider/compose/types.go`.

## Core mechanism: resource transformations, not providers

We use a Pulumi `ResourceTransformation` threaded through resource `opts`, **not**
a per-service provider with `defaultTags`.

- **Why not per-service providers.** A resource's provider is part of its
  identity. Giving each service its own provider (to leverage the AWS
  provider's `defaultTags`) risks **replacing every existing resource** on the
  next deploy of every existing stack. Transformations only modify *inputs* at
  registration time and never force replacement.
- **Transformations propagate to children.** The codebase threads `opts` into
  every resource-creation call (`pulumi.Parent(...)`, `childOpts...`,
  `pulumi.Composite`). Attaching the transformation once on the per-service or
  per-network `opts` reaches all downstream resources — *including* resources
  created under sub-providers (e.g. a cross-account Route 53 provider for DNS),
  which a provider-based approach would miss. This is decisive.

This mirrors the pattern already used by the Azure provider's
`DefaultTagsTransformation` (`provider/defangazure/azure/azure.go`), which
cascades the project-wide `defang-*` tags to every azure-native resource.

The TypeScript source of truth (`defang-mvp/pulumi/shared/aws/tags.ts`)
implements the same idea for AWS via `labelTagsTransformation`.

## Taggability: the sharp edge

You cannot blanket-add tags/labels to *every* resource — sending a `tags`
input to a type that has no such input is a **hard deploy failure**. The
transformation must therefore only touch resources that accept the field.

### Go advantage: reflection over an allowlist

The TS implementation maintains a hand-curated **allowlist** of taggable type
tokens, because in JS the resource `props` is an untyped object and there is no
way to ask "does this type accept tags?".

In Go we have typed `Args` structs, so the transformation inspects the props by
reflection and **only sets the field when it actually exists** with the right
type:

```go
f := v.FieldByName("Tags")          // or "Labels" for GCP
if !f.IsValid() || !f.CanSet() { return nil }
if !f.Type().Implements(stringMapInputType) && f.Type() != stringMapInputType {
    return nil
}
```

This is strictly more robust than an allowlist:

- An unrecognized / non-taggable type is silently skipped (graceful
  degradation), exactly like an allowlist — but with **no list to maintain**.
- It correctly excludes `aws:autoscaling/group:Group`, whose `Tags` field is a
  **list** of `{key,value,propagateAtLaunch}`, not a `StringMap` — the
  `StringMapInput` type check rejects it automatically. (The TS allowlist had
  to special-case this by hand.)
- Resources whose `Args` struct simply has no `Tags`/`Labels` field
  (`certificateValidation`, `rolePolicyAttachment`, `route53/record`,
  `lb/targetGroupAttachment`, `s3` sub-resources, etc.) are skipped because
  `FieldByName` returns an invalid value.

We additionally gate on the **type-token prefix** (`aws:`, `gcp:`,
`azure-native:`) so a provider's transformation never touches another
provider's resources.

### Robust tag/label merge

`props.Tags` may be a plain `pulumi.StringMap` or a `pulumi.StringMapOutput`.
Do not range over it directly. Convert both the incoming labels and the
existing value to `StringMapOutput` and merge inside `ApplyT`:

```go
merged := pulumi.All(labelsOut, existingOut).ApplyT(func(parts []any) map[string]string {
    out := map[string]string{}
    for k, v := range parts[0].(map[string]string) { out[k] = v } // labels first
    for k, v := range parts[1].(map[string]string) { out[k] = v } // existing wins
    return out
}).(pulumi.StringMapOutput)
```

### Precedence

**Functional/explicit tags win over user labels.** Spread labels *first*, then
the existing tags on top. This ensures `defang:service`, `defang:scope`, and
the project-wide `defang-*` tags can never be clobbered by a user label, which
would break tag-based selectors. Provider/project default tags merge in for
keys the user didn't set.

## No silent failures (AWS) vs. sanitization (GCP)

- **AWS / Azure** tags accept a wide character set (dots, mixed case, etc.).
  Labels are passed through **verbatim**. If a user supplies a reserved key
  (e.g. the `aws:` prefix), let the deploy **fail loudly** rather than silently
  dropping it. The label→tags normalizer only collapses empty → `nil`.

- **GCP** labels are validated strictly by the API: keys and values must match
  `[a-z]([a-z0-9_-]){0,62}` (lowercase, start with a letter, ≤63 chars), and
  dots are not allowed. Compose labels routinely use reverse-DNS keys with dots
  and mixed-case values (`com.acme.team=Core`). Passing those through verbatim
  would fail every GCP deploy that uses conventional labels. We therefore
  **sanitize** GCP label keys/values: lowercase, replace invalid characters
  (including `.`) with `_`, ensure a leading lowercase letter, and truncate to
  63 chars. (This is a deliberate divergence from the AWS verbatim rule,
  forced by GCP's validation.)

## Wiring per provider

The transformation is built from the relevant labels and merged into `opts`
just before the component is registered, using
`pulumi.Composite(opts, pulumi.Transformations([]pulumi.ResourceTransformation{t}))`.
A `nil` transformation (no labels) means we leave `opts` untouched.

In this provider both transformations are attached at the **component** level:
the per-service transform on each service component (`newService` /
`buildService` / `createServiceResources`), and the per-network (default-only)
transform on the **Project component** in `Construct`. Attaching the network
transform at the Project component is the simplest mechanism that reaches the
whole shared-infra subtree — including a multi-language component like the awsx
VPC, whose `Tags` it sets and which awsx then propagates to the subnets/NAT it
creates internally (an in-process transform can't reach those directly). The
trade-off: it also cascades onto service resources. That is acceptable — the
default network spans every service — and per-service labels and functional
tags win on key collision, so nothing is clobbered.

### AWS (`provider/defangaws`, prefix `"aws"` — matches `aws:` and `awsx:`)

- **Per-service.** From `svc.Labels` on the service component in `newService`
  (covers all branches: ECS service, RDS Postgres, ElastiCache Redis).
- **Per-network (default).** From `inputs.Networks[compose.DefaultNetwork].Labels`
  on the Project component in `Construct`.
- No existing transformations in the AWS provider — this is the first.

### GCP (`provider/defanggcp`, SDK: `pulumi-gcp`, prefix `"gcp:"`, sanitized)

- **Per-service.** Sanitized from `svc.Labels` on the service component in
  `buildService`. Resources that carry GCP `Labels`:
  `gcp:cloudrunv2/service:Service` and
  `gcp:compute/instanceTemplate:InstanceTemplate`. The instance group manager
  (`regionInstanceGroupManager`) has **no** `Labels` field — reflection skips it.
- **Per-network (default).** Sanitized, on the Project component in `Construct`.
  GCP `Network`/`Subnetwork`/`Firewall` have **no `Labels` field**, so the core
  network resources are skipped; the transform still lands on shared resources
  that do have `Labels` (e.g. the reserved public IP, DNS zone, Artifact
  Registry).

### Azure (`provider/defangazure`, SDK: `azure-native`, prefix `"azure-native:"`)

- A project-wide `DefaultTagsTransformation(BaseTags)` already cascades
  `defang-org/project/stack/etag` to every azure-native resource. The
  per-network transform is composed alongside it on the Project component.
- **Per-service.** From `svc.Labels` on the service component in
  `createServiceResources`. Existing `ServiceTags(serviceName)` and base tags
  win on collision.
- **Per-network (default).** On the Project component in `Construct`, composed
  with the base-tags transform. Lands on VNet, subnets, DNS zones, managed
  environment.

### Standalone Service path (all providers)

Per-service labels are wired only on the **project** dispatch path, where
`compose.ServiceConfig.Labels` is populated from the parsed compose file. The
standalone `ServiceInputs` structs do not carry a `Labels` field, so a directly
instantiated `Service` is not label-tagged. Adding standalone support would mean
adding `Labels` to each provider's `ServiceInputs` (a schema change) — left as a
follow-up.

## Scope boundaries

- **"Network label" = the default network's labels, applied project-wide.**
  Because the transform is attached at the Project component, default-network
  labels reach all shared infrastructure (VPC/subnets/NAT, ALB, listeners, ECS
  cluster, log groups, security groups, DNS zones, etc.) **and** service
  resources. Per-service labels and functional/`defang:*` tags win on collision.
- Resources created inside a multi-language component (e.g. awsx VPC subnets)
  are tagged via that component's own tag propagation, not the in-process
  transform, since transforms don't cross the MLC boundary.

## Known pre-existing gap (do not rely on it)

`defang:service` is **not** applied uniformly to every service-owned AWS
resource today (missing on the app-image CodeBuild project, ACME cert IAM
policies, ACME/app listener rules, and ACME target groups). So `defang:service`
is **not** a reliable "which service owns this" oracle for verification. The
label transformation does **not** depend on it — it works purely via `opts`
propagation. (Opportunity: the same per-service transformation could inject
`defang:service` uniformly to close the gap, at the cost of golden-file churn.)

## Verification

- Assert the invariant **per resource, scoped to the provider's resource tokens
  only**: a resource carries a label tag/label **iff** its owning
  service/network is labeled in the compose file.
- When matching ownership by resource name, account for **name sanitization**
  (e.g. `example.com` → `examplecom`, dots stripped) or you'll get
  false-positive "leaks."
- Unit-test the transformation directly (verbatim AWS/Azure merge, GCP
  sanitization, existing-tag precedence, empty → no-op) and end-to-end by
  registering a component with the transformation in `opts` and inspecting the
  resulting resource inputs via the mock Pulumi server in `tests/testutil`.

## Open TODOs

- `DeployConfig.Labels` and `BuildConfig.Labels` are intentionally left out for
  now (commented in `types.go`). If added, decide how they merge with the
  service-level labels (build labels → build resources only; deploy labels →
  runtime resources).
- GCP per-network labels remain unsupported by the SDK; revisit if pulumi-gcp
  adds `Labels` to network types.
</content>
</invoke>
