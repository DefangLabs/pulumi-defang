// Custom-domain support for Container Apps via the project delegate domain.
//
// The defang CLI creates the public DNS zone (PrepareDomainDelegation) in the
// project resource group before this provider runs, and tells Fabric to point
// NS records at it. Inside the zone, Pulumi owns:
//
//  1. A CNAME `<service>.<delegate-domain>` → the Container App's stable FQDN
//     (`<appName>.<env.DefaultDomain>`), so the host header reaches ACA.
//  2. An `asuid.<service>` TXT record containing the Container App's
//     `customDomainVerificationId`, which Azure inspects to prove the operator
//     controls the hostname before issuing a cert.
//  3. An ACA ManagedCertificate over the hostname (CNAME validation). Azure
//     waits on (1)+(2) to validate, so the cert resource declares DependsOn on
//     both records.
//
// The ManagedCertificate is *not* declared here either: Azure rejects cert
// creation with RequireCustomHostnameInEnvironment until the hostname is
// already registered as a customDomains entry on a Container App in the env,
// which in turn requires the asuid TXT (this file) to be in DNS — a cycle
// that single-pass Pulumi can't break. Both the cert and the customDomains
// binding stay with the CLI's IssueCert flow (defang/src/pkg/cli/client/byoc/
// azure/cert.go), which already sequences register-with-Disabled → cert →
// re-bind-with-SniEnabled correctly. See defang/defangdomain.md for the
// design notes.

package azure

