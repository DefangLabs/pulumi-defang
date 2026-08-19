package azure

import (
	"sort"
	"testing"

	"github.com/pulumi/pulumi-azure-native-sdk/network/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/require"
)

type dnsNameMocks struct {
	names []string
}

func (m *dnsNameMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if args.TypeToken == "azure-native:privatedns:PrivateZone" ||
		args.TypeToken == "azure-native:privatedns:VirtualNetworkLink" {
		m.names = append(m.names, args.Name)
	}
	return args.Name + "_id", args.Inputs, nil
}

func (m *dnsNameMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func TestCreateDNSZonesUsesStableProjectSharedLogicalNames(t *testing.T) {
	createNames := func(pgService, redisService string) []string {
		mocks := &dnsNameMocks{}
		err := pulumi.RunErr(func(ctx *pulumi.Context) error {
			rg, err := resources.NewResourceGroup(ctx, "project", &resources.ResourceGroupArgs{})
			if err != nil {
				return err
			}
			vnet, err := network.NewVirtualNetwork(ctx, "project", &network.VirtualNetworkArgs{
				ResourceGroupName: rg.Name,
			})
			if err != nil {
				return err
			}
			_, err = CreateDNSZones(ctx, "project", pgService, redisService,
				&SharedInfra{ResourceGroup: rg}, &NetworkingResult{VNet: vnet})
			return err
		}, pulumi.WithMocks("project", "stack", mocks))
		require.NoError(t, err)
		sort.Strings(mocks.names)
		return mocks.names
	}

	first := createNames("alpha-postgres", "alpha-redis")
	second := createNames("zulu-postgres", "zulu-redis")
	want := []string{
		postgresPrivateDNSName, postgresDNSVNetLinkName,
		redisPrivateDNSName, redisDNSVNetLinkName,
	}
	sort.Strings(want)
	require.Equal(t, want, first)
	require.Equal(t, first, second)
}
