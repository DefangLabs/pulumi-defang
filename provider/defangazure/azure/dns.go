package azure

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-azure-native-sdk/privatedns/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	postgresPrivateDNSName  = "postgres-private-dns"
	postgresDNSVNetLinkName = "postgres-vnet-link"
	redisPrivateDNSName     = "redis-private-dns"
	redisDNSVNetLinkName    = "redis-vnet-link"
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
// pgServiceName and redisServiceName select which zone kinds are needed. They
// are also retained as aliases for the old service-derived logical names.
// Pass empty to skip that kind's zone entirely.
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
		pgZone, err := privatedns.NewPrivateZone(ctx, postgresPrivateDNSName, &privatedns.PrivateZoneArgs{
			ResourceGroupName: infra.ResourceGroup.Name,
			Location:          pulumi.String("global"),
			PrivateZoneName:   pulumi.String(pgZoneName),
		}, common.MergeOptions(opts, pulumi.Aliases([]pulumi.Alias{{Name: pulumi.String(pgServiceName)}}))...)
		if err != nil {
			return nil, fmt.Errorf("creating postgres private DNS zone: %w", err)
		}

		_, err = privatedns.NewVirtualNetworkLink(ctx, postgresDNSVNetLinkName, &privatedns.VirtualNetworkLinkArgs{
			ResourceGroupName:   infra.ResourceGroup.Name,
			PrivateZoneName:     pgZone.Name,
			Location:            pulumi.String("global"),
			RegistrationEnabled: pulumi.Bool(false),
			VirtualNetwork:      &privatedns.SubResourceArgs{Id: networking.VNet.ID().ToStringOutput()},
		}, common.MergeOptions(opts,
			pulumi.Parent(pgZone),
			pulumi.Aliases([]pulumi.Alias{{Name: pulumi.String(pgServiceName)}}),
		)...)
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
		redisZone, err := privatedns.NewPrivateZone(ctx, redisPrivateDNSName, &privatedns.PrivateZoneArgs{
			ResourceGroupName: infra.ResourceGroup.Name,
			Location:          pulumi.String("global"),
			PrivateZoneName:   pulumi.String("privatelink.redis.azure.net"),
		}, common.MergeOptions(opts, pulumi.Aliases([]pulumi.Alias{{Name: pulumi.String(redisServiceName)}}))...)
		if err != nil {
			return nil, fmt.Errorf("creating Redis private DNS zone: %w", err)
		}

		redisLink, err := privatedns.NewVirtualNetworkLink(ctx, redisDNSVNetLinkName, &privatedns.VirtualNetworkLinkArgs{
			ResourceGroupName:   infra.ResourceGroup.Name,
			PrivateZoneName:     redisZone.Name,
			Location:            pulumi.String("global"),
			RegistrationEnabled: pulumi.Bool(false),
			VirtualNetwork:      &privatedns.SubResourceArgs{Id: networking.VNet.ID().ToStringOutput()},
		}, common.MergeOptions(opts,
			pulumi.Parent(redisZone),
			pulumi.Aliases([]pulumi.Alias{{Name: pulumi.String(redisServiceName)}}),
		)...)
		if err != nil {
			return nil, fmt.Errorf("creating Redis DNS VNet link: %w", err)
		}

		result.RedisPrivateZone = redisZone
		result.RedisVNetLink = redisLink
	}

	return result, nil
}
