package gcp

import (
	"github.com/DefangLabs/pulumi-defang/provider/common"
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
	// Logical names below deliberately omit projectName: Pulumi's default resource
	// ID already prefixes it with <pulumi-project>-<stack>, which includes
	// projectName, so repeating it here risked exceeding GCP's 63-char resource
	// ID limit.
	privateIpAlloc, err := compute.NewGlobalAddress(ctx, "peering-ip", &compute.GlobalAddressArgs{
		Purpose:      pulumi.String("VPC_PEERING"),
		AddressType:  pulumi.String("INTERNAL"),
		PrefixLength: pulumi.Int(16),
		Network:      vpcId,
	}, common.MergeOptions(opts,
		legacyAlias(legacyResourceName(ctx, projectName, "vpc-peering-ip")))...)
	if err != nil {
		return nil, err
	}

	// DeletionPolicy ABANDON, and deliberately NOT RetainOnDelete. The provider's delete calls
	// servicenetworking deleteConnection, which cannot be relied on: GCP requires every service
	// instance to be gone first and producer-side cleanup lags the instance delete by up to 4
	// days for Cloud SQL, and hashicorp/terraform-provider-google#18834 (still open) records that
	// the call fails even after that, because Google's own console removes the peering through
	// the Compute API instead. ABANDON skips the doomed call; the CLI's cleanup tool
	// (DefangLabs/defang#2157) then calls compute.networks.removePeering, which releases the
	// reserved range so the subnet and the VPC can go. RetainOnDelete would instead drop the
	// connection from the Pulumi state while leaving the whole VPC standing (issue #183).
	//
	// The field is Optional+Computed and not ForceNew, so it updates in place on an existing
	// stack and never replaces the peering.
	serviceConn, err := servicenetworking.NewConnection(ctx, "svc-conn",
		&servicenetworking.ConnectionArgs{
			Network:               vpcId,
			Service:               pulumi.String("servicenetworking.googleapis.com"),
			ReservedPeeringRanges: pulumi.StringArray{privateIpAlloc.Name},
			DeletionPolicy:        pulumi.String("ABANDON"),
		},
		common.MergeOptions(opts,
			legacyAlias(legacyResourceName(ctx, projectName, "service-connection")))...,
	)
	if err != nil {
		return nil, err
	}
	return serviceConn, nil
}
