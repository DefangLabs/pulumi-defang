// Custom-domain support for Container Apps via the project delegate domain.
//
// The defang CLI creates the public DNS zone (PrepareDomainDelegation) in the
// project resource group before this provider runs, and tells Fabric to point
// NS records at it. EnsureDomainZone then imports that zone into Pulumi state so
// teardown (`defang compose down`) deletes it instead of orphaning it. Inside
// the zone, Pulumi owns:
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

// EnsureDomainZone adopts the project's public DNS zone (infra.Domain) into
// Pulumi state by importing it. The zone is provisioned out-of-band by the
// defang CLI (PrepareDomainDelegation) before the CD task runs; until now Pulumi
// only managed the records inside it, so the zone itself was orphaned on
// teardown. Importing it makes `defang compose down` delete the zone — and
// every record set within it — as part of the normal destroy.
//
// Deliberately NO RetainOnDelete: cleanup on teardown is the whole point (unlike
// EnsureKeyVault, which retains the vault to preserve user secrets). The args
// mirror a public zone exactly so the import doesn't propose a replacement.
//
// The import ID is constructed (see domainZoneID) rather than fetched via
// dns.LookupZone: every component is already known, so a lookup invoke would
// only add an ARM round-trip without buying safety — a missing zone fails the
// import just the same. It also keeps `pulumi preview` from failing before the
// CLI has created the zone (an invoke runs in preview; an import is a no-op).
//
// Returns (nil, nil) when the project has no delegate domain. The returned zone
// is stored on infra.DomainZone and referenced by CreateCustomDomain so the
// per-service records depend on it (created after / deleted before the zone).
func EnsureDomainZone(
	ctx *pulumi.Context,
	projectName string,
	infra *SharedInfra,
	parentOpt pulumi.ResourceOrInvokeOption,
) (*dns.Zone, error) {
	if infra == nil || infra.Domain == "" {
		return nil, nil //nolint:nilnil // no delegate domain; nothing to import
	}

	importID := domainZoneID(resolveSubscriptionID(ctx), ProjectResourceGroupName(ctx, projectName), infra.Domain)

	// Public DNS zones always live at location "global". ResourceGroupName uses
	// the live RG resource handle (not the computed name) so the import is
	// ordered after the RG is adopted into state.
	zone, err := dns.NewZone(ctx, "domain-zone", &dns.ZoneArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		ZoneName:          pulumi.String(infra.Domain),
		Location:          pulumi.String("global"),
		ZoneType:          dns.ZoneTypePublic,
	}, parentOpt, pulumi.Import(pulumi.ID(importID)))
	if err != nil {
		return nil, fmt.Errorf("importing DNS zone %q: %w", infra.Domain, err)
	}
	return zone, nil
}

// domainZoneID builds the ARM resource ID of a public DNS zone from its parts.
// Azure's canonical ID lowercases the provider path segment ("dnszones"), so we
// match that exactly — a casing mismatch can make the import propose a
// replacement. This is the same value dns.LookupZone would return.
func domainZoneID(subscriptionID, resourceGroup, zoneName string) string {
	return fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/dnszones/%s",
		subscriptionID, resourceGroup, zoneName,
	)
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
	// Reference the imported zone's Name (same value as infra.Domain) so each
	// record set depends on the zone resource: created after the zone is adopted,
	// and deleted before it on teardown. Fall back to the literal domain when the
	// zone wasn't imported (e.g. standalone callers that set Domain directly).
	var zoneName pulumi.StringInput = pulumi.String(infra.Domain)
	if infra.DomainZone != nil {
		zoneName = infra.DomainZone.Name
	}
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
