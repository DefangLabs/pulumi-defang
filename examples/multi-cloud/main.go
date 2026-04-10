package main

import (
	defangaws "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws"
	defangazure "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-azure"
	defanggcp "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-gcp"
	"github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-gcp/compose"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	pulumiazurenative "github.com/pulumi/pulumi-azure-native-sdk/v3"
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
			Services: compose.ServiceConfigMap{
				"web": compose.ServiceConfigArgs{
					Image: pulumi.StringPtr("nginx:latest"),
					Ports: compose.ServicePortConfigArray{
						compose.ServicePortConfigArgs{
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
			Services: compose.ServiceConfigMap{
				"web": compose.ServiceConfigArgs{
					Image: pulumi.StringPtr("nginx:latest"),
					Ports: compose.ServicePortConfigArray{
						compose.ServicePortConfigArgs{
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
			Services: compose.ServiceConfigMap{
				"web": compose.ServiceConfigArgs{
					Image: pulumi.StringPtr("nginx:latest"),
					Ports: compose.ServicePortConfigArray{
						compose.ServicePortConfigArgs{
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
