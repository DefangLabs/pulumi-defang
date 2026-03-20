package aws

import (
	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-awsx/sdk/v3/go/awsx/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

type NetworkingResult struct {
	VpcID            pulumix.Output[string]
	PublicSubnetIDs  pulumix.Output[[]string]
	PrivateSubnetIDs pulumix.Output[[]string]
}

// ResolveNetworking creates a new VPC using awsx or uses provided VPC/subnet IDs.
func ResolveNetworking(ctx *pulumi.Context, cfg *common.AWSConfig, opts ...pulumi.ResourceOption) (*NetworkingResult, error) {
	if cfg != nil && cfg.VpcID != "" {
		// Use provided VPC and subnet IDs
		subnetIDs := make(pulumi.StringArray, len(cfg.SubnetIDs))
		for i, id := range cfg.SubnetIDs {
			subnetIDs[i] = pulumi.String(id)
		}
		privateSubnetIDs := make(pulumi.StringArray, len(cfg.PrivateSubnetIDs))
		for i, id := range cfg.PrivateSubnetIDs {
			privateSubnetIDs[i] = pulumi.String(id)
		}
		if len(privateSubnetIDs) == 0 {
			privateSubnetIDs = subnetIDs
		}
		return &NetworkingResult{
			VpcID:            pulumix.Val(cfg.VpcID),
			PublicSubnetIDs:  pulumix.Output[[]string](subnetIDs.ToStringArrayOutput()),
			PrivateSubnetIDs: pulumix.Output[[]string](privateSubnetIDs.ToStringArrayOutput()),
		}, nil
	}

	// Create a new VPC with public and private subnets.
	// Use a descriptive logical name so awsx prefixes all children (subnets, etc.) with it.
	vpcName := autonamePrefix(ctx, "shared-vpc")
	vpc, err := ec2.NewVpc(ctx, vpcName, &ec2.VpcArgs{
		EnableDnsHostnames: pulumi.Bool(true),
		EnableDnsSupport:   pulumi.Bool(true),
		NatGateways: &ec2.NatGatewayConfigurationArgs{
			Strategy: ec2.NatGatewayStrategySingle,
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	return &NetworkingResult{
		VpcID:            pulumix.Output[string](vpc.VpcId),
		PublicSubnetIDs:  pulumix.Output[[]string](vpc.PublicSubnetIds),
		PrivateSubnetIDs: pulumix.Output[[]string](vpc.PrivateSubnetIds),
	}, nil
}
