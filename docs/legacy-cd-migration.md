# Moving a project to this CD

Defang has shipped more than one deploy driver ("CD"). This page is for a
project that an older driver deployed and that now needs to run on this one.

## Why migration needs a preflight

The old and current drivers open the same Pulumi project and stack. Their
resource names and parent trees differ, so an ordinary deploy can look like a
request to create new infrastructure and delete the old infrastructure. If a
managed database is replaced, its data can be lost.

The current CD performs a state preflight before `preview` and `up`. It reads
resource identity (URN, type, and parent) from the snapshot and discards all
exported inputs and outputs, which can contain secrets. For supported AWS and
GCP databases it gives Pulumi the old resource's exact URN as an alias. The
alias preserves identity, but it does not control the cloud provider's diff:
an immutable input can still make the provider request replacement.

The guard therefore has two stages. First, `up` stops unless every persistent
legacy service can be mapped to exactly one managed service and every other
resource has an explicitly reviewed replacement path that the current CD image
can operate. Second, after final stack configuration is set, the CD
automatically runs a real Pulumi preview with the same targets and target
dependents as `up`. Only `same` and in-place `update` operations are accepted
for adopted data-bearing resources. A replacement, deletion, creation, unknown
operation, preview error, or incomplete event stream aborts before `up` can run.

## Start with preview

Run a preview with the same project, stack, compose file, recipe, and cloud that
the real deployment will use:

```console
defang cd preview --stack <old-stack>
```

The explicit `preview` command is deliberately non-blocking. It applies every
alias the state preflight can prove and prints both the prepared alias mappings
and any blockers that would stop a real `up`. Inspect the Pulumi plan. Each
existing managed database must be adopted in place: the plan must not create,
replace, or delete its RDS, ElastiCache, MemoryDB, Cloud SQL, or Memorystore
data resource. This operator-facing preview does not replace the automatic
provider-backed safety preview that every eligible migration `up` performs.

Do not run the real deployment if preview reports a blocker or plans a database
replacement. Contact Defang support or use the blue/green procedure below.

## In-place migrations that can proceed

The preflight can prepare exact-URN aliases for these resources:

- AWS RDS instances and their subnet/security groups.
- AWS ElastiCache replication groups and their subnet/security groups.
- AWS MemoryDB clusters and their subnet, parameter, and security groups.
- GCP Cloud SQL instances and their optional user/database children.
- GCP Memorystore instances.

The requested compose project must still contain a uniquely matching managed
Postgres or Redis service. Existing `x-defang-aliases` values must agree with
the state, and an AWS Redis deployment must keep the same ElastiCache or
MemoryDB engine. Mixed states are accepted only when each surviving legacy
database has one target and that target does not already exist in the current
resource tree.

When the state checks pass, `up` logs the exact aliases it prepared and runs
the automatic provider-backed safety preview. It proceeds only if every
adopted data-bearing resource is unchanged or updated in place. Other
explicitly reviewed stateless or reconstructible legacy infrastructure may be
recreated as part of the move, so an operator preview is still recommended. An
unknown resource type fails closed even when it belongs to the selected cloud;
a cloud package match by itself is not evidence that replacement is safe.

## Cases that require blue/green migration

Use a new stack when the preflight cannot prove an in-place migration safe. In
particular:

- A legacy GCP state containing `cloudbuild:index:CloudBuild` is blocked because
  the current minimal GCP CD image does not contain that private provider
  plugin.
- A legacy AWS state containing `pulumi-nodejs:dynamic:Resource` is blocked
  because the current minimal AWS CD image does not contain the legacy Node
  dynamic provider/runtime.
- Legacy Azure state has no defined in-place adoption path.
- Switching an existing stack between AWS, GCP, and Azure is blocked.
- A renamed, removed, ambiguous, duplicated, or otherwise unmapped managed
  database is blocked.
- An unclassified AWS, GCP, or third-party resource type is blocked. This
  includes storage or secret resources that have no explicit adoption or
  replacement rule.
- An unreadable or malformed state is blocked for `up`; the CD will not guess
  that it is safe.

These runtime limitations need separate compatibility work. An exact-stack
override can bypass the guard, but it does not install a missing provider or
make an unaliased database safe.

### Blue/green procedure

1. Deploy to a stack name the project has never used:

   ```console
   defang compose up --stack <new-stack>
   ```

2. Back up the old database and restore it into the new stack using the normal
   database tools. Defang does not copy data between stacks.

3. Exercise the application against the new stack's URLs.

4. Move the project's traffic or domain to the new stack.

5. Only after verification and a recoverable backup, remove the old stack:

   ```console
   defang compose down --stack <old-stack>
   ```

`down` and `destroy` are intentionally not guarded. They delete the selected
stack, including its databases, because deletion is exactly what those commands
request. Never run either command merely to clear a migration error.

## If the result looks wrong

A blocked `up` stops before running the Pulumi update or changing cloud
infrastructure. The CD may already have written the requested stack config,
because the provider-backed preview must evaluate that final configuration.
Retry a transient state-backend or preview failure. If the same error remains,
keep the old stack in place and contact Defang support with the preflight
message and operator preview. State inputs, outputs, and raw preview errors are
not included in migration diagnostics.

## For Defang operators

The break-glass override is `defang:allowLegacyStateTakeover` in the recipe's
`pulumi_config`. Its value must be the exact `<project>/<stack>` target:

```yaml
config:
  defang:allowLegacyStateTakeover: acme-shop/prod
```

The value is not a boolean. A recipe is tenant/mode scoped and can be shared by
multiple projects, so an exact target prevents an authorization for one stack
from disabling the guard for another. The preflight still applies every alias
it can prove before the override bypasses remaining blockers. It also runs the
automatic provider-backed preview when adopted data-bearing resources were
found, but the override may continue after a destructive result or preview
failure. Both bypasses write prominent, bounded, secret-free warnings to the
deployment log. Remove the entry immediately after the migration; it has no
expiry.

`DEFANG_ALLOW_LEGACY_STATE_TAKEOVER` provides the same exact-stack override for
a CD run started by hand. The CLI does not forward it during ordinary deploys,
and self-destruct resources explicitly exclude it from their environment.

An override is not a repair. Before using one, verify from an explicit preview
and the alias audit log that every database is adopted in place, resolve
immutable-input and legacy provider/runtime differences, take a tested backup,
and record who approved the exception.
