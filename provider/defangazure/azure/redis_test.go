package azure

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/network/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/privatedns/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// vnetRedisInfra builds the minimal SharedInfra needed to exercise the
// VNet/private-endpoint path of CreateRedisEnterprise: a resource group, a
// private-endpoints subnet, and the Redis private DNS zone + VNet link.
func vnetRedisInfra(t *testing.T, ctx *pulumi.Context) *SharedInfra {
	t.Helper()

	rg, err := resources.NewResourceGroup(ctx, "rg", &resources.ResourceGroupArgs{
		ResourceGroupName: pulumi.String("rg"),
	})
	require.NoError(t, err)

	vnet, err := network.NewVirtualNetwork(ctx, "vnet", &network.VirtualNetworkArgs{
		ResourceGroupName: rg.Name,
	})
	require.NoError(t, err)

	subnet, err := network.NewSubnet(ctx, "pe", &network.SubnetArgs{
		ResourceGroupName:  rg.Name,
		VirtualNetworkName: vnet.Name,
	})
	require.NoError(t, err)

	zone, err := privatedns.NewPrivateZone(ctx, "redis", &privatedns.PrivateZoneArgs{
		ResourceGroupName: rg.Name,
		PrivateZoneName:   pulumi.String("privatelink.redis.azure.net"),
	})
	require.NoError(t, err)

	link, err := privatedns.NewVirtualNetworkLink(ctx, "redis", &privatedns.VirtualNetworkLinkArgs{
		ResourceGroupName: rg.Name,
		PrivateZoneName:   zone.Name,
		VirtualNetwork:    &privatedns.SubResourceArgs{Id: vnet.ID().ToStringOutput()},
	})
	require.NoError(t, err)

	return &SharedInfra{
		ResourceGroup: rg,
		Networking:    &NetworkingResult{VNet: vnet, PrivateEndpointsSubnet: subnet},
		DNS:           &DNSResult{RedisPrivateZone: zone, RedisVNetLink: link},
	}
}

// TestCreateRedisEnterpriseReadinessVNet verifies that the VNet path populates
// Readiness with the private-DNS plumbing (zone group + VNet link). Downstream
// apps fold these into the outputs they consume so Pulumi won't start a Container
// App before the cluster's A record is published and resolvable — the fix for the
// fresh-deploy DNS race in issue #287.
func TestCreateRedisEnterpriseReadinessVNet(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		infra := vnetRedisInfra(t, ctx)
		svc := compose.ServiceConfig{Redis: &compose.RedisConfig{}}

		result, err := CreateRedisEnterprise(ctx, "cache", svc, infra)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Zone group (publishes the A record) + VNet link (makes it resolvable).
		assert.Len(t, result.Readiness, 2,
			"VNet Redis must expose the DNS zone group and VNet link as readiness deps")
		for i, r := range result.Readiness {
			assert.NotNil(t, r, "readiness resource %d must not be nil", i)
		}
		return nil
	}, pulumi.WithMocks("proj", "stack", azureNoopMocks{}))
	require.NoError(t, err)
}

// TestCreateRedisEnterpriseReadinessNoVNet verifies the standalone/no-VNet path
// (Encrypted protocol, public rediss:// endpoint) exposes no readiness deps —
// there is no private DNS to wait on, so apps connect directly to the cluster.
func TestCreateRedisEnterpriseReadinessNoVNet(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		rg, err := resources.NewResourceGroup(ctx, "rg", &resources.ResourceGroupArgs{
			ResourceGroupName: pulumi.String("rg"),
		})
		require.NoError(t, err)

		infra := &SharedInfra{ResourceGroup: rg} // no Networking, no DNS
		svc := compose.ServiceConfig{Redis: &compose.RedisConfig{}}

		result, err := CreateRedisEnterprise(ctx, "cache", svc, infra)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Empty(t, result.Readiness,
			"no-VNet Redis has no private DNS to wait on, so Readiness must be empty")
		return nil
	}, pulumi.WithMocks("proj", "stack", azureNoopMocks{}))
	require.NoError(t, err)
}
