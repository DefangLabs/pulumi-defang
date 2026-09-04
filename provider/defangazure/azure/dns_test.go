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
	switch args.TypeToken {
	case "azure-native:privatedns:PrivateZone", "azure-native:privatedns:VirtualNetworkLink":
		m.names = append(m.names, args.Name)
	}
	return args.Name + "_id", args.Inputs, nil
}

func (m *dnsNameMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

// The zones are project-shared, so their identity must not move when the
// project gains a service that sorts before the one the caller happened to
// pick: that would replace the zone the rest of the project resolves through.
func TestCreateDNSZonesNamesAreIndependentOfTheService(t *testing.T) {
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

	// Two zones and one link each; the links share a name because each is
	// parented to its own zone.
	require.Equal(t, []string{"link", "link", "postgres", "redis"}, createNames("alpha-pg", "alpha-cache"))
	require.Equal(t, createNames("alpha-pg", "alpha-cache"), createNames("zulu-pg", "zulu-cache"))
}
