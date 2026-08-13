# Azure BYOD DNS-zone support

## Goal

Mirror the AWS BYOD ("bring your own domain") DNS behaviour for Azure: when a
service sets a custom `domainname` and a **public Azure DNS zone for that domain
already exists in the deploy subscription**, Defang manages the DNS records
directly in that zone and issues an Azure-managed TLS certificate for the
hostname — instead of falling back to the ACME / `defang cert generate` flow.

Decisions (from product owner):

1. **The lookup runs in the CD task**, not the CLI. The CD program resolves each
   BYOD domain to a zone ARM resource ID and passes it to the provider. AWS does
   the equivalent lookup CLI-side (`ServiceInfo.ZoneId`); Azure does not, because
   the CD task runs in the customer's cloud with the deploy identity, which
   already has DNS read permission and knows its subscription. A user running
   `defang compose up` therefore needs no extra client-side role.
2. **A managed cert is issued whenever the zone is used.**
3. **No restriction on which zone matches** — longest DNS-suffix match across the
   whole subscription wins (no tag/ownership filter). No cross-subscription /
   assume-role support (Azure has no AssumeRole; out of scope).

## How AWS does it today (reference)

- CLI `byoc/aws/byoc.go` `UpdateServiceInfo` → `findZone(domain, dnsRole)` does a
  Route53 `ListHostedZonesByName` (optionally after `AssumeRole`). Found → set
  `si.ZoneId`; not found → `si.UseAcmeCert = true`.
- The provider/CD then manages records + ACM cert in that zone; otherwise ACME.

Azure deliberately diverges on *where* the lookup happens (decision 1 above), so
the `ServiceInfo.zone_id` proto field is not used on the Azure path.

## Starting point

- CLI `byoc/azure/byoc.go` `UpdateServiceInfo` unconditionally sets
  `si.UseAcmeCert = true` for any service with a `DomainName`, keeping
  `defang cert generate` available. This is unchanged by this design.
