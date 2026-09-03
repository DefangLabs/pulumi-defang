package gcp

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/certificatemanager"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/dns"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var errConflictingByodZones = errors.New("conflicting Cloud DNS zones for one BYOD authorization domain")

type byodDnsAuthorization struct {
	zone     string
	resource *certificatemanager.DnsAuthorization
}

type byodHostnameRequest struct {
	hostname string
	services []string
}

// createByodCerts issues one Certificate Manager managed certificate per BYOD
// hostname and attaches it to the load balancer's certificate map, so a service
// with a `domainname` serves TLS on that name.
//
// Which authorization GCP uses depends on whether zone discovery found a Cloud
// DNS zone that can host the name (see FindZones):
//
//   - zone found: an A record pointing at the LB plus a DNS authorization, whose
//     challenge record Defang writes into that same zone. The certificate
//     validates on its own, before any traffic reaches the load balancer.
//   - no zone: load-balancer authorization — the certificate carries only its
//     domain, and GCP validates it by observing that the domain resolves to this
//     load balancer. It stays PROVISIONING until the owner points the name at the
//     LB's IP, then activates without a redeploy.
//
// The second case is the common one for a customer whose DNS lives outside the
// deploy project, and it is what the legacy GCP CD did for every BYOD domain:
// `ServiceInfo.ZoneId` was never populated on GCP, so its else-branch
// (defang-mvp pulumi/cd/gcp/gcpcd/defangservice.go:559-571) was the only path
// that ran. Dropping it would take TLS away from those services.
//
// One certificate per hostname, rather than one multi-domain certificate per
// service, so a single unvalidated name cannot block the others — the same
// reason the legacy CD gave for its per-alias certificates.
func createByodCerts(
	ctx *pulumi.Context,
	certMap *certificatemanager.CertificateMapResource,
	infra *SharedInfra,
	entries []LBServiceEntry,
	opts ...pulumi.ResourceOption,
) error {
	requests := byodHostnameRequests(entries)
	if err := validateByodAuthorizationZones(requests, infra.DnsZones); err != nil {
		return err
	}

	authorizations := map[string]byodDnsAuthorization{}
	for _, request := range requests {
		hostname := request.hostname
		if len(request.services) > 1 {
			warnf(ctx, "BYOD TLS: %q is requested by services %q; creating one shared certificate-map entry",
				hostname, request.services)
		}

		// Use a short hash of the normalized hostname so resource identity is
		// independent of which service asks for an overlapping name, while staying
		// below Certificate Manager's 63-character name limit.
		name := byodResourceName("cert", hostname)

		cert, err := createByodCert(ctx, name, hostname, infra, authorizations, opts...)
		if err != nil {
			return err
		}
		if cert == nil {
			continue // not certifiable; createByodCert has warned
		}

		// Hostname (not Matcher: PRIMARY) so this certificate is served for SNI
		// requests for exactly this name. The delegate-domain wildcard keeps the
		// PRIMARY slot as the fallback, so the two never compete.
		if _, err := certificatemanager.NewCertificateMapEntry(ctx, name+"-map-entry",
			&certificatemanager.CertificateMapEntryArgs{
				Map:          certMap.Name,
				Certificates: pulumi.StringArray{cert.ID()},
				Hostname:     pulumi.String(hostname),
			}, opts...); err != nil {
			return err
		}
	}
	return nil
}

func byodHostnameRequests(entries []LBServiceEntry) []byodHostnameRequest {
	requesters := map[string]map[string]struct{}{}
	for _, entry := range entries {
		for _, hostname := range common.ByodHostnames(entry.Config) {
			hostname = common.NormalizeDNS(hostname)
			if hostname == "" {
				continue
			}
			if requesters[hostname] == nil {
				requesters[hostname] = map[string]struct{}{}
			}
			requesters[hostname][entry.Name] = struct{}{}
		}
	}

	hostnames := make([]string, 0, len(requesters))
	for hostname := range requesters {
		hostnames = append(hostnames, hostname)
	}
	sort.Strings(hostnames)

	requests := make([]byodHostnameRequest, 0, len(hostnames))
	for _, hostname := range hostnames {
		services := make([]string, 0, len(requesters[hostname]))
		for service := range requesters[hostname] {
			services = append(services, service)
		}
		sort.Strings(services)
		requests = append(requests, byodHostnameRequest{hostname: hostname, services: services})
	}
	return requests
}

