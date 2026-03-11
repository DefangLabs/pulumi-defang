package aws

import (
	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type networkingResult struct {
	vpcID            pulumi.StringOutput
	publicSubnetIDs  pulumi.StringArrayOutput
	privateSubnetIDs pulumi.StringArrayOutput
}

// resolveNetworking creates a new VPC using awsx or uses provided VPC/subnet IDs.
func resolveNetworking(ctx *pulumi.Context, projectName string, cfg *common.AWSConfig, opts ...pulumi.ResourceOption) (*networkingResult, error) {
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
		return &networkingResult{
			vpcID:            pulumi.String(cfg.VpcID).ToStringOutput(),
			publicSubnetIDs:  subnetIDs.ToStringArrayOutput(),
			privateSubnetIDs: privateSubnetIDs.ToStringArrayOutput(),
		}, nil
	}

	// Create a new VPC with public and private subnets
	vpc, err := ec2.NewVpc(ctx, projectName+"-vpc", &ec2.VpcArgs{
		EnableDnsHostnames: pulumi.Bool(true),
		EnableDnsSupport:   pulumi.Bool(true),
		NatGateways: &ec2.NatGatewayConfigurationArgs{
			Strategy: ec2.NatGatewayStrategySingle,
		},
		Tags: pulumi.StringMap{
			"defang:project": pulumi.String(projectName),
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	return &networkingResult{
		vpcID:            vpc.VpcId,
		publicSubnetIDs:  vpc.PublicSubnetIds,
		privateSubnetIDs: vpc.PrivateSubnetIds,
	}, nil
}
