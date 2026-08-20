package gcp

import (
	"fmt"

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

// peeringCleanupResource is the Pulumi resource state for the
// defang-gcp:defanggcp:PeeringCleanup custom resource. It has no outputs: the
// resource exists only for what its delete does.
type peeringCleanupResource struct {
	pulumi.CustomResourceState
}

// createVPCPeeringInfra allocates a private IP range and creates a service networking
// connection for Cloud SQL private IP access.
//
// It returns the peering-cleanup resource rather than the connection, because that
// is what the managed instances must depend on — see PeeringCleanup.
func createVPCPeeringInfra(
	ctx *pulumi.Context,
	projectName string,
	gcpProject string,
	vpcId pulumi.StringOutput,
	opts ...pulumi.ResourceOption,
) (pulumi.Resource, error) {
	privateIpAlloc, err := compute.NewGlobalAddress(ctx, projectName+"-peering-ip", &compute.GlobalAddressArgs{
		Purpose:      pulumi.String("VPC_PEERING"),
		AddressType:  pulumi.String("INTERNAL"),
		PrefixLength: pulumi.Int(16),
		Network:      vpcId,
	}, opts...)
	if err != nil {
		return nil, err
	}

	// The provider cannot delete a service networking connection at all:
	// hashicorp/terraform-provider-google#16275 is a 5.x regression
	// (removePeering -> deleteConnection) that fails even once the dependent
	// Cloud SQL instances are gone. It was closed as a duplicate of #16944,
	// whose resolution — merged 2024-01-09 and still the current behaviour —
	// added this *abandon* deletion policy instead of a working delete. So
	// ABANDON is upstream's own answer here, not a workaround invented for this
	// repo, and a plain delete would fail for ever.
	//
	// ABANDON makes the destroy skip the API call entirely, which is what lets
	// Pulumi delete the rest of the VPC. What it leaves behind is the network
	// peering, and that still holds the reserved range above — so the
	// PeeringCleanup resource below removes it, with the Compute API call that
	// Google's own console uses.
	//
	// The field is Optional+Computed and not ForceNew upstream, so setting it on
	// an existing connection updates in place and never replaces the peering.
	serviceConn, err := servicenetworking.NewConnection(ctx, projectName+"-svc-conn",
		&servicenetworking.ConnectionArgs{
			Network:               vpcId,
			Service:               pulumi.String("servicenetworking.googleapis.com"),
			ReservedPeeringRanges: pulumi.StringArray{privateIpAlloc.Name},
			DeletionPolicy:        pulumi.String("ABANDON"),
		},
		opts...,
	)
	if err != nil {
		return nil, err
	}

	// Deleting this removes the peering the abandoned connection leaves behind.
	// It depends on the connection, so Pulumi deletes it first — and the managed
	// instances depend on it in turn (SharedInfra.ServiceNetworking), which puts
	// the whole teardown in the only order GCP accepts: instances, peering,
	// reserved range, subnet, VPC.
	cleanupOpts := make([]pulumi.ResourceOption, 0, len(opts)+1)
	cleanupOpts = append(cleanupOpts, opts...)
	cleanupOpts = append(cleanupOpts, pulumi.DependsOn([]pulumi.Resource{serviceConn}))

	var cleanup peeringCleanupResource
	if err := ctx.RegisterResource(
		"defang-gcp:defanggcp:PeeringCleanup",
		projectName+"-svc-conn-peering",
		pulumi.Map{
			"projectId": pulumi.String(gcpProject),
			"network":   vpcId,
		},
		&cleanup,
		cleanupOpts...,
	); err != nil {
		return nil, fmt.Errorf("creating PeeringCleanup resource: %w", err)
	}
	return &cleanup, nil
}
