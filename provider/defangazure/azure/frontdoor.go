// Wildcard custom-domain support for Container Apps, via Azure Front Door.
//
// Container Apps binds custom domains one hostname at a time — each with its own
// `asuid.<label>` TXT record and its own certificate binding (see
// customdomain.go) — and its ingress routes strictly on the Host header,
// answering 404 to any hostname it wasn't told about. A wildcard hostname like
// `*.auth.example.com` therefore can't be served by ACA at all: there is no
// wildcard binding, and enumerating the subdomains means one ARM call and one
// certificate each, against the environment's certificate limits.
//
// Azure Front Door can serve it. A Standard profile accepts a wildcard custom
// domain and issues an Azure-managed certificate for it, validated by a DNS TXT
// record, so one profile and one certificate cover every subdomain and nothing
// has to be provisioned when a new subdomain starts being used.
//
// What gets created, and only when some service declares a wildcard hostname:
//
//  1. A project-scoped cdn.Profile plus cdn.AFDEndpoint (EnsureFrontDoor).
//  2. Per service, a cdn.AFDOriginGroup and cdn.AFDOrigin aimed at the Container
//     App's stable FQDN, a cdn.AFDCustomDomain per wildcard hostname, and a
//     cdn.Route joining them (CreateWildcardDomain).
//
// The origin is sent OriginHostHeader = the app's own FQDN, because ACA would
// 404 the hostname the client used. Front Door passes that original hostname on
// in X-Forwarded-Host, so an app serving per-subdomain content must read that
// header rather than Host. This differs from the AWS path, where the ALB
// forwards the client's Host untouched.
//
// Two DNS records make a wildcard domain live: a TXT at `_dnsauth.<label>`
// carrying the validation token, and a wildcard CNAME at `*.<label>` pointing at
// the Front Door endpoint. The provider writes both itself into whichever zone it
// can reach: the project's delegate-domain zone, which it owns, or a BYOD zone
// the CD task found in the deploy subscription (FindZone). A hostname whose zone
// is in neither place — another subscription, or another cloud's DNS entirely —
// gets the two records logged for the operator instead, the same
// warn-and-degrade rule the AWS BYOD path follows. No CAA record is written: Front Door only needs one where the zone
// already restricts issuance, and adding a CAA where none exists would restrict
// every other certificate in the zone.
//
// Caveat worth knowing about: Front Door does not auto-rotate a managed
// certificate for a wildcard domain, though it does for subdomains and apex
// domains. 45 days before expiry the domain moves to Pending revalidation and
// the TXT record has to be replaced with a fresh token. A redeploy re-reads the
// token and rewrites the record wherever the provider owns the zone; everywhere
// else it's a manual step.

package azure

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/app/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/cdn/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/dns/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	// frontDoorSKU is the cheapest tier that serves wildcard custom domains with
	// an Azure-managed certificate. Premium adds managed WAF rulesets and Private
	// Link origins; this path uses neither.
	frontDoorSKU = "Standard_AzureFrontDoor"

	// frontDoorLocation is the only location an AFD profile or endpoint accepts —
	// they are global resources.
	frontDoorLocation = "Global"

	// dnsAuthPrefix names the TXT record Front Door reads to validate ownership
	// of a domain: `_dnsauth.<label>`.
	dnsAuthPrefix = "_dnsauth"

	// dnsAuthTTL is the TTL Front Door's docs prescribe for the validation TXT
	// record (1 hour). A shorter TTL isn't better here: on revalidation the old
	// record must have expired before the new token is read, so a long-lived
	// record only delays a rotation that already runs 45 days ahead of expiry.
	dnsAuthTTL = 3600
)

// errWildcardNeedsProject is returned when a service asks for a wildcard
// hostname on the standalone Service path, which has no project-scoped Front
// Door profile to route it through.
var errWildcardNeedsProject = errors.New(
	"wildcard domains on Azure need an Azure Front Door profile, which only the Project component provisions",
)