// validateByodAuthorizationZones rejects an internally inconsistent discovery
// result before registering any BYOD resources. A DNS authorization has one
// challenge record and therefore one destination zone; the same normalized
// authorization domain cannot safely point at two zones.
func validateByodAuthorizationZones(requests []byodHostnameRequest, zones map[string]string) error {
	authorizationZones := map[string]string{}
	for _, request := range requests {
		zone := zones[request.hostname]
		if zone == "" {
			continue
		}
		domain := authorizationDomain(request.hostname)
		if firstZone, ok := authorizationZones[domain]; ok && firstZone != zone {
			zoneA, zoneB := firstZone, zone
			if zoneB < zoneA {
				zoneA, zoneB = zoneB, zoneA
			}
			return fmt.Errorf("%w: BYOD TLS authorization domain %q resolved to %q and %q",
				errConflictingByodZones, domain, zoneA, zoneB)
		}
		authorizationZones[domain] = zone
	}
	return nil
}

// createByodCert creates the managed certificate for one BYOD hostname, plus the
// DNS records it needs when a zone was found for it. It returns a nil
// certificate — with a warning, never an error — when GCP cannot issue one, so a
// single uncertifiable hostname never fails the deploy.
func createByodCert(
	ctx *pulumi.Context,
	name, hostname string,
	infra *SharedInfra,
	authorizations map[string]byodDnsAuthorization,
	opts ...pulumi.ResourceOption,
) (*certificatemanager.Certificate, error) {
	zone := infra.DnsZones[hostname]
	if zone == "" {
		// Load-balancer authorization cannot cover a wildcard: GCP requires DNS
		// authorization for those. Issuing the certificate anyway would leave it
		// PROVISIONING forever, which reads as a stuck deploy rather than a
		// configuration problem.
		if common.IsWildcardHost(hostname) {
			warnf(ctx, "BYOD TLS: no Cloud DNS zone in this project hosts %q, and GCP cannot issue a "+
				"certificate for a wildcard without one. Create the zone in the deploy project and "+
				"redeploy, or run `defang cert gen` to issue a certificate via ACME.", hostname)
			//nolint:nilnil // documented contract: nil cert + nil error means "skip, already warned"
			return nil, nil
		}
		warnf(ctx, "BYOD TLS: no Cloud DNS zone in this project hosts %q, so its certificate uses "+
			"load balancer authorization: point %s at this load balancer's IP address and GCP "+
			"activates the certificate on its own, with no redeploy needed.", hostname, hostname)
		return certificatemanager.NewCertificate(ctx, name, &certificatemanager.CertificateArgs{
			Description: pulumi.StringPtr("Load balancer authorized certificate for " + hostname),
			Managed: &certificatemanager.CertificateManagedArgs{
				Domains: pulumi.StringArray{pulumi.String(hostname)},
			},
		}, opts...)
	}

	if err := createByodRecord(ctx, name, hostname, zone, infra, opts...); err != nil {
		return nil, err
	}

	// One DNS authorization covers a domain and its wildcard. Reuse it across
	// hostnames and services so example.com plus *.example.com publish one CNAME
	// challenge while still receiving separate certificates and map entries.
	authzDomain := authorizationDomain(hostname)
	authz, ok := authorizations[authzDomain]
	if ok && authz.zone != zone {
		return nil, fmt.Errorf("%w: BYOD TLS authorization domain %q resolved to %q and %q",
			errConflictingByodZones, authzDomain, authz.zone, zone)
	}
	if !ok {
		resource, err := createByodDnsAuthorization(ctx, authzDomain, zone, opts...)
		if err != nil {
			return nil, err
		}
		authz = byodDnsAuthorization{zone: zone, resource: resource}
		authorizations[authzDomain] = authz
	}

	return certificatemanager.NewCertificate(ctx, name, &certificatemanager.CertificateArgs{
		Description: pulumi.StringPtr("DNS authorized certificate for " + hostname),
		Managed: &certificatemanager.CertificateManagedArgs{
			Domains:           pulumi.StringArray{pulumi.String(hostname)},
			DnsAuthorizations: pulumi.StringArray{authz.resource.ID()},
		},
	}, opts...)
}

