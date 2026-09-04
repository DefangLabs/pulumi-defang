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

type dnsResource struct {
	typeToken string
	name      string
}

type dnsNameMocks struct {
	resources []dnsResource
}

func (m *dnsNameMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	switch args.TypeToken {
	case "azure-native:privatedns:PrivateZone", "azure-native:privatedns:VirtualNetworkLink":
		m.resources = append(m.resources, dnsResource{typeToken: args.TypeToken, name: args.Name})
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
	createNames := func(pgService, redisService string) []dnsResource {
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
		sort.Slice(mocks.resources, func(i, j int) bool {
			if mocks.resources[i].typeToken != mocks.resources[j].typeToken {
				return mocks.resources[i].typeToken < mocks.resources[j].typeToken
			}
			return mocks.resources[i].name < mocks.resources[j].name
		})
		return mocks.resources
	}

	const (
		zone = "azure-native:privatedns:PrivateZone"
		link = "azure-native:privatedns:VirtualNetworkLink"
		pg   = "postgres"
		rd   = "redis"
	)

	first := createNames("alpha-pg", "alpha-cache")
	require.Equal(t, []dnsResource{
		{zone, pg}, {zone, rd}, {link, pg}, {link, rd},
	}, first)

	// Same identities for a project whose services sort differently.
	require.Equal(t, first, createNames("zulu-pg", "zulu-cache"))

	// A Pulumi URN inherits its parent's TYPE but not its name, so two
	// resources of one type must not share a logical name even when their
	// parents differ -- that is a duplicate URN and the deploy bails. Pulumi's
	// mock runtime does not run the step generator, so nothing but this check
	// would catch it here.
	seen := map[dnsResource]bool{}
	for _, r := range first {
		require.False(t, seen[r], "duplicate logical name %q for type %s", r.name, r.typeToken)
		seen[r] = true
	}
}
