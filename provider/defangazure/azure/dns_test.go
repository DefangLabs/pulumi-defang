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
	aliases   []dnsResource
}

func (m *dnsNameMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	switch args.TypeToken {
	case "azure-native:privatedns:PrivateZone", "azure-native:privatedns:VirtualNetworkLink":
		m.resources = append(m.resources, dnsResource{typeToken: args.TypeToken, name: args.Name})
		for _, alias := range args.RegisterRPC.GetAliases() {
			// The azure-native SDK attaches an alias per historical API version
			// to every resource; those carry a type and no name. Only the ones
			// this package adds name a previous logical name.
			if spec := alias.GetSpec(); spec != nil && spec.GetName() != "" {
				m.aliases = append(m.aliases, dnsResource{typeToken: args.TypeToken, name: spec.GetName()})
			}
		}
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

// Azure runs in production, so the rename above must not drop the URNs the old
// service-derived names created: each zone and link claims its old name as an
// alias. Without them an `up` on a live project deletes the zone every server
// and private endpoint resolves through.
func TestCreateDNSZonesKeepTheOldServiceNamesAsAliases(t *testing.T) {
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
		_, err = CreateDNSZones(ctx, "project", "db", "cache",
			&SharedInfra{ResourceGroup: rg}, &NetworkingResult{VNet: vnet})
		return err
	}, pulumi.WithMocks("project", "stack", mocks))
	require.NoError(t, err)

	const zone = "azure-native:privatedns:PrivateZone"
	const link = "azure-native:privatedns:VirtualNetworkLink"
	require.ElementsMatch(t, []dnsResource{
		{zone, "db"}, {link, "db"}, {zone, "cache"}, {link, "cache"},
	}, mocks.aliases)
}

// A project whose service is already called "postgres" needs no alias: the old
// name and the new one are the same URN.
func TestCreateDNSZonesDeclareNoSelfAlias(t *testing.T) {
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
		_, err = CreateDNSZones(ctx, "project", postgresName, redisName,
			&SharedInfra{ResourceGroup: rg}, &NetworkingResult{VNet: vnet})
		return err
	}, pulumi.WithMocks("project", "stack", mocks))
	require.NoError(t, err)

	require.Empty(t, mocks.aliases)
}
