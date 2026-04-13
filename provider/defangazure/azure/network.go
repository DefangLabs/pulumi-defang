package azure

import (
	"fmt"

	"github.com/pulumi/pulumi-azure-native-sdk/network/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// NetworkingResult holds shared VNet resources for a project.
type NetworkingResult struct {
	VNet                   *network.VirtualNetwork
	AppsSubnet             *network.Subnet
	PostgresSubnet         *network.Subnet
	PrivateEndpointsSubnet *network.Subnet // subnet for private endpoints (Redis, etc.)
}

// CreateNetworking creates a VNet with three subnets:
//   - apps subnet (10.0.0.0/23): used by Container Apps managed environment
//   - postgres subnet (10.0.2.0/24): delegated to Microsoft.DBforPostgreSQL/flexibleServers
//   - endpoints subnet (10.0.3.0/24): for private endpoints (Redis, etc.)
func CreateNetworking(
	ctx *pulumi.Context,
	name string,
	infra *SharedInfra,
	location string,
	opts ...pulumi.ResourceOption,
) (*NetworkingResult, error) {
	vnet, err := network.NewVirtualNetwork(ctx, name, &network.VirtualNetworkArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		Location:          pulumi.String(location),
		AddressSpace: &network.AddressSpaceArgs{
			AddressPrefixes: pulumi.StringArray{pulumi.String("10.0.0.0/16")},
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating VNet: %w", err)
	}

	appsSubnet, err := network.NewSubnet(ctx, name+"-apps", &network.SubnetArgs{
		ResourceGroupName:  infra.ResourceGroup.Name,
		VirtualNetworkName: vnet.Name,
		AddressPrefix:      pulumi.String("10.0.0.0/23"),
		Delegations: network.DelegationArray{
			network.DelegationArgs{
				Name:        pulumi.String("containerApps-delegation"),
				ServiceName: pulumi.String("Microsoft.App/environments"),
			},
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating apps subnet: %w", err)
	}

	pgSubnet, err := network.NewSubnet(ctx, name+"-postgres", &network.SubnetArgs{
		ResourceGroupName:  infra.ResourceGroup.Name,
		VirtualNetworkName: vnet.Name,
		AddressPrefix:      pulumi.String("10.0.2.0/24"),
		Delegations: network.DelegationArray{
			network.DelegationArgs{
				Name:        pulumi.String("postgres-delegation"),
				ServiceName: pulumi.String("Microsoft.DBforPostgreSQL/flexibleServers"),
			},
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating postgres subnet: %w", err)
	}

	peSubnet, err := network.NewSubnet(ctx, name+"-endpoints", &network.SubnetArgs{
		ResourceGroupName:                    infra.ResourceGroup.Name,
		VirtualNetworkName:                   vnet.Name,
		AddressPrefix:                        pulumi.String("10.0.3.0/24"),
		PrivateEndpointNetworkPolicies:       pulumi.String("Disabled"),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating private endpoints subnet: %w", err)
	}

	return &NetworkingResult{
		VNet:                   vnet,
		AppsSubnet:             appsSubnet,
		PostgresSubnet:         pgSubnet,
		PrivateEndpointsSubnet: peSubnet,
	}, nil
}
