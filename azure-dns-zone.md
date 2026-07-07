# Azure BYOD DNS-zone support

## Goal

Mirror the AWS BYOD ("bring your own domain") DNS behaviour for Azure: when a
service sets a custom `domainname` and a **public Azure DNS zone for that domain
already exists in the current subscription**, Defang should manage the DNS
records directly in that zone and issue an Azure-managed TLS certificate for the
hostname — instead of falling back to the ACME / `defang cert generate` flow.

Decisions (from product owner):

1. **Lookup runs CLI-side.** The CLI finds the zone and sets `ServiceInfo.ZoneId`
   (an ARM resource ID), exactly like AWS sets the Route53 zone id.
2. **A managed cert is issued whenever the zone is used.**
3. **No restriction on which zone matches** — longest DNS-suffix match across the
   whole subscription wins (no tag/ownership filter). No cross-subscription /
   assume-role support (Azure has no AssumeRole; out of scope).

## How AWS does it today (reference)

- CLI `byoc/aws/byoc.go` `UpdateServiceInfo` → `findZone(domain, dnsRole)` does a
  Route53 `ListHostedZonesByName` (optionally after `AssumeRole`). Found → set
  `si.ZoneId`; not found → `si.UseAcmeCert = true`.
- The provider/CD then manages records + ACM cert in that zone; otherwise ACME.

The `ServiceInfo.zone_id` proto field (field 12, "zone ID for byod domain")
already exists and is shared by all providers — no proto change needed.

## Current Azure state

- CLI `byoc/azure/byoc.go` `UpdateServiceInfo` **unconditionally** sets
  `si.UseAcmeCert = true` for any service with a `DomainName`. No zone lookup.