import (
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/app/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/dns/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// CustomDomainResult bundles the records created for a service's custom domain.
// All fields are nil when the creator returns nothing to do. Cname and A are
// mutually exclusive: subdomains route via Cname, apex domains via A (a CNAME is
// illegal at the zone apex).
type CustomDomainResult struct {
	Cname *dns.RecordSet
	A     *dns.RecordSet
	Asuid *dns.RecordSet
}

// CreateCustomDomain provisions per-service DNS records under infra.Domain.
// Returns (nil, nil) — and creates nothing — when the project has no delegate
// domain or when the service does not expose a public ingress.
//
// The CNAME target is the Container App's stable hostname
// (`<appName>.<env.DefaultDomain>`); the per-revision LatestRevisionFqdn would
// move on every deploy and so would the record. The asuid TXT carries the
// CA's CustomDomainVerificationId so the cert-issuance step Azure runs out
// of band can authorise itself without any manual TXT publishing.
func CreateCustomDomain(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	containerApp *app.ContainerApp,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*CustomDomainResult, error) {
	// No delegate domain configured → keep the CA's auto-FQDN as the only entry.
	if infra == nil || infra.Domain == "" {
		return nil, nil //nolint:nilnil // nothing to provision; the caller treats nil as "skipped" without inspecting fields
	}
	// Internal-only services never need a public hostname, so skip them.
	// Public-vs-internal network filtering mirrors buildIngress (which today
	// also doesn't consult top-level networks) — add it here if that changes.
	if !svc.HasIngressPorts() {
		return nil, nil //nolint:nilnil // nothing to provision; the caller treats nil as "skipped" without inspecting fields
	}

	rgName := infra.ResourceGroup.Name
	zoneName := pulumi.String(infra.Domain)
	tags := ServiceTags(serviceName)

	// CNAME — `<service>.<domain>` points at the app's stable FQDN. Using the
	// env's DefaultDomain (not LatestRevisionFqdn) keeps the record valid
	// across new revisions on the same app.
	target := pulumi.Sprintf("%s.%s", containerApp.Name, infra.Environment.DefaultDomain)
	cname, err := dns.NewRecordSet(ctx, serviceName+"-cname", &dns.RecordSetArgs{
		ResourceGroupName:     rgName,
		ZoneName:              zoneName,
		RelativeRecordSetName: pulumi.String(common.ServiceLabel(serviceName)),
		RecordType:            pulumi.String("CNAME"),
		Ttl:                   pulumi.Float64(60),
		CnameRecord: &dns.CnameRecordArgs{
			Cname: target,
		},
		Metadata: tags,
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating CNAME for %s: %w", serviceName, err)
	}

	// asuid TXT — Azure validates ownership by reading this record before
	// issuing the managed cert. The verification ID is per-app and stable
	// after the CA is created.
	asuid, err := dns.NewRecordSet(ctx, serviceName+"-asuid", &dns.RecordSetArgs{
		ResourceGroupName:     rgName,
		ZoneName:              zoneName,
		RelativeRecordSetName: pulumi.String("asuid." + common.ServiceLabel(serviceName)),
		RecordType:            pulumi.String("TXT"),
		Ttl:                   pulumi.Float64(60),
		TxtRecords: dns.TxtRecordArray{
			&dns.TxtRecordArgs{
				Value: pulumi.StringArray{containerApp.CustomDomainVerificationId},
			},
		},
		Metadata: tags,
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating asuid TXT for %s: %w", serviceName, err)
	}

	return &CustomDomainResult{Cname: cname, Asuid: asuid}, nil
}

// CreateByodDomain provisions the per-service DNS records for a "bring your own
// domain" custom hostname (svc.DomainName) in the customer's *own* public DNS
// zone, identified by dnsZoneID (an ARM resource ID the CLI resolved via
// DNS.FindZone and threaded through ServiceInfo.ZoneId). This is the Azure
// analogue of the AWS BYOD path: when a zone for the domain exists in the
// subscription, Defang writes records there directly (and the CD program issues
// a managed cert) instead of the ACME fallback.
//
// Records created in the zone (resource group + zone name parsed from
// dnsZoneID):
//   - CNAME `<relative>` → the Container App's stable FQDN.
//   - TXT `asuid[.<relative>]` carrying the App's CustomDomainVerificationId.
//
// Returns (nil, nil) when there is nothing to do: no custom domain, no zone id,
// or the service has no public ingress. Apex domains (DomainName == zone name)
// are not supported because Azure DNS rejects a CNAME at the zone apex; such a
// service falls through to its delegate-domain / auto FQDN.
func CreateByodDomain(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	containerApp *app.ContainerApp,
	infra *SharedInfra,
	dnsZoneID string,
	opts ...pulumi.ResourceOption,
) (*CustomDomainResult, error) {
	if svc.DomainName == "" || dnsZoneID == "" || !svc.HasIngressPorts() {
		return nil, nil //nolint:nilnil // nothing to provision; caller treats nil as "skipped"
	}

	rgName, zoneName, ok := parseDNSZoneID(dnsZoneID)
	if !ok {
		// A malformed zone id shouldn't happen (the CLI resolves a real ARM id),
		// but warn-and-skip rather than fail: the Container App still serves on its
		// default FQDN, and a hard error would block an otherwise-healthy deploy.
		_ = ctx.Log.Warn(fmt.Sprintf(
			"service %s: unparseable DNS zone id %q; skipping BYOD DNS records (no managed cert for %q)",
			serviceName, dnsZoneID, svc.DomainName), nil)
		return nil, nil //nolint:nilnil // best-effort: skip BYOD records, deploy continues
	}

	tags := ServiceTags(serviceName)

	// Apex (domain == zone): a CNAME is illegal at the zone apex (RFC 1034), so
	// route with an A record at "@" pointing to the Container App environment's
	// static inbound IP, and put the asuid TXT at "asuid". Azure validates apex
	// managed certs via HTTP (see aca.IssueCert), which needs no extra record.
	//
	// CAVEAT: Environment.StaticIp is stable only for the life of the
	// environment — replacing it (e.g. a vnetConfiguration change, which forces
	// ManagedEnvironment replacement) changes the IP and breaks this record. An
	// apex custom domain accepts that trade-off; subdomains (CNAME) don't have it.
	if isApexDomain(svc.DomainName, zoneName) {
		a, err := dns.NewRecordSet(ctx, serviceName+"-byod-a", &dns.RecordSetArgs{
			ResourceGroupName:     pulumi.String(rgName),
			ZoneName:              pulumi.String(zoneName),
			RelativeRecordSetName: pulumi.String("@"),
			RecordType:            pulumi.String("A"),
			Ttl:                   pulumi.Float64(60),
			ARecords: dns.ARecordArray{
				&dns.ARecordArgs{Ipv4Address: infra.Environment.StaticIp.ToStringPtrOutput()},
			},
			Metadata: tags,
		}, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating BYOD A record for %s: %w", serviceName, err)
		}
		asuid, err := createAsuidTXT(ctx, serviceName+"-byod-asuid", rgName, zoneName, "asuid", containerApp, tags, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating BYOD asuid TXT for %s: %w", serviceName, err)
		}
		return &CustomDomainResult{A: a, Asuid: asuid}, nil
	}

	relative, ok := relativeRecordName(svc.DomainName, zoneName)
	if !ok {
		// The domain is not within the zone (apex is handled above). The CLI only
		// ever resolves a parent zone, so this shouldn't happen — warn rather than
		// fail (CodeRabbit flagged the prior silent skip as undiagnosable; a hard
		// error would block an otherwise-healthy deploy). The service still serves
		// on its default FQDN; no managed cert is issued for the custom domain.
		_ = ctx.Log.Warn(fmt.Sprintf(
			"service %s: custom domain %q is not within DNS zone %q; skipping BYOD DNS records (no managed cert for it)",
			serviceName, svc.DomainName, zoneName), nil)
		return nil, nil //nolint:nilnil // best-effort: skip BYOD records, deploy continues
	}

	target := pulumi.Sprintf("%s.%s", containerApp.Name, infra.Environment.DefaultDomain)
	cname, err := dns.NewRecordSet(ctx, serviceName+"-byod-cname", &dns.RecordSetArgs{
		ResourceGroupName:     pulumi.String(rgName),
		ZoneName:              pulumi.String(zoneName),
		RelativeRecordSetName: pulumi.String(relative),
		RecordType:            pulumi.String("CNAME"),
		Ttl:                   pulumi.Float64(60),
		CnameRecord: &dns.CnameRecordArgs{
			Cname: target,
		},
		Metadata: tags,
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating BYOD CNAME for %s: %w", serviceName, err)
	}

	asuid, err := createAsuidTXT(ctx, serviceName+"-byod-asuid", rgName, zoneName, "asuid."+relative, containerApp, tags, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating BYOD asuid TXT for %s: %w", serviceName, err)
	}

	return &CustomDomainResult{Cname: cname, Asuid: asuid}, nil
}

// createAsuidTXT creates the `asuid[.<sub>]` TXT record that carries the
// Container App's CustomDomainVerificationId — the value Azure reads to prove
// the operator controls the hostname before binding it / issuing a cert.
func createAsuidTXT(
	ctx *pulumi.Context,
	logicalName, rgName, zoneName, relativeName string,
	containerApp *app.ContainerApp,
	tags pulumi.StringMapInput,
	opts ...pulumi.ResourceOption,
) (*dns.RecordSet, error) {
	return dns.NewRecordSet(ctx, logicalName, &dns.RecordSetArgs{
		ResourceGroupName:     pulumi.String(rgName),
		ZoneName:              pulumi.String(zoneName),
		RelativeRecordSetName: pulumi.String(relativeName),
		RecordType:            pulumi.String("TXT"),
		Ttl:                   pulumi.Float64(60),
		TxtRecords: dns.TxtRecordArray{
			&dns.TxtRecordArgs{
				Value: pulumi.StringArray{containerApp.CustomDomainVerificationId},
			},
		},
		Metadata: tags,
	}, opts...)
}

// ByodRecordEligible reports whether CreateByodDomain would create DNS records
// for the given custom domain + zone id: the zone id must parse and the domain
// must be the zone apex or a subdomain of it. Exported so the CD program's cert
// scheduler enqueues a managed-cert job only for hostnames whose records will
// actually exist (otherwise aca.IssueCert waits out its DNS timeout for nothing).
// Callers must also confirm the service has public ingress.
func ByodRecordEligible(domain, dnsZoneID string) bool {
	if domain == "" || dnsZoneID == "" {
		return false
	}
	_, zoneName, ok := parseDNSZoneID(dnsZoneID)
	if !ok {
		return false
	}
	if isApexDomain(domain, zoneName) {
		return true
	}
	_, ok = relativeRecordName(domain, zoneName)
	return ok
}

// isApexDomain reports whether domain is the apex of zoneName (case- and
// trailing-dot-insensitive), e.g. "example.com" in zone "example.com".
func isApexDomain(domain, zoneName string) bool {
	d := strings.ToLower(strings.TrimSuffix(domain, "."))
	z := strings.ToLower(strings.TrimSuffix(zoneName, "."))
	return d == z
}

// parseDNSZoneID extracts the resource group and zone name from an Azure DNS
// zone ARM resource ID of the form
// /subscriptions/<sub>/resourceGroups/<rg>/providers/Microsoft.Network/dnszones/<zone>.
// Segment matching is case-insensitive (ARM ids vary in casing); the zone name
// is the final path segment. Returns ok=false if the id lacks a resource group
// or a final segment.
func parseDNSZoneID(id string) (resourceGroup, zoneName string, ok bool) {
	parts := strings.Split(strings.Trim(id, "/"), "/")
	for i, p := range parts {
		if strings.EqualFold(p, "resourceGroups") && i+1 < len(parts) {
			resourceGroup = parts[i+1]
			break
		}
	}
	if len(parts) > 0 {
		zoneName = parts[len(parts)-1]
	}
	if resourceGroup == "" || zoneName == "" || strings.EqualFold(zoneName, "dnszones") {
		return "", "", false
	}
	return resourceGroup, zoneName, true
}

// relativeRecordName returns the record name relative to zoneName for a fully
// qualified domain (e.g. domain "api.example.com" in zone "example.com" → "api").
// ok is false for the zone apex (domain == zoneName) — Azure DNS rejects a CNAME
// at the apex — or when domain is not actually within the zone.
func relativeRecordName(domain, zoneName string) (string, bool) {
	d := strings.ToLower(strings.TrimSuffix(domain, "."))
	z := strings.ToLower(strings.TrimSuffix(zoneName, "."))
	if d == z {
		return "", false // apex CNAME unsupported
	}
	if suffix := "." + z; strings.HasSuffix(d, suffix) {
		return domain[:len(d)-len(suffix)], true
	}
	return "", false
}
