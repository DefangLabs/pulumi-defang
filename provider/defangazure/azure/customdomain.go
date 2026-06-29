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

// CustomDomainResult bundles the records created for a service's custom
// domain. Both fields are nil when CreateCustomDomain returns nothing to do.
type CustomDomainResult struct {
	Cname *dns.RecordSet
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
		return nil, fmt.Errorf("service %s: unparseable DNS zone id %q", serviceName, dnsZoneID)
	}
	relative, ok := relativeRecordName(svc.DomainName, zoneName)
	if !ok {
		// Either an apex record (unsupported) or the domain isn't under the zone.
		return nil, nil //nolint:nilnil // nothing to provision; caller treats nil as "skipped"
	}

	tags := ServiceTags(serviceName)
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

	asuid, err := dns.NewRecordSet(ctx, serviceName+"-byod-asuid", &dns.RecordSetArgs{
		ResourceGroupName:     pulumi.String(rgName),
		ZoneName:              pulumi.String(zoneName),
		RelativeRecordSetName: pulumi.String("asuid." + relative),
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
		return nil, fmt.Errorf("creating BYOD asuid TXT for %s: %w", serviceName, err)
	}

	return &CustomDomainResult{Cname: cname, Asuid: asuid}, nil
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