- Provider `defangazure/azure/customdomain.go` `CreateCustomDomain` only writes
  CNAME + `asuid` TXT records into the **delegate-domain** zone
  (`infra.Domain`, pre-created by the CLI's `PrepareDomainDelegation`). It
  ignores the service's real `DomainName` for record creation.
- Delegate-domain managed certs are issued by the CD program
  (`cd/program/azure.go` `provisionDelegateDomainCerts`) for `<svc>.<domain>`.
- The azure-native Pulumi provider is a single instance on the deployment's
  subscription (`cd/program/azure.go`); records are created with the ambient
  provider. Since the matched zone is in the same subscription, no second
  provider is needed.

## Design

### 1. CLI: `defang` repo

**`src/pkg/clouds/azure/dns/dns.go`** — add:

```go
// FindZone returns the ARM resource ID of the public DNS zone in the current
// subscription whose name is the longest DNS suffix of domain, or "" if none
// matches. No ownership/tag filtering (per design). Subscription-wide list.
func (d *DNS) FindZone(ctx context.Context, domain string) (string, error)
```

Implementation: `armdns.ZonesClient.NewListPager(nil)` (lists every zone in the
subscription), normalise to lower-case, keep the match (`domain == name` or
`strings.HasSuffix(domain, "."+name)`) with the longest `name`, return its `*ID`.

**`src/pkg/cli/client/byoc/azure/byoc.go`** `UpdateServiceInfo`:

```go
if service.DomainName == "" { return nil }
if err := b.setUpLocation(); err != nil || b.driver.SubscriptionID == "" {
    si.UseAcmeCert = true   // can't look up → ACME fallback
    return nil
}
zoneID, err := azuredns.New("", b.driver.Azure).FindZone(ctx, service.DomainName)
if err != nil {
    term.Debugf("FindZone(%q): %v; using ACME", service.DomainName, err)
    si.UseAcmeCert = true   // lookup failed (e.g. missing DNS read perms) → ACME
    return nil
}
if zoneID == "" {
    si.UseAcmeCert = true
} else {
    si.ZoneId = zoneID      // CD will manage records + issue a managed cert
}
```

Lookup failure falls back to ACME rather than hard-failing the deploy (a missing
DNS read permission should not block the whole `compose up`).

### 2. Provider: `pulumi-defang` repo

- **`ProjectInputs`** (`provider/defangazure/project.go`): add
  `DnsZones map[string]string` (`pulumi:"dnsZones,optional"`), service name →
  zone ARM resource ID.
- **`ServiceInputs`** (`provider/defangazure/service.go`): add
  `DnsZoneId string` (`pulumi:"dnsZoneId,optional"`) for the standalone path.
- Thread the per-service zone id through `createServiceResources` →
  `createContainerApp` → `azure.CreateContainerApp(... dnsZoneId string ...)`.
- **`provider/defangazure/azure/customdomain.go`**: add `CreateByodDomain`,
  called right after `CreateCustomDomain` in `CreateContainerApp`. When the
  service has a `DomainName`, a non-empty `dnsZoneId`, and public ingress:
  - parse `resourceGroup` + `zoneName` from the ARM id,
  - compute the relative record name (`domain` minus `.zoneName`; `@` for apex),
  - create a CNAME `<relative>` → `<app>.<env.DefaultDomain>` and a TXT
    `asuid[.<relative>]` carrying `containerApp.CustomDomainVerificationId`,
    in that customer zone/RG.
  - **Apex limitation:** Azure rejects a CNAME at the zone apex. Subdomains are
    supported; apex custom domains would need an A record to the env inbound IP
    and are out of scope for this change (documented, not implemented).

The delegate-domain records continue to be created as before, so a BYOD service
remains reachable on both its custom domain and `<svc>.<delegate-domain>`.

### 3. CD program: `pulumi-defang/cd/program/azure.go`

- In `deployAzure`, after `toAzureArgs`, populate `args.DnsZones` from
  `projectUpdate.Services` (`si.GetService().GetName()` → `si.GetZoneId()`,
  skipping empties).
- Refactor `provisionDelegateDomainCerts` into a generic `provisionCerts(jobs)`
  that issues + binds an ACA managed cert per `(serviceName, hostname)`:
  - delegate-domain jobs: `<svc>.<domain>` for each ingress service (when
    `domain != ""`) — unchanged behaviour.
  - BYOD jobs: `si.Domainname` for each service with `si.ZoneId != ""`.
  - Both bind on the container app in the project resource group; records already
    exist (delegate zone or customer zone) so `aca.IssueCert` DNS-wait passes.
  - Chained off `project.Endpoints` and exported, same as today, so Pulumi
    sequences cert issuance after all records/apps exist.

### 4. SDK regeneration

`ProjectInputs`/`ServiceInputs` changes require regenerating the Azure SDK:
`make schema` + `make sdks` (azure), commit the `sdk/v2/**` diff alongside the
source change (pre-push hook enforces a clean `sdk/v2/`).

## Files touched

defang (CLI):
- `src/pkg/clouds/azure/dns/dns.go` — `FindZone`
- `src/pkg/cli/client/byoc/azure/byoc.go` — `UpdateServiceInfo`

pulumi-defang (provider + CD):
- `provider/defangazure/project.go` — `DnsZones` input + threading
- `provider/defangazure/service.go` — `DnsZoneId` input + threading
- `provider/defangazure/azure/containerapp.go` — `CreateContainerApp` param + call
- `provider/defangazure/azure/customdomain.go` — `CreateByodDomain`
- `cd/program/azure.go` — `DnsZones` wiring + BYOD cert jobs
- `sdk/v2/**` — regenerated

## Edge cases / notes

- **Not delegated at registrar:** the zone may exist but its NS not be delegated
  from the parent at the registrar; cert validation then fails. That's the user's
  responsibility (same as AWS). Records are still created.
- **Multiple matching zones:** longest-suffix wins; ties are not expected (zone
  names are unique). Subscription-wide, no ownership filter (per design).
- **Apex domains:** unsupported (CNAME-at-apex limitation); subdomains only.
- **Lookup permission:** if the deploy identity can't list DNS zones, the CLI
  falls back to ACME rather than failing.
</content>
</invoke>
