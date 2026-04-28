package aws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/route53"
	awsxec2 "github.com/pulumi/pulumi-awsx/sdk/v3/go/awsx/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type NetworkingResult struct {
	VpcID            pulumi.StringOutput
	PublicSubnetIDs  pulumi.StringArrayOutput
	PrivateSubnetIDs pulumi.StringArrayOutput
	PrivateDomain    string
	PrivateZone      *route53.Zone            // optional Route53 private hosted zone ID for the VPC
	PublicNatIPs     pulumi.StringArrayOutput // only populated if UseNatGW is true and strategy is OnePerAz
	UseNatGW         bool                     // whether to use NAT Gateways (vs. public subnets for outbound)
}

// ResolveNetworking creates a new VPC using awsx or uses provided VPC/subnet IDs.
func ResolveNetworking(
	ctx *pulumi.Context, projectName string, opt pulumi.ResourceOrInvokeOption,
) (*NetworkingResult, error) {
	privateDomain := common.SafeLabel(projectName) + ".internal"

	strategy := awsxec2.NatGatewayStrategy(NatGatewayStrategy.Get(ctx)) // TODO: missing type checking
	numberOfAZs := NumberOfAvailabilityZones.Get(ctx)

	// if cfg != nil && cfg.VpcID != "" {
	// 	// Use provided VPC and subnet IDs
	// 	subnetIDs := make(pulumi.StringArray, len(cfg.PublicSubnetIDs))
	// 	for i, id := range cfg.PublicSubnetIDs {
	// 		subnetIDs[i] = pulumi.String(id)
	// 	}
	// 	privateSubnetIDs := make(pulumi.StringArray, len(cfg.PrivateSubnetIDs))
	// 	for i, id := range cfg.PrivateSubnetIDs {
	// 		privateSubnetIDs[i] = pulumi.String(id)
	// 	}
	// 	if len(privateSubnetIDs) == 0 {
	// 		privateSubnetIDs = subnetIDs
	// 	}
	// 	return &NetworkingResult{
	// 		VpcID:            pulumi.String(cfg.VpcID).ToStringOutput(),
	// 		PublicSubnetIDs:  subnetIDs.ToStringArrayOutput(),
	// 		PrivateSubnetIDs: privateSubnetIDs.ToStringArrayOutput(),
	// 		UseNatGW:         strategy != awsxec2.NatGatewayStrategyNone,
	// 		// PrivateZoneID:    pulumi.IDPtrOutput{},  no private hosted zone since we didn't create the VPC
	// 	}, nil
	// }

	region, err := aws.GetRegion(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("getting AWS region: %w", err)
	}

	// Probe the AZs available to this account in this region. If fewer are accessible than the
	// recipe asks for, awsx would fail with "does not have at least N Availability Zones"; instead
	// pin the explicit list and warn so the user can lower the recipe value or change region.
	azs, err := aws.GetAvailabilityZones(ctx, &aws.GetAvailabilityZonesArgs{State: common.Ptr("available")}, opt)
	if err != nil {
		return nil, fmt.Errorf("getting AWS availability zones: %w", err)
	}

	if len(azs.Names) < numberOfAZs {
		// slices.Sort(azs.Names) // deterministic order: AWS does not guarantee response ordering
		_ = ctx.Log.Warn(fmt.Sprintf("region %s only has %d availability zone(s) (%v); recipe %q requested %d",
			region.Region, len(azs.Names), azs.Names, "number-of-availability-zones", numberOfAZs), nil)
		numberOfAZs = len(azs.Names)
	}

	// Create a new VPC with public and private subnets.
	vpcName := common.AutonamingPrefix(ctx, "shared-vpc")

	vpcArgs := &awsxec2.VpcArgs{
		EnableDnsHostnames: pulumi.Bool(true),
		// EnableDnsSupport:   pulumi.Bool(true),
		NatGateways: &awsxec2.NatGatewayConfigurationArgs{
			Strategy: strategy,
		},
		NumberOfAvailabilityZones: &numberOfAZs,
		VpcEndpointSpecs: []awsxec2.VpcEndpointSpecArgs{
			{
				ServiceName:     fmt.Sprintf("com.amazonaws.%s.s3", region.Region),
				VpcEndpointType: pulumi.String("Gateway"), // Gateway is free
				// Tags: pulumi.StringMap{
				// 	"Name": pulumi.String(fmt.Sprintf("%s-s3-endpoint", vpcName)),
				// },
			},
		},
		Tags: pulumi.StringMap{
			"Name": pulumi.String(vpcName),
		},
	}

	vpc, err := awsxec2.NewVpc(ctx, "shared-vpc", vpcArgs, opt)
	if err != nil {
		return nil, err
	}

	// TODO: make this optional, so we can save $$
	privateZone, err := route53.NewZone(ctx, privateDomain, &route53.ZoneArgs{
		Comment:      pulumi.String(common.DefangComment),
		Name:         pulumi.String(privateDomain),
		ForceDestroy: pulumi.Bool(ForceDestroyHostedzone.Get(ctx)),
		Vpcs: route53.ZoneVpcArray{
			route53.ZoneVpcArgs{VpcId: vpc.VpcId},
		},
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating Route53 private hosted zone: %w", err)
	}

	// Lower the negative caching TTL to 15 seconds
	_, err = createSoaRecord(ctx, privateDomain, privateZone.ToZoneOutput(), SoaRecordArgs{
		Serial:  pulumi.Int(2023022101),
		Minimum: pulumi.Int(15),
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating SOA record: %w", err)
	}

	options, err := ec2.NewVpcDhcpOptions(ctx, "dhcp-options", &ec2.VpcDhcpOptionsArgs{
		DomainName:        pulumi.String(privateDomain),
		DomainNameServers: pulumi.StringArray{pulumi.String("AmazonProvidedDNS")},
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating VPC DHCP options: %w", err)
	}

	_, err = ec2.NewVpcDhcpOptionsAssociation(ctx, "dhcp-options-association", &ec2.VpcDhcpOptionsAssociationArgs{
		DhcpOptionsId: options.ID(),
		VpcId:         vpc.VpcId,
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating VPC DHCP options association: %w", err)
	}

	publicNatIps := vpc.NatGateways.ApplyT(func(ngws []*ec2.NatGateway) (pulumi.StringArray, error) {
		ips := make(pulumi.StringArray, len(ngws))
		for i, ngw := range ngws {
			ips[i] = ngw.PublicIp
		}
		return ips, nil
	}).(pulumi.StringArrayOutput)

	return &NetworkingResult{
		VpcID:            vpc.VpcId,
		PublicSubnetIDs:  vpc.PublicSubnetIds,
		PrivateSubnetIDs: vpc.PrivateSubnetIds,
		PrivateDomain:    privateDomain,
		PrivateZone:      privateZone,
		PublicNatIPs:     publicNatIps,
		UseNatGW:         strategy != awsxec2.NatGatewayStrategyNone,
	}, nil
}
