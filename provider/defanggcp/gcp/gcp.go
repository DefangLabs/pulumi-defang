package gcp

import (
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// GlobalConfig holds project-level GCP resources shared across all services.
type GlobalConfig struct {
	Region   string
	VpcId    pulumi.StringOutput
	SubnetId pulumi.StringOutput
}

// BuildGlobalConfig creates shared GCP infrastructure for a multi-service project.
func BuildGlobalConfig(
	ctx *pulumi.Context,
	projectName string,
	services map[string]compose.ServiceConfig,
	opts ...pulumi.ResourceOption,
) (*GlobalConfig, error) {
	region := GcpRegion(ctx)

	vpc, err := compute.NewNetwork(ctx, projectName+"-vpc", &compute.NetworkArgs{
		AutoCreateSubnetworks: pulumi.Bool(false),
	}, append(opts, pulumi.RetainOnDelete(true))...)
	if err != nil {
		return nil, err
	}

	subnet, err := compute.NewSubnetwork(ctx, projectName+"-shared-subnet", &compute.SubnetworkArgs{
		IpCidrRange: pulumi.String("10.0.0.0/16"),
		Region:      pulumi.String(region),
		Network:     vpc.ID(),
	}, append(opts, pulumi.RetainOnDelete(true))...)
	if err != nil {
		return nil, err
	}

	return &GlobalConfig{
		Region:   region,
		VpcId:    vpc.ID().ToStringOutput(),
		SubnetId: subnet.ID().ToStringOutput(),
	}, nil
}

const defaultGCPRegion = "us-central1"

// GcpRegion reads the GCP region from Pulumi stack config, falling back to the default.
func GcpRegion(ctx *pulumi.Context) string {
	cfg := config.New(ctx, "gcp")
	if r := cfg.Get("region"); r != "" {
		return r
	}
	return defaultGCPRegion
}
