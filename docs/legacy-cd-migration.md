# Moving a project to this CD

Defang has shipped more than one deploy driver ("CD"). This page is for a
project that an older one deployed, and that you now want on this one.

## Why you cannot deploy in place

Every CD is given the same Pulumi state: same bucket, same project, same stack.
So this CD opens the older CD's state and finds resources it does not recognise.

The two CDs share no resource identifiers, and this CD has no code to adopt the
older one's resources. Pulumi therefore reads every existing resource as gone.
One deploy would:

1. create a complete new set of infrastructure, including a new, empty database,
   then
2. delete everything the older CD made, including the database that holds your
   data.

Outside production mode, managed databases are created with deletion protection
and backups off, so that data is not recoverable.

This is why `defang compose up` stops with an error on such a stack, instead of
deploying. The guard that stops it is in `cd/legacy_state.go`.

## What to do instead

Deploy to a new stack, move traffic to it, then shut the old one down. The two
stacks run side by side while you check the new one, and the old one keeps
serving until you are ready.

1. **Deploy to a new stack.** Pick a name the project has never used.

   ```
   defang compose up --stack <new-name>
   ```

   This creates its own infrastructure and its own database. Nothing in the old
   stack is touched.

2. **Move your data.** Back up the old database and restore into the new one,
   with your normal database tooling. Defang does not copy data between stacks.

3. **Check the new stack.** Its URLs are separate from the old ones. Exercise
   the app against them before you send anyone real.

4. **Move traffic.** Point your domain at the new stack.

5. **Shut the old stack down**, once you are confident and your backups are
   safe.

   ```
   defang compose down --stack <old-name>
   ```

   `down` and `destroy` are not blocked on an old stack. They delete what the
   old CD made, which is what you are asking for at this point.

## If the error is wrong

The check refuses any stack holding resources this CD did not create. If you
believe your stack does belong to this CD, contact Defang support rather than
working around it — a wrong deploy here is not reversible.

Also do not run `down` or `destroy` to clear the error. Neither is blocked, and
both delete the resources the error just listed. The stack is not stuck — the
deploy stopped before it changed anything.

## For Defang operators

A deliberate takeover is authorised with `defang:allowLegacyStateTakeover` in
the recipe's `pulumi_config`. The value is the `<project>/<stack>` it applies
to, for example:

```yaml
config:
  defang:allowLegacyStateTakeover: acme-shop/prod
```

The value is a stack name and not `true` on purpose. A recipe is keyed by
tenant and mode, so one recipe is shared by every project that tenant deploys
in that mode. `true` would disarm the guard for all of them, including projects
nobody is migrating. Naming the target keeps a tenant-wide setting to a
single-stack effect.

Remove the entry once the migration is done. There is no expiry.

`DEFANG_ALLOW_LEGACY_STATE_TAKEOVER` does the same thing, with the same
`<project>/<stack>` value, for a CD run started by hand — which has no recipe.
The CLI does not pass it through, so it has no effect on a normal deploy.