func createByodDnsAuthorization(
	ctx *pulumi.Context,
	domain, zone string,
	opts ...pulumi.ResourceOption,
) (*certificatemanager.DnsAuthorization, error) {
	name := byodResourceName("authz", domain)
	authz, err := certificatemanager.NewDnsAuthorization(ctx, name,
		&certificatemanager.DnsAuthorizationArgs{
			Description: pulumi.StringPtr("DNS authorization for " + domain),
			Domain:      pulumi.String(domain),
		}, opts...)
	if err != nil {
		return nil, err
	}

	// A FIXED_RECORD authorization (the default) publishes exactly one challenge
	// record, so read it off the output by index rather than creating resources
	// inside an ApplyT the way createWildcardCert still has to.
	record := authz.DnsResourceRecords.Index(pulumi.Int(0))
	if _, err := dns.NewRecordSet(ctx, name+"-record", &dns.RecordSetArgs{
		ManagedZone: pulumi.String(zone),
		Name:        record.Name().Elem(),
		Type:        record.Type().Elem(),
		Ttl:         pulumi.Int(60),
		Rrdatas:     pulumi.StringArray{record.Data().Elem()},
	}, opts...); err != nil {
		return nil, err
	}
	return authz, nil
}

// createByodRecord points a BYOD hostname at the load balancer from inside the
// zone that hosts it. An A record works at a zone apex as well as below it, so
// unlike Azure's CNAME there is no apex special case.
func createByodRecord(
	ctx *pulumi.Context,
	name, hostname, zone string,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) error {
	if infra.PublicIP == nil {
		// Standalone callers have no shared load balancer to point at. The
		// certificate is still worth creating; the record is not.
		warnf(ctx, "BYOD TLS: not creating a DNS record for %q because this project has no "+
			"load balancer address", hostname)
		return nil
	}
	_, err := dns.NewRecordSet(ctx, name+"-record", &dns.RecordSetArgs{
		ManagedZone: pulumi.String(zone),
		Name:        pulumi.String(hostname + "."),
		Type:        pulumi.String(string(common.RecordTypeA)),
		Ttl:         pulumi.Int(60),
		Rrdatas:     pulumi.StringArray{infra.PublicIP.Address},
	}, opts...)
	return err
}

// trimWildcard returns the name a wildcard hostname stands for, or hostname
// unchanged when it is not a wildcard.
func trimWildcard(hostname string) string {
	if common.IsWildcardHost(hostname) {
		return hostname[len(common.WildcardPrefix):]
	}
	return hostname
}

func authorizationDomain(hostname string) string {
	return common.NormalizeDNS(trimWildcard(common.NormalizeDNS(hostname)))
}

func byodResourceName(kind, value string) string {
	digest := sha256.Sum256([]byte(value))
	return fmt.Sprintf("byod-%s-%x", kind, digest[:6])
}

// warnf logs a Pulumi warning. BYOD TLS problems are reported this way rather
// than returned: the services are already deployed and reachable on their
// delegate-domain hostnames, so failing the deploy over a certificate would undo
// working infrastructure. This is the same warn-and-degrade rule AWS and Azure
// follow (see docs/byod-dns-zones.md).
func warnf(ctx *pulumi.Context, format string, args ...any) {
	_ = ctx.Log.Warn(fmt.Sprintf(format, args...), nil)
}