// FrontDoorInfra holds the project-scoped Front Door resources that every
// service with a wildcard hostname shares.
type FrontDoorInfra struct {
	Profile  *cdn.Profile
	Endpoint *cdn.AFDEndpoint
}

// WildcardDomainResult bundles what CreateWildcardDomain provisioned for one
// service. All fields are empty when it had nothing to do.
type WildcardDomainResult struct {
	// Domains holds one AFDCustomDomain per wildcard hostname, in the order the
	// hostnames appear in the service's compose config.
	Domains []*cdn.AFDCustomDomain
	// Records holds the DNS record sets written into the zone that hosts the
	// hostnames — the delegate-domain zone or a resolved BYOD zone. Empty when
	// neither can host them, in which case the records were logged for the
	// operator instead.
	Records []*dns.RecordSet
}

// HasWildcardHostname reports whether any service in the project asks to be
// reachable on a wildcard hostname, which is what makes a Front Door profile
// worth its cost. Used by the project dispatcher to decide whether to call
// EnsureFrontDoor at all.
func HasWildcardHostname(services compose.Services) bool {
	for _, svc := range services {
		if len(wildcardHostnames(svc)) > 0 {
			return true
		}
	}
	return false
}

// wildcardHostnames returns the service's BYOD hostnames that are wildcards.
// A service with no ingress ports gets none: Front Door needs a public origin to
// forward to, and an internal-only service has no reachable hostname to offer.
func wildcardHostnames(svc compose.ServiceConfig) []string {
	if !svc.HasIngressPorts() {
		return nil
	}
	var wildcards []string
	for _, hostname := range common.ByodHostnames(svc) {
		if common.IsWildcardHost(hostname) {
			wildcards = append(wildcards, common.NormalizeDNS(hostname))
		}
	}
	return wildcards
}

// EnsureFrontDoor creates the project's Front Door profile and endpoint.
//
// Returns (nil, nil) when no service declares a wildcard hostname, so a project
// that doesn't need Front Door never pays for one. Both resources are named by
// Pulumi's auto-naming rather than derived from the project: an endpoint name is
// capped at 46 characters and a profile name is unique only within its resource
// group, and auto-naming satisfies both without a truncation scheme. The
// endpoint's hostname — the CNAME target every wildcard domain points at — is
// stable across updates because Pulumi persists the generated name.
func EnsureFrontDoor(
	ctx *pulumi.Context,
	services compose.Services,
	infra *SharedInfra,
	parentOpt pulumi.ResourceOption,
) (*FrontDoorInfra, error) {
	if infra == nil || !HasWildcardHostname(services) {
		return nil, nil //nolint:nilnil // no wildcard hostnames; the caller treats nil as "not needed"
	}

	profile, err := cdn.NewProfile(ctx, "front-door", &cdn.ProfileArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		Location:          pulumi.String(frontDoorLocation),
		Sku:               &cdn.SkuArgs{Name: pulumi.String(frontDoorSKU)},
	}, parentOpt)
	if err != nil {
		return nil, fmt.Errorf("creating Front Door profile: %w", err)
	}

	endpoint, err := cdn.NewAFDEndpoint(ctx, "front-door", &cdn.AFDEndpointArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		ProfileName:       profile.Name,
		Location:          pulumi.String(frontDoorLocation),
		EnabledState:      pulumi.String("Enabled"),
	}, parentOpt)
	if err != nil {
		return nil, fmt.Errorf("creating Front Door endpoint: %w", err)
	}

	return &FrontDoorInfra{Profile: profile, Endpoint: endpoint}, nil
}

