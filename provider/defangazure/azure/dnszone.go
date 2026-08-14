package azure

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
)

// FindZones maps each hostname to the ARM resource ID of the public DNS zone in
// subscriptionID whose name is its longest DNS suffix, omitting hostnames no zone
// matches. The subscription's zones are listed once for the whole batch, since a
// project's hostnames are resolved together and each ARM listing is a round trip.
//
// This is Azure's half of BYOD ("bring your own domain") zone discovery: when a
// service asks for a custom hostname and a zone that can host it already exists,
// Defang manages the records there and issues an Azure-managed cert, instead of
// falling back to the ACME / `defang cert generate` path. AWS's half is
// GetHostedZoneForHost + ErrZoneNotFound in provider/defangaws/aws/route53.go;
// both answer the same question per hostname, and "no zone" is a normal answer on
// both.
//
// Unlike AWS's, this can't be a Pulumi invoke: azure-native exposes getZone
// (resource group + zone name required) but no subscription-wide listing, and the
// zone's resource group is exactly what we don't know yet. So it's an imperative
// ARM call, like the other out-of-band lookups in this package
// (readLiveCustomDomains, ModelSelector).
//
// The search applies no ownership or tag filter (per the BYOD design): whichever
// existing zone is the closest parent of the hostname wins. Only subscriptionID is
// searched — Azure has no cross-subscription equivalent of Route53's AssumeRole,
// so there is no analogue of x-defang-dns-role here.
func FindZones(ctx context.Context, subscriptionID string, hostnames []string) (map[string]string, error) {
	if len(hostnames) == 0 {
		return map[string]string{}, nil
	}
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("finding DNS zones: building credential: %w", err)
	}
	client, err := armdns.NewZonesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("finding DNS zones: creating zones client: %w", err)
	}

	var zones []*armdns.Zone
	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing DNS zones: %w", err)
		}
		zones = append(zones, page.Value...)
	}

	found := map[string]string{}
	for _, hostname := range hostnames {
		if _, id := bestZoneMatch(hostname, zones); id != "" {
			found[strings.ToLower(strings.TrimSuffix(hostname, "."))] = id
		}
	}
	return found, nil
}

// bestZoneMatch picks the zone whose name is the longest DNS suffix of domain,
// returning its name and ARM resource ID (both "" when nothing matches). A zone
// matches domain itself or any parent of it, so "api.example.com" matches a zone
// named "example.com" but "example.com.evil.com" does not.
func bestZoneMatch(domain string, zones []*armdns.Zone) (string, string) {
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))
	var bestName, bestID string
	for _, z := range zones {
		if z == nil || z.Name == nil || z.ID == nil {
			continue
		}
		name := strings.ToLower(strings.TrimSuffix(*z.Name, "."))
		if domain != name && !strings.HasSuffix(domain, "."+name) {
			continue
		}
		if len(name) > len(bestName) {
			bestName, bestID = name, *z.ID
		}
	}
	return bestName, bestID
}
