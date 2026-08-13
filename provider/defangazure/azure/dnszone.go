package azure

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
)

// FindZone returns the ARM resource ID of the public DNS zone in subscriptionID
// whose name is the longest DNS suffix of domain, or "" if no zone matches.
//
// This is Azure's half of BYOD ("bring your own domain") zone discovery: when a
// service sets a custom domainname and a zone that can host it already exists,
// Defang manages the records there and issues an Azure-managed cert, instead of
// falling back to the ACME / `defang cert generate` path. AWS's half is
// GetHostedZoneForHost + IsZoneNotFound in provider/defangaws/aws/route53.go;
// both answer the same question, and "no zone" is a normal answer on both.
//
// Unlike AWS's, this can't be a Pulumi invoke: azure-native exposes getZone
// (resource group + zone name required) but no subscription-wide listing, and
// the zone's resource group is exactly what we don't know yet. So it's an
// imperative ARM call, like the other out-of-band lookups in this package
// (readLiveCustomDomains, ModelSelector).
//
// The search applies no ownership or tag filter (per the BYOD design): whichever
// existing zone is the closest parent of domain wins. Only subscriptionID is
// searched — Azure has no cross-subscription equivalent of Route53's AssumeRole,
// so there is no analogue of x-defang-dns-role here.
func FindZone(ctx context.Context, subscriptionID, domain string) (string, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return "", fmt.Errorf("finding DNS zone for %q: building credential: %w", domain, err)
	}
	client, err := armdns.NewZonesClient(subscriptionID, cred, nil)
	if err != nil {
		return "", fmt.Errorf("finding DNS zone for %q: creating zones client: %w", domain, err)
	}

	var zones []*armdns.Zone
	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return "", fmt.Errorf("listing DNS zones: %w", err)
		}
		zones = append(zones, page.Value...)
	}

	_, id := bestZoneMatch(domain, zones)
	return id, nil
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
