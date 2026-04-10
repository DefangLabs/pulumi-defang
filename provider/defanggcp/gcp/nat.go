package gcp

import (
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func CreateNAT(
	ctx *pulumi.Context, vpcId pulumi.StringInput, config GlobalConfig, opts ...pulumi.ResourceOption,
) error {
	router, err := compute.NewRouter(ctx, "nat-router", &compute.RouterArgs{
		Region:  pulumi.String(config.Region),
		Network: vpcId,
		Bgp: &compute.RouterBgpArgs{
			Asn: pulumi.Int(64514), // 64512-65534 is the range for private ASNs
		},
	}, opts...)
	if err != nil {
		return err
	}
	_, err = compute.NewRouterNat(ctx, "nat", &compute.RouterNatArgs{
		Router:                        router.Name,
		Region:                        router.Region,
		NatIpAllocateOption:           pulumi.String("AUTO_ONLY"),
		SourceSubnetworkIpRangesToNat: pulumi.String("ALL_SUBNETWORKS_ALL_IP_RANGES"),
		LogConfig: &compute.RouterNatLogConfigArgs{
			Enable: pulumi.Bool(true),
			Filter: pulumi.String("ERRORS_ONLY"),
		},
	}, opts...)
	return err
}
