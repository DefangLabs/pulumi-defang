package aws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/route53"
	"github.com/pulumi/pulumi-awsx/sdk/v3/go/awsx/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type NetworkingResult struct {
	VpcID            pulumi.StringOutput
	PublicSubnetIDs  pulumi.StringArrayOutput
	PrivateSubnetIDs pulumi.StringArrayOutput
	PrivateDomain    string
	PrivateZoneID    pulumi.IDOutput // optional Route53 private hosted zone ID for the VPC
}

// ResolveNetworking creates a new VPC using awsx or uses provided VPC/subnet IDs.
func ResolveNetworking(ctx *pulumi.Context, cfg *common.AWSConfig, recipe Recipe, opts ...pulumi.ResourceOption) (*NetworkingResult, error) {
	if cfg != nil && cfg.VpcID != "" {
		// Use provided VPC and subnet IDs
		subnetIDs := make(pulumi.StringArray, len(cfg.PublicSubnetIDs))
		for i, id := range cfg.PublicSubnetIDs {
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
			VpcID:            pulumi.String(cfg.VpcID).ToStringOutput(),
			PublicSubnetIDs:  subnetIDs.ToStringArrayOutput(),
			PrivateSubnetIDs: privateSubnetIDs.ToStringArrayOutput(),
		}, nil
	}

	// Create a new VPC with public and private subnets.
	// Use a descriptive logical name so awsx prefixes all children (subnets, etc.) with it.
	vpcName := autonamePrefix(ctx, "shared-vpc")
	vpc, err := ec2.NewVpc(ctx, vpcName, &ec2.VpcArgs{
		EnableDnsHostnames: pulumi.Bool(true),
		EnableDnsSupport:   pulumi.Bool(true),
		NatGateways: &ec2.NatGatewayConfigurationArgs{
			Strategy: ec2.NatGatewayStrategySingle, // FIXME: from recipe
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	privateDomain := fmt.Sprintf("%s-%s.internal", ctx.Project(), ctx.Stack())

	// TODO: make this optional, so we can save $$
	privateZone, err := route53.NewZone(ctx, privateDomain, &route53.ZoneArgs{
		Comment:      pulumi.String(common.DefangComment),
		Name:         pulumi.String(privateDomain),
		ForceDestroy: pulumi.Bool(recipe.ForceDestroyHostedzone),
		Vpcs: route53.ZoneVpcArray{
			route53.ZoneVpcArgs{VpcId: vpc.VpcId},
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Route53 private hosted zone: %w", err)
	}

	// Lower the negative caching TTL to 15 seconds
	_, err = CreateSoaRecord(ctx, privateDomain, privateZone.ToZoneOutput(), SoaRecordArgs{
		Serial:  pulumi.Int(2023022101),
		Minimum: pulumi.Int(15),
	}, recipe, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating SOA record: %w", err)
	}

	return &NetworkingResult{
		VpcID:            vpc.VpcId,
		PublicSubnetIDs:  vpc.PublicSubnetIds,
		PrivateSubnetIDs: vpc.PrivateSubnetIds,
		PrivateDomain:    privateDomain,
		PrivateZoneID:    privateZone.ID(),
	}, nil
}
