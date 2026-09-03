# BYOD: finding an existing DNS zone

## Goal

When a service sets a custom `domainname` ("bring your own domain"), Defang
should check whether a DNS zone that can host that domain **already exists** in
the account being deployed to and is authorized for this use. If one does,
manage the records there and issue a cloud-managed TLS certificate. If none
does, that is a normal outcome, not an error: keep the service reachable on its
delegate-domain hostname and use the cloud's safe fallback.

Every cloud answers the same two questions, so they should behave the same way:

1. **Which authorized existing zone can host this domain?** — longest DNS-suffix match.
2. **What if none can?** — degrade, don't fail.

This document covers where that lookup lives per cloud and what it does. The
per-cloud record and certificate mechanics differ and are described where they
diverge.

## Shared decisions

- **The lookup runs in the CD task**, inside the customer's cloud. The deploy
  identity there already has DNS read permission and knows which account it is
  in, so a user running `defang compose up` needs no extra client-side role.
- **A managed cert is issued whenever a zone is used.**
- **Zone trust is cloud-specific.** GCP requires an explicit zone-owner opt-in
  before Defang writes records; see the GCP section. Existing AWS and Azure
  behavior is described in their sections below.
- **"No zone" is not an error.** AWS and Azure retain the delegate-domain +
  ACME path. GCP exact hostnames use load-balancer authorization; GCP wildcards
  retain the delegate-domain + ACME path because they require DNS authorization.

## AWS

The lookup already runs CD-side and needs no move: `createCertsAndRoute53Dns`
(`provider/defangaws/aws/cert.go`) calls `GetHostedZoneForHost`, which walks up
from the hostname's parent zone. `ServiceInfo.ZoneId` is **not** read — the CLI
still computes it, but only the `UseAcmeCert` boolean it implies is used, to
decide whether to print the `defang cert gen` hint.

Cross-account zones are handled by a role-assuming provider built from
`awsConfig.DnsRoleArn` (`x-defang-dns-role`), see `aws/infra.go`.

**What this change fixes:** a missing hosted zone was fatal. The lookup error
propagated and the whole deploy failed, and the BYOD block fired on
`svc.DomainName` alone with no zone-existence gate. The legacy CD gated it on
`serviceInfo.zoneId` (`defang-mvp/pulumi/cd/aws/defang_service.ts:414`), so "no
zone" simply meant "no BYOD records" and the ACME path still worked. That
difference would have landed as a cutover regression for exactly the services
the CLI marks `UseAcmeCert`.

`IsZoneNotFound` (`aws/route53.go`) separates "no zone matched" from a real
failure — bad credentials, an un-assumable DNS role, throttling — and only the
former skips the hostname, with a warning pointing at `defang cert gen`.
Everything else still fails loudly.

The match is on message text. The invoke crosses the Pulumi engine over gRPC,
which flattens upstream's typed `*retry.NotFoundError` into an opaque string, so
`errors.Is`/`As` can't see it. Both the current wording
(`no matching Route 53 Hosted Zone found`) and the pre-AWS-SDK-v2 one
(`no matching Route53Zone found`, pulumi-aws <= v6.37) are accepted. If upstream
rewords it again the match **fails closed** — back to the hard failure that
predates this helper, never to a silent skip.

## Azure

`FindZone` (`provider/defangazure/azure/dnszone.go`) lists the subscription's
public zones and picks the longest DNS suffix of the domain via `bestZoneMatch`,
returning the zone's ARM resource ID or `""`.

Unlike AWS's, this can't be a Pulumi invoke: azure-native exposes `getZone`
(resource group + zone name required) but no subscription-wide listing, and the
zone's resource group is exactly what isn't known yet. So it is an imperative ARM
call, like the other out-of-band lookups in that package (`readLiveCustomDomains`,
`ModelSelector`).

Only the deploy subscription is searched. Azure has no cross-subscription
equivalent of Route53's `AssumeRole`, so there is no analogue of
`x-defang-dns-role`.

### CD program: `cd/program/azure.go`

- `findByodZones(ctx, cf)` collects every service with a `domainname` and public
  ingress, calls `FindZone` per domain, and returns `service name → zone ARM id`
  for the ones that matched.
