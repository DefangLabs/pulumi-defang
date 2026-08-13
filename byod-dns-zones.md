# BYOD: finding an existing DNS zone

## Goal

When a service sets a custom `domainname` ("bring your own domain"), Defang
should check whether a DNS zone that can host that domain **already exists** in
the account being deployed to. If one does, manage the records there and issue a
cloud-managed TLS certificate. If none does, that is a normal outcome, not an
error: skip the BYOD records, keep the service reachable on its delegate-domain
hostname, and leave the ACME / `defang cert generate` path available.

Every cloud answers the same two questions, so they should behave the same way:

1. **Which existing zone can host this domain?** — longest DNS-suffix match.
2. **What if none can?** — degrade, don't fail.

This document covers where that lookup lives per cloud and what it does. The
per-cloud record and certificate mechanics differ and are described where they
diverge.

## Shared decisions

- **The lookup runs in the CD task**, inside the customer's cloud. The deploy
  identity there already has DNS read permission and knows which account it is
  in, so a user running `defang compose up` needs no extra client-side role.
- **A managed cert is issued whenever a zone is used.**
- **No restriction on which zone matches** — longest DNS-suffix match, no
  ownership or tag filter. Whichever existing zone is the closest parent wins.
- **"No zone" is not an error.** It degrades to the delegate domain + ACME.

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

Not implemented. `ServiceInfo.ZoneId` is unused on the GCP path and BYOD domains
go through ACME. When it is added, it should answer the same two questions: a
longest-suffix match over the project's Cloud DNS managed zones, and a warn-and-
degrade response when nothing matches.

## SDK regeneration

`ProjectInputs`/`ServiceInputs` changes require regenerating the Azure SDK:
`make schema` + `make sdks` (azure), committed alongside the source change (the
pre-push hook enforces a clean `sdk/v2/`).

## Files

pulumi-defang:
- `provider/defangaws/aws/route53.go` — `IsZoneNotFound`
- `provider/defangaws/aws/cert.go` — skip BYOD records when no hosted zone exists
- `provider/defangazure/azure/dnszone.go` — `FindZone` + `bestZoneMatch`
- `cd/program/azure.go` — `findByodZones` + `DnsZones` wiring + BYOD cert jobs
- `provider/defangazure/project.go` — `DnsZones` input + threading
- `provider/defangazure/service.go` — `DnsZoneId` input + threading
- `provider/defangazure/azure/containerapp.go` — `CreateContainerApp` param + call
- `provider/defangazure/azure/customdomain.go` — `CreateByodDomain`
- `sdk/v2/**` — regenerated

defang:
- `src/pkg/clouds/azure/aca/cert.go` — HTTP validation for apex domains

## Edge cases / notes

- **Not delegated at registrar:** the zone may exist without its NS being
  delegated from the parent at the registrar; cert validation then fails. That is
  the user's responsibility, on every cloud. Records are still created.
- **Multiple matching zones:** longest-suffix wins; ties are not expected (zone
  names are unique within an account).
- **Record ownership:** Pulumi owns the records it creates, so `compose down`
  deletes them (the zone itself is untouched). A pre-existing record at the same
  name conflicts on create — the same ownership model on AWS and Azure.
