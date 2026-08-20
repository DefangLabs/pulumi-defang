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

	// Deleted normally, as the legacy CD did. The connection delete fails while
	// a producer service still uses it (upstream
	// hashicorp/terraform-provider-google#19908), which here is a race against
	// the Cloud SQL instance delete rather than a permanent state. Retaining it
	// would instead block the VPC delete for ever, so the CD schedules a retry
	// on that failure — see cd/cleanup_gcp.go.
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
