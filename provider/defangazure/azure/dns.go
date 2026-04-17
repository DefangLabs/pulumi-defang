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
}

// CreateDNSZones creates the private DNS zones linked to the project VNet:
//  1. A postgres zone ("<name>.private.postgres.database.azure.com") for Flexible Server integration.
//  2. The Redis Enterprise privatelink zone for private endpoint resolution.
func CreateDNSZones(
	ctx *pulumi.Context,
	name string,
	infra *SharedInfra,
	networking *NetworkingResult,
	opts ...pulumi.ResourceOption,
) (*DNSResult, error) {
	// Postgres private DNS zone — name must end in ".private.postgres.database.azure.com".
	pgZoneName := name + ".private.postgres.database.azure.com"
	pgZone, err := privatedns.NewPrivateZone(ctx, name+"-pg-dns", &privatedns.PrivateZoneArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		Location:          pulumi.String("global"),
		PrivateZoneName:   pulumi.String(pgZoneName),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating postgres private DNS zone: %w", err)
	}

	_, err = privatedns.NewVirtualNetworkLink(ctx, name+"-pg-dns-link", &privatedns.VirtualNetworkLinkArgs{
		ResourceGroupName:   infra.ResourceGroup.Name,
		PrivateZoneName:     pgZone.Name,
		Location:            pulumi.String("global"),
		RegistrationEnabled: pulumi.Bool(false),
		VirtualNetwork:      &privatedns.SubResourceArgs{Id: networking.VNet.ID().ToStringOutput()},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating postgres DNS VNet link: %w", err)
	}

	// Redis Enterprise private DNS zone — required for private endpoint DNS resolution.
	// Azure resolves <cluster>.westus3.redis.azure.net → <cluster>.privatelink.redis.azure.net
	// and this zone maps that to the private endpoint IP.
	redisZone, err := privatedns.NewPrivateZone(ctx, name+"-redis-dns", &privatedns.PrivateZoneArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		Location:          pulumi.String("global"),
		PrivateZoneName:   pulumi.String("privatelink.redis.azure.net"),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Redis private DNS zone: %w", err)
	}

	_, err = privatedns.NewVirtualNetworkLink(ctx, name+"-redis-dns-link", &privatedns.VirtualNetworkLinkArgs{
		ResourceGroupName:   infra.ResourceGroup.Name,
		PrivateZoneName:     redisZone.Name,
		Location:            pulumi.String("global"),
		RegistrationEnabled: pulumi.Bool(false),
		VirtualNetwork:      &privatedns.SubResourceArgs{Id: networking.VNet.ID().ToStringOutput()},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Redis DNS VNet link: %w", err)
	}

	return &DNSResult{
		PostgresZoneName: pgZone.Name,
		PostgresZoneID:   pgZone.ID().ToStringOutput(),
		RedisPrivateZone: redisZone,
	}, nil
}
