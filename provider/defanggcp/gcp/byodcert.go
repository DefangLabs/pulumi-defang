package gcp

import (
	"fmt"
	"strconv"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/certificatemanager"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/dns"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

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
	seen := map[string]bool{}
	for _, entry := range entries {
		for i, hostname := range common.ByodHostnames(entry.Config) {
			hostname = common.NormalizeDNS(hostname)
			if hostname == "" {
				continue
			}
			if seen[hostname] {
				// Two names that resolve to the same certificate map entry would make
				// the GCP API reject the map, so keep the first and say which one lost.
				warnf(ctx, "BYOD TLS: %q is requested more than once in this project; "+
					"certifying it for the first service that asked and ignoring the repeat", hostname)
				continue
			}
			seen[hostname] = true

			// Names are derived from the service and the hostname's position in its
			// list, not from the hostname itself: a Certificate Manager resource name
			// is capped at 63 characters and Pulumi prefixes ours with
			// "<project>-<stack>-" and suffixes a hash, which a long hostname would
			// blow past.
			name := entry.Name + "-byod-cert"
			if i > 0 {
				name += "-" + strconv.Itoa(i)
			}

			cert, err := createByodCert(ctx, name, hostname, infra, opts...)
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

	// A DNS authorization is issued for the name the certificate covers, which for
	// a wildcard is the parent it stands for: an authorization for "example.com"
	// covers both "example.com" and "*.example.com".
	authzDomain := common.NormalizeDNS(trimWildcard(hostname))
	authz, err := certificatemanager.NewDnsAuthorization(ctx, name+"-authz",
		&certificatemanager.DnsAuthorizationArgs{
			Description: pulumi.StringPtr("DNS authorization for " + hostname),
			Domain:      pulumi.String(authzDomain),
		}, opts...)
	if err != nil {
		return nil, err
	}

	// A FIXED_RECORD authorization (the default) publishes exactly one challenge
	// record, so read it off the output by index rather than creating resources
	// inside an ApplyT the way createWildcardCert still has to.
	record := authz.DnsResourceRecords.Index(pulumi.Int(0))
	if _, err := dns.NewRecordSet(ctx, name+"-authz-record", &dns.RecordSetArgs{
		ManagedZone: pulumi.String(zone),
		Name:        record.Name().Elem(),
		Type:        record.Type().Elem(),
		Ttl:         pulumi.Int(60),
		Rrdatas:     pulumi.StringArray{record.Data().Elem()},
	}, opts...); err != nil {
		return nil, err
	}

	return certificatemanager.NewCertificate(ctx, name, &certificatemanager.CertificateArgs{
		Description: pulumi.StringPtr("DNS authorized certificate for " + hostname),
		Managed: &certificatemanager.CertificateManagedArgs{
			Domains:           pulumi.StringArray{pulumi.String(hostname)},
			DnsAuthorizations: pulumi.StringArray{authz.ID()},
		},
	}, opts...)
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

// warnf logs a Pulumi warning. BYOD TLS problems are reported this way rather
// than returned: the services are already deployed and reachable on their
// delegate-domain hostnames, so failing the deploy over a certificate would undo
// working infrastructure. This is the same warn-and-degrade rule AWS and Azure
// follow (see docs/byod-dns-zones.md).
func warnf(ctx *pulumi.Context, format string, args ...any) {
	_ = ctx.Log.Warn(fmt.Sprintf(format, args...), nil)
}
