package main

import (
	defangaws "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws/defangaws"
	awsShared "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws/shared"
	defangazure "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-azure/defangazure"
	azureShared "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-azure/shared"
	defanggcp "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-gcp/defanggcp"
	gcpShared "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-gcp/shared"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	pulumiazurenative "github.com/pulumi/pulumi-azure-native-sdk/v2"
	"github.com/pulumi/pulumi-gcp/sdk/v8/go/gcp"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// --- AWS ---
		awsProv, err := aws.NewProvider(ctx, "aws", &aws.ProviderArgs{
			Region: pulumi.StringPtr("us-west-2"),
		})
		if err != nil {
			return err
		}

		awsProj, err := defangaws.NewProject(ctx, "awsProject", &defangaws.ProjectArgs{
			Services: awsShared.ServiceInputMap{
				"web": awsShared.ServiceInputArgs{
					Image: pulumi.StringPtr("nginx:latest"),
					Ports: awsShared.PortConfigArray{
						awsShared.PortConfigArgs{
							Target:      pulumi.Int(80),
							Mode:        pulumi.StringPtr("ingress"),
							AppProtocol: pulumi.StringPtr("http"),
						},
					},
				},
			},
		}, pulumi.Providers(awsProv))
		if err != nil {
			return err
		}

		// --- GCP ---
		gcpProv, err := gcp.NewProvider(ctx, "gcp", &gcp.ProviderArgs{
			Project: pulumi.StringPtr("liotest-443018"),
			Region:  pulumi.StringPtr("us-central1"),
		})
		if err != nil {
			return err
		}

		gcpProj, err := defanggcp.NewProject(ctx, "gcpProject", &defanggcp.ProjectArgs{
			Services: gcpShared.ServiceInputMap{
				"web": gcpShared.ServiceInputArgs{
					Image: pulumi.StringPtr("nginx:latest"),
					Ports: gcpShared.PortConfigArray{
						gcpShared.PortConfigArgs{
							Target:      pulumi.Int(80),
							Mode:        pulumi.StringPtr("ingress"),
							AppProtocol: pulumi.StringPtr("http"),
						},
					},
				},
			},
		}, pulumi.Providers(gcpProv))
		if err != nil {
			return err
		}

		// --- Azure ---
		azureProv, err := pulumiazurenative.NewProvider(ctx, "azure", &pulumiazurenative.ProviderArgs{
			Location: pulumi.StringPtr("eastus"),
		})
		if err != nil {
			return err
		}

		azureProj, err := defangazure.NewProject(ctx, "azureProject", &defangazure.ProjectArgs{
			Services: azureShared.ServiceInputMap{
				"web": azureShared.ServiceInputArgs{
					Image: pulumi.StringPtr("nginx:latest"),
					Ports: azureShared.PortConfigArray{
						azureShared.PortConfigArgs{
							Target:      pulumi.Int(80),
							Mode:        pulumi.StringPtr("ingress"),
							AppProtocol: pulumi.StringPtr("http"),
						},
					},
				},
			},
		}, pulumi.Providers(azureProv))
		if err != nil {
			return err
		}

		ctx.Export("awsEndpoints", awsProj.Endpoints)
		ctx.Export("gcpEndpoints", gcpProj.Endpoints)
		ctx.Export("azureEndpoints", azureProj.Endpoints)

		return nil
	})
}