- A failed lookup (e.g. the deploy identity lacks DNS read permission) or a
  missing `AZURE_SUBSCRIPTION_ID` is a Pulumi **warning**, not an error. A domain
  with no matching zone is logged at info level. Both keep the delegate-domain /
  ACME behaviour — the AWS rule, applied here.
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
  - Chained off `project.Endpoints` and exported, so Pulumi sequences cert
    issuance after all records and apps exist.

### Provider

- **`ProjectInputs.DnsZones`** (`provider/defangazure/project.go`):
  `map[string]string` (`pulumi:"dnsZones,optional"`), service name → zone ARM id.
- **`ServiceInputs.DnsZoneId`** (`provider/defangazure/service.go`) for the
  standalone path.
- Threaded through `createServiceResources` → `createContainerApp` →
  `azure.CreateContainerApp(... dnsZoneID string ...)`.
- **`azure/customdomain.go`** `CreateByodDomain`, called right after
  `CreateCustomDomain`. When the service has a `DomainName`, a non-empty
  `dnsZoneID`, and public ingress:
  - parse `resourceGroup` + `zoneName` from the ARM id,
  - compute the relative record name (`domain` minus `.zoneName`; `@` for apex),
  - create the routing record and a TXT `asuid[.<relative>]` carrying
    `containerApp.CustomDomainVerificationId`, in that customer zone/RG:
    - **subdomain** → CNAME `<relative>` → `<app>.<env.DefaultDomain>`,
    - **apex** → A record `@` → the managed environment's static inbound IP
      (Azure rejects a CNAME at the zone apex, RFC 1034).
  - A domain the resolved zone cannot host (neither apex nor a subdomain of it)
    logs a warning rather than being silently skipped.

The delegate-domain records are still created, so a BYOD service remains
reachable on both its custom domain and `<svc>.<delegate-domain>`.

### Apex cert validation (`defang` repo)

Apex domains route via an A record, so Azure rejects CNAME cert validation with
`InvalidValidationMethod`. `aca.issueManagedCertificate` retries with **HTTP**
validation, which Container Apps completes on its own once the A record is in
place — no extra DNS record, so it works unattended in the CD task. TXT
validation (the interactive `_dnsauth` dance) stays as the last resort for the
CLI path. Lands in DefangLabs/defang#2156, since it sits next to `aca.IssueCert`.

The DNS wait needs no apex special-case: `dns.CheckDomainDNSReady` already
accepts an A record whose addresses match those of the app FQDN, which is the
environment's static IP — exactly what an apex A record must point at.

## GCP

`FindZones` (`provider/defanggcp/gcp/dnszone.go`) lists the deploy project's
Cloud DNS zones once and matches every ingress service's `domainname` plus its
aliases on the Compose `default` network. Matching is case-insensitive, ignores
a trailing dot, understands `*.` wildcards, and chooses the longest eligible DNS
suffix.

### Authorizing a Cloud DNS zone

Seeing a zone in the deploy project does not prove that a Compose project owns
it. This matters when several teams or applications share one GCP project: a
service could otherwise set `domainname` to another application's hostname and
cause Defang's project-wide DNS identity to overwrite that record.

The zone owner must therefore add this exact whitespace-delimited marker to the
public managed zone's description:

```text
defang.dev/byod-dns=authorized
```

For example, while preserving any useful existing description:

```bash
gcloud dns managed-zones update ZONE_NAME \
  --project=GCP_PROJECT_ID \
  --description="Customer DNS zone defang.dev/byod-dns=authorized"
```

The marker authorizes Defang deployments using that project's CD identity to
create and delete BYOD routing and certificate-validation records anywhere in
the zone. In a shared project, prefer a dedicated subzone for the application
and authorize that narrower zone. Defang never adds the marker implicitly; a
zone owner must make this trust decision explicitly, including for an existing
Defang delegate zone.

Unmarked and private zones are ignored. If an unmarked child zone is a closer
suffix than an authorized parent zone, the authorized parent wins. Equivalent
matches use the managed-zone name as a stable tie-breaker.

### Records and certificates

For a hostname in an authorized zone, the provider creates:

- an A record pointing the hostname to the global load-balancer address;
- a Certificate Manager DNS authorization and its CNAME challenge; and
- a managed certificate attached to the existing certificate map with an exact
  hostname entry.

A DNS authorization covers both a base domain and its wildcard. Requests for
`example.com` and `*.example.com`, including requests from different services,
therefore share one authorization and one CNAME challenge. They still receive
separate certificates and certificate-map hostname entries so Certificate
Manager can select the correct certificate for each SNI name.

