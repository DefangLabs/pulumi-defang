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

	// The legacy CD retains this connection, citing
	// hashicorp/terraform-provider-google#16275: the provider cannot delete a
	// service networking connection at all. That is a 5.x regression
	// (removePeering -> deleteConnection) which fails even once the dependent
	// Cloud SQL instances are gone; it was closed as a duplicate of #16944,
	// whose resolution added an *abandon* deletion_policy rather than a working
	// delete.
	//
	// Deleting it normally here is therefore NOT yet justified: the CD's
	// scheduled retry cannot succeed either, and would abandon the VPC at its
	// deadline. Retaining it does block the VPC delete, so neither option is
	// good — see issue 183. If this stays unretained, DeletionPolicy: ABANDON
	// plus an out-of-band removePeering is the honest expression of it.
	serviceConn, err := servicenetworking.NewConnection(ctx, projectName+"-svc-conn",
		&servicenetworking.ConnectionArgs{
			Network:               vpcId,
			Service:               pulumi.String("servicenetworking.googleapis.com"),
			ReservedPeeringRanges: pulumi.StringArray{privateIpAlloc.Name},
		},
		opts...,
	)
	if err != nil {
		return nil, err
	}
	return serviceConn, nil
}
