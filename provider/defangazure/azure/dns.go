package azure

import (
	"fmt"

	"github.com/pulumi/pulumi-azure-native-sdk/privatedns/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// DNSResult holds private DNS zones for a project.
type DNSResult struct {
	// PostgresZoneName is the private DNS zone used by PostgreSQL Flexible Servers
	// for automatic hostname registration (e.g. "projectname.private.postgres.database.azure.com").
	PostgresZoneName pulumi.StringOutput

	// PostgresZoneID is the resource ID of the postgres private DNS zone,
	// needed when creating a Flexible Server with VNet integration.
	PostgresZoneID pulumi.StringOutput

	// RedisPrivateZone is the private DNS zone for Redis Enterprise private endpoints
	// ("privatelink.redis.azure.net"). The PrivateDnsZoneGroup on the private endpoint
	// auto-registers A records here so cluster FQDNs resolve to private IPs within the VNet.
	RedisPrivateZone *privatedns.PrivateZone

	// RedisVNetLink links the Redis private zone to the project VNet. A registered
	// A record only resolves from inside the VNet once this link is effective, so
	// it's threaded into Redis readiness deps to keep apps from starting before
	// the hostname resolves (see issue #287).
	RedisVNetLink *privatedns.VirtualNetworkLink
}

// CreateDNSZones creates the private DNS zones linked to the project VNet:
//  1. A postgres zone ("<project>.private.postgres.database.azure.com") for Flexible Server integration.
//  2. The Redis Enterprise privatelink zone for private endpoint resolution.
//
// pgServiceName and redisServiceName only select which zones are needed; pass
// empty to skip that kind's zone entirely. They are deliberately NOT used as
// logical names: both zones are project-shared, the caller picks the first
// alphabetical service of each kind, and naming a shared resource after one of
// its users means adding an earlier-sorting service replaces the zone the rest
// of the project is already resolving through.
func CreateDNSZones(
	ctx *pulumi.Context,
	projectName, pgServiceName, redisServiceName string,
	infra *SharedInfra,
	networking *NetworkingResult,
	opts ...pulumi.ResourceOption,
) (*DNSResult, error) {
	result := &DNSResult{}

	if pgServiceName != "" {
		// Postgres private DNS zone — name must end in ".private.postgres.database.azure.com".
		pgZoneName := projectName + ".private.postgres.database.azure.com"
		pgZone, err := privatedns.NewPrivateZone(ctx, "postgres", &privatedns.PrivateZoneArgs{
			ResourceGroupName: infra.ResourceGroup.Name,
			Location:          pulumi.String("global"),
			PrivateZoneName:   pulumi.String(pgZoneName),
		}, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating postgres private DNS zone: %w", err)
		}

		// Named for its zone, not "link": a Pulumi URN inherits its parent's
		// TYPE but not its name, so two links called "link" under two
		// PrivateZone parents would produce one URN and fail the deploy.
		_, err = privatedns.NewVirtualNetworkLink(ctx, "postgres", &privatedns.VirtualNetworkLinkArgs{
			ResourceGroupName:   infra.ResourceGroup.Name,
			PrivateZoneName:     pgZone.Name,
			Location:            pulumi.String("global"),
			RegistrationEnabled: pulumi.Bool(false),
			VirtualNetwork:      &privatedns.SubResourceArgs{Id: networking.VNet.ID().ToStringOutput()},
		}, append(opts, pulumi.Parent(pgZone))...)
		if err != nil {
			return nil, fmt.Errorf("creating postgres DNS VNet link: %w", err)
		}

		result.PostgresZoneName = pgZone.Name
		result.PostgresZoneID = pgZone.ID().ToStringOutput()
	}

	if redisServiceName != "" {
		// Redis Enterprise private DNS zone — required for private endpoint DNS resolution.
		// Azure resolves <cluster>.westus3.redis.azure.net → <cluster>.privatelink.redis.azure.net
		// and this zone maps that to the private endpoint IP.
		redisZone, err := privatedns.NewPrivateZone(ctx, "redis", &privatedns.PrivateZoneArgs{
			ResourceGroupName: infra.ResourceGroup.Name,
			Location:          pulumi.String("global"),
			PrivateZoneName:   pulumi.String("privatelink.redis.azure.net"),
		}, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating Redis private DNS zone: %w", err)
		}

		redisLink, err := privatedns.NewVirtualNetworkLink(ctx, "redis", &privatedns.VirtualNetworkLinkArgs{
			ResourceGroupName:   infra.ResourceGroup.Name,
			PrivateZoneName:     redisZone.Name,
			Location:            pulumi.String("global"),
			RegistrationEnabled: pulumi.Bool(false),
			VirtualNetwork:      &privatedns.SubResourceArgs{Id: networking.VNet.ID().ToStringOutput()},
		}, append(opts, pulumi.Parent(redisZone))...)
		if err != nil {
			return nil, fmt.Errorf("creating Redis DNS VNet link: %w", err)
		}

		result.RedisPrivateZone = redisZone
		result.RedisVNetLink = redisLink
	}

	return result, nil
}