Hostnames are normalized and sorted before deduplication, and requesting
service names are sorted. If multiple services request the same normalized
hostname, Defang creates one shared certificate-map entry and warns with the
sorted service list, independent of Go map iteration order.

### Safe fallback

When no authorized zone matches, an ordinary hostname still gets a
load-balancer-authorized managed certificate. Defang does not write Cloud DNS
records; the domain owner points the hostname directly at the load balancer's IP
address, and Certificate Manager activates the certificate after the public DNS
and port 443 path are ready. No redeploy is needed.

Certificate Manager does not support wildcard names with load-balancer
authorization. A wildcard with no authorized zone is skipped with an actionable
warning; use an authorized Cloud DNS zone or `defang cert gen` instead.

If listing Cloud DNS zones fails, all ordinary hostnames take the same
load-balancer-authorization fallback and wildcard names are skipped. Resource
registration failures after an authorized zone is selected are returned rather
than silently leaving a partial DNS/certificate configuration.

This restores behavior from the legacy Go GCP CD
(`defang-mvp/pulumi/cd/gcp/gcpcd/defangservice.go`), whose load-balancer
authorization branch was the production path because GCP never populated
`ServiceInfo.ZoneId`. DNS authorization and safe zone selection are deliberate
improvements over that implementation.

## One rule for a missing zone, on every cloud

"No authorized zone hosts this hostname" is a **normal answer**. AWS and Azure
warn and leave `defang cert gen` (ACME) available. GCP can do better for ordinary
hostnames: it creates a load-balancer-authorized managed certificate without
writing into any DNS zone. GCP wildcards still require DNS authorization, so
they warn and leave the ACME path available.

This matches the legacy TypeScript CD, which is the behaviour a cutover must not
regress: `defang-mvp/pulumi/cd/aws/defang_service.ts` gated the whole Route 53 +
ACM block on `serviceInfo.zoneId`, fell back to `createCertsAcme` when the CLI had
marked the service `useAcmeCert`, and otherwise did nothing (a `TODO` for BYOC
domains without an explicit zone id) — never an error. It also passed the
default-network `aliases` alongside `domainname`, which is why zone resolution here
is per hostname rather than per service.

The only asymmetry left is *scope of the search*, which is a cloud constraint
rather than a choice: AWS can reach zones in another account through
`x-defang-dns-role`, while Azure has no cross-subscription equivalent of
`AssumeRole`, so it searches the deploy subscription only.

## SDK regeneration

`ProjectInputs`/`ServiceInputs` changes require regenerating the affected
provider schema and SDK. Run `make provider schema go_sdk` and commit any
generated changes alongside the source change (the pre-push hook enforces a
clean `sdk/v2/`).

## Files

pulumi-defang:
- `provider/defangaws/aws/route53.go` — `ErrZoneNotFound` + `asZoneNotFound`
- `provider/defangaws/aws/cert.go` — skip BYOD records when no hosted zone exists
- `provider/defangazure/azure/dnszone.go` — `FindZones` + `bestZoneMatch`
- `cd/program/azure.go` — `findByodZones` + `DnsZones` wiring + BYOD cert jobs
- `provider/defangazure/project.go` — `DnsZones` input + threading
- `provider/defangazure/service.go` — `DnsZones` input + threading
- `provider/defangazure/azure/containerapp.go` — `CreateContainerApp` param + call
- `provider/defangazure/azure/customdomain.go` — `CreateByodDomain`
- `provider/defanggcp/gcp/dnszone.go` — trusted longest-suffix zone selection
- `provider/defanggcp/gcp/byodcert.go` — A records, DNS/LB authorization, cert-map entries
- `cd/program/gcp.go` — GCP zone discovery and `DnsZones` wiring
- `sdk/v2/**` — regenerated

defang:
- `src/pkg/clouds/azure/aca/cert.go` — HTTP validation for apex domains

## Edge cases / notes

- **Not delegated at registrar:** the zone may exist without its NS being
  delegated from the parent at the registrar; cert validation then fails. That is
  the user's responsibility, on every cloud. Records are still created.
- **Multiple matching zones:** longest authorized suffix wins. GCP ignores
  unmarked zones and breaks equivalent matches by managed-zone name.
- **Record ownership:** Pulumi owns the records it creates, so `compose down`
  deletes them (the zone itself is untouched). A pre-existing record at the same
  name conflicts on create — the same ownership model on AWS and Azure.