- Provider `defangazure/azure/customdomain.go` `CreateCustomDomain` only writes
  CNAME + `asuid` TXT records into the **delegate-domain** zone
  (`infra.Domain`, pre-created by the CLI's `PrepareDomainDelegation`). It
  ignores the service's real `DomainName` for record creation.
- Delegate-domain managed certs are issued by the CD program
  (`cd/program/azure.go`) for `<svc>.<domain>`.
- The azure-native Pulumi provider is a single instance on the deployment's
  subscription (`cd/program/azure.go`); records are created with the ambient
  provider. Since the matched zone is in the same subscription, no second
  provider is needed.

## Design

### 1. Zone lookup helper: `defang` repo

**`src/pkg/clouds/azure/dns/dns.go`** — `FindZone`:

```go
// FindZone returns the ARM resource ID of the public DNS zone in the current
// subscription whose name is the longest DNS suffix of domain, or "" if none
// matches. No ownership/tag filtering (per design). Subscription-wide list.
func (d *DNS) FindZone(ctx context.Context, domain string) (string, error)
```

Implementation: `armdns.ZonesClient.NewListPager(nil)` (lists every zone in the
subscription), then `bestZoneMatch` normalises to lower-case and keeps the match
(`domain == name` or `strings.HasSuffix(domain, "."+name)`) with the longest
`name`, returning its `*ID`.

It lives in the `defang` repo because `aca.IssueCert` already does, and the CD
module imports that package — one cloud-SDK layer, two callers.

### 2. CD program: `pulumi-defang/cd/program/azure.go`

- `findByodZones(ctx, cf)` collects every service with a `domainname` and public
  ingress, calls `FindZone` per domain, and returns `service name → zone ARM id`
  for the ones that matched.
- A failed lookup (e.g. the deploy identity lacks DNS read permission) or a
  missing `AZURE_SUBSCRIPTION_ID` is a Pulumi **warning**, not an error: the
  Container App still deploys and serves on its `azurecontainerapps.io` URL, and
  the delegate domain is unaffected. A domain with no matching zone is logged at
  info level and keeps the delegate-domain / ACME behaviour.
- `deployAzure` threads the result into `args.DnsZones` before
  `defangazure.NewProject`.
- `provisionCerts(jobs)` issues + binds an ACA managed cert per
  `(serviceName, hostname)`:
  - delegate-domain jobs: `<svc>.<domain>` for each ingress service (when
    `domain != ""`) — unchanged behaviour.
  - BYOD jobs: `svc.DomainName` for each service in the `findByodZones` result
    that passes `ByodRecordEligible` (i.e. the provider will actually create the
    records, so `aca.IssueCert` won't wait out its DNS timeout for records that
    never appear).
  - Both bind on the container app in the project resource group; records already
    exist (delegate zone or customer zone) so the `aca.IssueCert` DNS-wait passes.
  - Chained off `project.Endpoints` and exported, same as today, so Pulumi
    sequences cert issuance after all records/apps exist.

### 3. Provider: `pulumi-defang` repo

- **`ProjectInputs`** (`provider/defangazure/project.go`):
  `DnsZones map[string]string` (`pulumi:"dnsZones,optional"`), service name →
  zone ARM resource ID.
- **`ServiceInputs`** (`provider/defangazure/service.go`):
  `DnsZoneId string` (`pulumi:"dnsZoneId,optional"`) for the standalone path.
- The per-service zone id is threaded through `createServiceResources` →
  `createContainerApp` → `azure.CreateContainerApp(... dnsZoneID string ...)`.
- **`provider/defangazure/azure/customdomain.go`**: `CreateByodDomain`, called
  right after `CreateCustomDomain` in `CreateContainerApp`. When the service has a
  `DomainName`, a non-empty `dnsZoneID`, and public ingress:
  - parse `resourceGroup` + `zoneName` from the ARM id,
  - compute the relative record name (`domain` minus `.zoneName`; `@` for apex),
  - create the routing record and a TXT `asuid[.<relative>]` carrying
    `containerApp.CustomDomainVerificationId`, in that customer zone/RG:
    - **subdomain** → CNAME `<relative>` → `<app>.<env.DefaultDomain>`,
    - **apex** → A record `@` → the managed environment's static inbound IP
      (Azure rejects a CNAME at the zone apex, RFC 1034).
  - A domain the resolved zone cannot host (neither apex nor a subdomain of it)
    logs a warning rather than being silently skipped.

The delegate-domain records continue to be created as before, so a BYOD service
remains reachable on both its custom domain and `<svc>.<delegate-domain>`.

### 4. Apex cert validation (`defang` repo)

Apex domains route via an A record, so Azure rejects CNAME cert validation with
`InvalidValidationMethod`. `aca.issueManagedCertificate` retries with **HTTP**
validation, which Container Apps completes on its own once the A record is in
place — no extra DNS record, so it works unattended in the CD task. TXT
validation (the interactive `_dnsauth` dance) stays as the last resort for the
CLI path.

The DNS wait needs no apex special-case: `dns.CheckDomainDNSReady` already
accepts an A record whose addresses match those of the app FQDN, which is the
environment's static IP — exactly what an apex A record must point at.

### 5. AWS: don't fail the deploy when there is no hosted zone

Not Azure, but the same "zone may not exist" question, and worth fixing before the
AWS cutover. On AWS the zone lookup **already** runs CD-side —
`createCertsAndRoute53Dns` calls `GetHostedZoneForHost` and never reads
`ServiceInfo.ZoneId` — but it treats a missing zone as fatal, so the whole deploy
fails. The legacy CD gated the block on `serviceInfo.zoneId`
(`defang-mvp/pulumi/cd/aws/defang_service.ts:414`), so "no zone" simply meant "no
BYOD records" and the ACME path still worked. That difference would land as a
cutover regression for exactly the services the CLI marks `UseAcmeCert`.

`IsZoneNotFound` distinguishes "no zone matched" from a real failure (bad
credentials, an un-assumable `x-defang-dns-role`, throttling) and the caller skips
that hostname with a warning; anything else still fails loudly. The match is on
text because the invoke crosses the Pulumi engine over gRPC, which flattens
upstream's typed `*retry.NotFoundError` — see the helper's comment. If upstream
rewords the message the match fails closed, i.e. back to today's hard failure,
never to a silent skip.

### 6. SDK regeneration

`ProjectInputs`/`ServiceInputs` changes require regenerating the Azure SDK:
`make schema` + `make sdks` (azure), commit the `sdk/v2/**` diff alongside the
source change (pre-push hook enforces a clean `sdk/v2/`).

## Files touched

defang (cloud-SDK layer):
- `src/pkg/clouds/azure/dns/dns.go` — `FindZone` + `bestZoneMatch`
- `src/pkg/clouds/azure/aca/cert.go` — HTTP validation for apex domains

pulumi-defang (provider + CD):
- `cd/program/azure.go` — `findByodZones` + `DnsZones` wiring + BYOD cert jobs
- `provider/defangaws/aws/route53.go` — `IsZoneNotFound`
- `provider/defangaws/aws/cert.go` — skip BYOD records when no hosted zone exists
- `provider/defangazure/project.go` — `DnsZones` input + threading
- `provider/defangazure/service.go` — `DnsZoneId` input + threading
- `provider/defangazure/azure/containerapp.go` — `CreateContainerApp` param + call
- `provider/defangazure/azure/customdomain.go` — `CreateByodDomain`
- `sdk/v2/**` — regenerated

## Edge cases / notes

- **Not delegated at registrar:** the zone may exist but its NS not be delegated
  from the parent at the registrar; cert validation then fails. That's the user's
  responsibility (same as AWS). Records are still created.
- **Multiple matching zones:** longest-suffix wins; ties are not expected (zone
  names are unique). Subscription-wide, no ownership filter (per design).
- **Lookup permission:** if the deploy identity can't list DNS zones, the CD
  warns and the service keeps the delegate domain only.
- **Record ownership:** Pulumi owns the records it creates, so `compose down`
  deletes them (the zone itself is untouched). A pre-existing record at the same
  name conflicts on create — same ownership model as AWS.