// CreateWildcardDomain routes the service's wildcard hostnames through the
// project's Front Door endpoint to its Container App.
//
// Returns (nil, nil) — and creates nothing — when the project has no Front Door
// (no service asked for a wildcard) or when this service declares no wildcard
// hostname of its own. Non-wildcard hostnames are left alone: ACA binds those
// directly, with a managed certificate of its own, and sending them through
// Front Door would add a second TLS hop and a second bill for nothing.
func CreateWildcardDomain(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	containerApp *app.ContainerApp,
	infra *SharedInfra,
	dnsZoneID string,
	opts ...pulumi.ResourceOption,
) (*WildcardDomainResult, error) {
	hostnames := wildcardHostnames(svc)
	if len(hostnames) == 0 {
		return nil, nil //nolint:nilnil // this service has no wildcard hostname
	}
	if infra == nil || infra.FrontDoor == nil {
		// Only the Project component provisions a Front Door profile, so a
		// standalone Service can't serve a wildcard. Fail rather than deploy a
		// service that looks configured for a hostname nothing will answer on.
		return nil, fmt.Errorf("service %s (%s): %w", serviceName, strings.Join(hostnames, ", "), errWildcardNeedsProject)
	}

	fd := infra.FrontDoor
	rgName := infra.ResourceGroup.Name

	originGroup, err := cdn.NewAFDOriginGroup(ctx, serviceName+"-origin-group", &cdn.AFDOriginGroupArgs{
		ResourceGroupName: rgName,
		ProfileName:       fd.Profile.Name,
		LoadBalancingSettings: &cdn.LoadBalancingSettingsParametersArgs{
			SampleSize:                      pulumi.Int(4),
			SuccessfulSamplesRequired:       pulumi.Int(3),
			AdditionalLatencyInMilliseconds: pulumi.Int(50),
		},
		// No HealthProbeSettings, which leaves probing off. The group holds a
		// single origin, so a probe can only ever take the one backend out of
		// rotation with nothing to fail over to — and its default probe of "/"
		// would call an app unhealthy for answering anything but 200, which a
		// redirect to a login page (the very case this path exists for) does.
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Front Door origin group for %s: %w", serviceName, err)
	}

	// The Container App's stable hostname, same target CreateCustomDomain uses:
	// LatestRevisionFqdn would move on every deploy.
	appFQDN := pulumi.Sprintf("%s.%s", containerApp.Name, infra.Environment.DefaultDomain)
	_, err = cdn.NewAFDOrigin(ctx, serviceName+"-origin", &cdn.AFDOriginArgs{
		ResourceGroupName: rgName,
		ProfileName:       fd.Profile.Name,
		OriginGroupName:   originGroup.Name,
		HostName:          appFQDN,
		// ACA routes on Host and 404s a hostname it has no binding for, so the
		// app's own FQDN has to be what reaches it. The hostname the client asked
		// for arrives as X-Forwarded-Host.
		OriginHostHeader: appFQDN,
		HttpsPort:        pulumi.Int(443),
		HttpPort:         pulumi.Int(80),
		// The origin hostname is the real certificate subject, so name checking
		// costs nothing and keeps the hop authenticated.
		EnforceCertificateNameCheck: pulumi.Bool(true),
		Priority:                    pulumi.Int(1),
		Weight:                      pulumi.Int(1000),
		EnabledState:                pulumi.String("Enabled"),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Front Door origin for %s: %w", serviceName, err)
	}

	result := &WildcardDomainResult{}
	domainRefs := cdn.ActivatedResourceReferenceArray{}
	for i, hostname := range hostnames {
		domain, err := cdn.NewAFDCustomDomain(ctx, fmt.Sprintf("%s-wildcard-%d", serviceName, i), &cdn.AFDCustomDomainArgs{
			ResourceGroupName: rgName,
			ProfileName:       fd.Profile.Name,
			HostName:          pulumi.String(hostname),
			TlsSettings: &cdn.AFDDomainHttpsParametersArgs{
				// No MinimumTlsVersion: from API version 2025-06-01 it only
				// applies alongside a Customized cipherSuiteSetType, and Front
				// Door's own default minimum is already TLS 1.2.
				CertificateType: pulumi.String("ManagedCertificate"),
			},
		}, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating Front Door domain %s for %s: %w", hostname, serviceName, err)
		}
		result.Domains = append(result.Domains, domain)
		domainRefs = append(domainRefs, &cdn.ActivatedResourceReferenceArgs{Id: domain.ID()})

		records, err := publishValidation(ctx, serviceName, hostname, domain, fd, infra, dnsZoneID, opts...)
		if err != nil {
			return nil, err
		}
		result.Records = append(result.Records, records...)
	}

	// The route is created whatever state the domains are in. Front Door accepts
	// an unvalidated domain on a route — that is the normal add-domain-then-add-DNS
	// order — and starts serving it the moment validation completes.
	_, err = cdn.NewRoute(ctx, serviceName+"-route", &cdn.RouteArgs{
		ResourceGroupName:  rgName,
		ProfileName:        fd.Profile.Name,
		EndpointName:       fd.Endpoint.Name,
		OriginGroup:        &cdn.ResourceReferenceArgs{Id: originGroup.ID()},
		CustomDomains:      domainRefs,
		PatternsToMatch:    pulumi.StringArray{pulumi.String("/*")},
		SupportedProtocols: pulumi.StringArray{pulumi.String("Http"), pulumi.String("Https")},
		ForwardingProtocol: pulumi.String("HttpsOnly"),
		HttpsRedirect:      pulumi.String("Enabled"),
		// The endpoint's own *.azurefd.net hostname stays unrouted: it would be a
		// second, unauthenticated way into the app that nobody asked for.
		LinkToDefaultDomain: pulumi.String("Disabled"),
		EnabledState:        pulumi.String("Enabled"),
		// No CacheConfiguration, which leaves caching off — these are application
		// origins, not static assets.
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Front Door route for %s: %w", serviceName, err)
	}

	return result, nil
}

// publishValidation gets the two DNS records a wildcard domain needs into place:
// the `_dnsauth.<label>` TXT carrying Front Door's validation token, and the
// `*.<label>` CNAME pointing at the Front Door endpoint.
//
// It writes them into whichever zone this deployment can reach: the project's
// delegate-domain zone, which the provider owns, or the BYOD zone the CD task
// found in the subscription (dnsZoneID, see FindZone). With neither it logs them
// — with the token, once Front Door has issued it — and returns no records,
// leaving the operator to add them wherever the zone actually lives.
func publishValidation(
	ctx *pulumi.Context,
	serviceName, hostname string,
	domain *cdn.AFDCustomDomain,
	fd *FrontDoorInfra,
	infra *SharedInfra,
	dnsZoneID string,
	opts ...pulumi.ResourceOption,
) ([]*dns.RecordSet, error) {
	token := domain.ValidationProperties.ValidationToken()
	// Strip the "*." so both records are named from the domain being validated:
	// "*.auth.example.com" validates via _dnsauth.auth.example.com.
	base := strings.TrimPrefix(hostname, common.WildcardPrefix)

	rgName, zoneName, label, ok := wildcardZoneTarget(base, infra, dnsZoneID)
	if !ok {
		logManualRecords(ctx, serviceName, hostname, base, token, fd.Endpoint.HostName)
		return nil, nil
	}

	tags := ServiceTags(serviceName)

	auth, err := dns.NewRecordSet(ctx, serviceName+"-"+common.SafeLabel(base)+"-dnsauth", &dns.RecordSetArgs{
		ResourceGroupName:     rgName,
		ZoneName:              zoneName,
		RelativeRecordSetName: pulumi.String(joinLabels(dnsAuthPrefix, label)),
		RecordType:            pulumi.String("TXT"),
		Ttl:                   pulumi.Float64(dnsAuthTTL),
		TxtRecords: dns.TxtRecordArray{
			&dns.TxtRecordArgs{Value: pulumi.StringArray{token}},
		},
		Metadata: tags,
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating validation TXT for %s: %w", hostname, err)
	}

	cname, err := dns.NewRecordSet(ctx, serviceName+"-"+common.SafeLabel(base)+"-wildcard", &dns.RecordSetArgs{
		ResourceGroupName:     rgName,
		ZoneName:              zoneName,
		RelativeRecordSetName: pulumi.String(joinLabels("*", label)),
		RecordType:            pulumi.String("CNAME"),
		Ttl:                   pulumi.Float64(60),
		CnameRecord:           &dns.CnameRecordArgs{Cname: fd.Endpoint.HostName},
		Metadata:              tags,
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating wildcard CNAME for %s: %w", hostname, err)
	}

	return []*dns.RecordSet{auth, cname}, nil
}

// wildcardZoneTarget picks the zone the wildcard's records go into and names
// them relative to it, given the domain being validated (the hostname with its
// "*." stripped). Reports false when no reachable zone can host them, which
// leaves the records to the operator.
//
// The delegate-domain zone is preferred: the provider created it, so records in
// it are torn down with the project. The BYOD zone found by the CD task comes
// second — this deployment writes records into it but does not own it, so they
// outlive nothing else. Only the zone resolved for the service's own
// `domainname` is considered, so a wildcard alias pointing outside that zone
// still falls through to the manual path.
func wildcardZoneTarget(
	base string,
	infra *SharedInfra,
	dnsZoneID string,
) (pulumi.StringInput, pulumi.StringInput, string, bool) {
	if infra == nil {
		return nil, nil, "", false
	}
	if label, inZone := relativeRecordName(base, infra.Domain); inZone && infra.DomainZone != nil {
		return infra.ResourceGroup.Name, infra.DomainZone.Name, label, true
	}
	if byodRG, byodZone, parsed := parseDNSZoneID(dnsZoneID); parsed {
		if label, inZone := relativeRecordName(base, byodZone); inZone {
			return pulumi.String(byodRG), pulumi.String(byodZone), label, true
		}
	}
	return nil, nil, "", false
}

// logManualRecords prints the two records the operator has to add for a wildcard
// hostname whose zone this deployment can't write to. It runs inside an apply so
// the validation token and endpoint hostname are resolved values; during a
// preview neither is known yet and nothing is printed.
func logManualRecords(
	ctx *pulumi.Context,
	serviceName, hostname, base string,
	token, endpointHost pulumi.StringOutput,
) {
	pulumi.All(token, endpointHost).ApplyT(func(vs []any) string {
		validationToken, _ := vs[0].(string)
		endpoint, _ := vs[1].(string)
		msg := fmt.Sprintf(
			"service %q: %s is served by Azure Front Door, but its DNS zone isn't managed by this deployment.\n"+
				"Add these records to the zone for %s to finish it:\n"+
				"  TXT    %s.%s    %q\n"+
				"  CNAME  %s          %s\n"+
				"Front Door does not auto-renew a wildcard certificate: the TXT record has to be "+
				"refreshed when the domain enters Pending revalidation, 45 days before expiry.",
			serviceName, hostname, base,
			dnsAuthPrefix, base, validationToken,
			hostname, endpoint,
		)
		_ = ctx.Log.Warn(msg, nil)
		return msg
	})
}

// relativeRecordName expresses fqdn relative to zone, the form Azure DNS record
// sets are named in — "auth.example.com" in zone "example.com" is "auth", and
// the zone apex is "@". Reports false when fqdn isn't inside zone, or when zone
// is empty.
func relativeRecordName(fqdn, zone string) (string, bool) {
	fqdn, zone = common.NormalizeDNS(fqdn), common.NormalizeDNS(zone)
	if zone == "" {
		return "", false
	}
	if fqdn == zone {
		return "@", true
	}
	suffix := "." + zone
	if !strings.HasSuffix(fqdn, suffix) {
		return "", false
	}
	return strings.TrimSuffix(fqdn, suffix), true
}

// joinLabels prefixes a relative record name with another label, collapsing the
// apex: "_dnsauth" + "@" is "_dnsauth", not "_dnsauth.@".
func joinLabels(prefix, relative string) string {
	if relative == "@" {
		return prefix
	}
	return prefix + "." + relative
}
