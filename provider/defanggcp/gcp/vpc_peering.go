package gcp

import (
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/servicenetworking"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// needsVpcPeering reports whether any service in the map requires VPC peering.
func needsVpcPeering(services map[string]compose.ServiceConfig) bool {
	for _, svc := range services {
		if svc.Postgres != nil || svc.Redis != nil {
			return true
		}
	}
	return false
}

// createVPCPeeringInfra allocates a private IP range and creates a service networking
// connection for Cloud SQL private IP access.
func createVPCPeeringInfra(
	ctx *pulumi.Context,
	projectName string,
	vpcId pulumi.StringOutput,
	opts ...pulumi.ResourceOption,
) (*servicenetworking.Connection, error) {
	privateIpAlloc, err := compute.NewGlobalAddress(ctx, projectName+"-peering-ip", &compute.GlobalAddressArgs{
		Purpose:      pulumi.String("VPC_PEERING"),
		AddressType:  pulumi.String("INTERNAL"),
		PrefixLength: pulumi.Int(16),
		Network:      vpcId,
	}, opts...)
	if err != nil {
		return nil, err
	}

	serviceConn, err := servicenetworking.NewConnection(ctx, projectName+"-svc-conn",
		&servicenetworking.ConnectionArgs{
			Network:               vpcId,
			Service:               pulumi.String("servicenetworking.googleapis.com"),
			ReservedPeeringRanges: pulumi.StringArray{privateIpAlloc.Name},
		},
		append(opts, pulumi.RetainOnDelete(true))...,
	)
	if err != nil {
		return nil, err
	}
	return serviceConn, nil
}
