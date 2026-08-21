package gcp

import (
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/dns"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// FindZones maps each hostname to the name of the public Cloud DNS managed zone
// in gcpProject whose DNS name is its longest suffix, omitting hostnames no zone
// matches. The project's zones are listed once for the whole batch, since a
// project's hostnames are resolved together and the listing is a round trip.
//
// This is GCP's half of BYOD ("bring your own domain") zone discovery: when a
// service asks for a custom hostname and a zone that can host it already exists,
// Defang manages the records there and issues a DNS-authorized managed cert
// instead of a load-balancer-authorized one. AWS's half is GetHostedZoneForHost
// in provider/defangaws/aws/route53.go and Azure's is FindZones in
// provider/defangazure/azure/dnszone.go; all three answer the same question per
// hostname, and "no zone" is a normal answer on all three.
//
// Unlike Azure's, this is a plain Pulumi invoke: gcp:dns/getManagedZones lists
// every managed zone in a project, which is exactly the question being asked, so
// there is no need for an out-of-band client like Azure's ARM call.
//
// Only public zones are considered. A private zone is only resolvable from
// inside its authorized VPCs, so neither a public client nor Certificate
// Manager's DNS-01 validator could see a record placed in one.
//
// The search applies no ownership or tag filter (per the BYOD design): whichever
// existing zone is the closest parent of the hostname wins. Only gcpProject is
// searched — the deploy identity's own project — matching Azure's
// deploy-subscription-only scope. GCP has no analogue of Route 53's
// x-defang-dns-role cross-account hop.
func FindZones(
	ctx *pulumi.Context, gcpProject string, hostnames []string, opts ...pulumi.InvokeOption,
) (map[string]string, error) {
	if len(hostnames) == 0 {
		return map[string]string{}, nil
	}
	args := &dns.GetManagedZonesArgs{}
	if gcpProject != "" {
		args.Project = &gcpProject
	}
	result, err := dns.GetManagedZones(ctx, args, opts...)
	if err != nil {
		return nil, fmt.Errorf("listing Cloud DNS managed zones: %w", err)
	}

	found := map[string]string{}
	for _, hostname := range hostnames {
		if zone := bestZoneMatch(hostname, result.ManagedZones); zone != "" {
			found[common.NormalizeDNS(hostname)] = zone
		}
	}
	return found, nil
}

// bestZoneMatch picks the public zone whose DNS name is the longest suffix of
// hostname and returns that zone's name ("" when nothing matches). A zone
// matches hostname itself or any parent of it, so "api.example.com" matches a
// zone for "example.com" but "example.com.evil.com" does not.
//
// A wildcard hostname matches on the name it stands for: "*.example.com" is
// hosted by the "example.com" zone, and by a "foo.example.com" zone when the
// hostname is "*.foo.example.com".
func bestZoneMatch(hostname string, zones []dns.GetManagedZonesManagedZone) string {
	hostname = common.NormalizeDNS(strings.TrimPrefix(hostname, common.WildcardPrefix))
	if hostname == "" {
		return ""
	}
	var bestZone, bestName string
	for _, z := range zones {
		// Visibility is "public" or "private"; the API leaves it empty for public
		// zones created before the field existed, so treat empty as public.
		if z.Visibility != "" && z.Visibility != "public" {
			continue
		}
		if z.Name == nil || *z.Name == "" {
			continue
		}
		name := common.NormalizeDNS(z.DnsName)
		if name == "" {
			continue
		}
		if hostname != name && !strings.HasSuffix(hostname, "."+name) {
			continue
		}
		if len(name) > len(bestName) {
			bestName, bestZone = name, *z.Name
		}
	}
	return bestZone
}
