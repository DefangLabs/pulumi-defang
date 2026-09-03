package gcp

import (
	"fmt"
	"slices"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/dns"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// byodZoneAuthorizationMarker is an explicit opt-in from a Cloud DNS zone
// owner allowing Defang deployments in the project to create BYOD records in
// that zone. It must appear as a whitespace-delimited word in the zone's
// description.
//
// Listing zones proves only that the deploy identity can see them. In a shared
// project, that is not evidence that the author of a particular Compose project
// owns every visible DNS namespace. Requiring a zone-owner-controlled marker
// keeps an arbitrary domainname from turning project-wide DNS permissions into
// authority to overwrite another application's records.
const byodZoneAuthorizationMarker = "defang.dev/byod-dns=authorized"

// FindZones maps each hostname to the name of the authorized public Cloud DNS
// managed zone in gcpProject whose DNS name is its longest suffix, omitting
// hostnames no trusted zone matches. The project's zones are listed once for the
// whole batch, since a project's hostnames are resolved together and the
// listing is a round trip.
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
// A zone is eligible only when its description contains
// byodZoneAuthorizationMarker as a whitespace-delimited word. This is the
// explicit authorization boundary for shared GCP projects: project-wide list
// and record permissions alone do not prove that one deployment owns another
// team's zone. Only gcpProject is searched — the deploy identity's own project.
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
		zone, err := bestZoneMatch(hostname, result.ManagedZones)
		if err != nil {
			return nil, err
		}
		if zone != "" {
			found[common.NormalizeDNS(hostname)] = zone
		}
	}
	return found, nil
}

// bestZoneMatch picks the authorized public zone whose DNS name is the longest
// suffix of hostname and returns that zone's name ("" when nothing matches). It
// returns an error when distinct authorized zones share that equally best DNS
// suffix because their names cannot reveal which one is publicly delegated. A
// zone matches hostname itself or any parent of it, so "api.example.com"
// matches a zone for "example.com" but "example.com.evil.com" does not.
//
// A wildcard hostname matches on the name it stands for: "*.example.com" is
// hosted by the "example.com" zone, and by a "foo.example.com" zone when the
// hostname is "*.foo.example.com".
func bestZoneMatch(hostname string, zones []dns.GetManagedZonesManagedZone) (string, error) {
	hostname = common.NormalizeDNS(strings.TrimPrefix(hostname, common.WildcardPrefix))
	if hostname == "" {
		return "", nil
	}
	var bestName string
	bestZones := map[string]struct{}{}
	for _, z := range zones {
		// Visibility is "public" or "private"; the API leaves it empty for public
		// zones created before the field existed, so treat empty as public.
		if z.Visibility != "" && z.Visibility != "public" {
			continue
		}
		if z.Name == nil || *z.Name == "" {
			continue
		}
		if !zoneAllowsByodRecords(z.Description) {
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
			bestName = name
			bestZones = map[string]struct{}{*z.Name: {}}
		} else if len(name) == len(bestName) {
			bestZones[*z.Name] = struct{}{}
		}
	}
	if len(bestZones) == 0 {
		return "", nil
	}
	zoneNames := make([]string, 0, len(bestZones))
	for zoneName := range bestZones {
		zoneNames = append(zoneNames, zoneName)
	}
	slices.Sort(zoneNames)
	if len(zoneNames) > 1 {
		return "", fmt.Errorf(
			"multiple authorized public Cloud DNS managed zones match hostname %q at DNS suffix %q: %s; keep %q in the description of only the publicly delegated zone",
			hostname, bestName, strings.Join(zoneNames, ", "), byodZoneAuthorizationMarker,
		)
	}
	return zoneNames[0], nil
}

func zoneAllowsByodRecords(description string) bool {
	for _, word := range strings.Fields(description) {
		if word == byodZoneAuthorizationMarker {
			return true
		}
	}
	return false
}
