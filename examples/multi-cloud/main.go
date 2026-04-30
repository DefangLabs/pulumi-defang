package main

import (
	defangaws "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws"
	awscompose "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws/compose"
	defangazure "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-azure"
	azurecompose "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-azure/compose"
	defanggcp "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-gcp"
	gcpcompose "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-gcp/compose"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	pulumiazurenative "github.com/pulumi/pulumi-azure-native-sdk/v3"
	"github.com/pulumi/pulumi-gcp/sdk/v8/go/gcp"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// --- AWS ---
		awsProv, err := aws.NewProvider(ctx, "aws", &aws.ProviderArgs{
			Region: pulumi.String("us-west-2"),
		})
		if err != nil {
			return err
		}

		awsProj, err := defangaws.NewProject(ctx, "awsProject", &defangaws.ProjectArgs{
			Services: awscompose.ServiceConfigMap{
				"web": awscompose.ServiceConfigArgs{
					Image: pulumi.String("nginx:latest"),
					Ports: awscompose.ServicePortConfigArray{
						awscompose.ServicePortConfigArgs{
							Target:      pulumi.Int(80),
							Mode:        pulumi.String("ingress"),
							AppProtocol: pulumi.String("http"),
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
			Project: pulumi.String("liotest-443018"),
			Region:  pulumi.String("us-central1"),
		})
		if err != nil {
			return err
		}

		gcpProj, err := defanggcp.NewProject(ctx, "gcpProject", &defanggcp.ProjectArgs{
			Services: gcpcompose.ServiceConfigMap{
				"web": gcpcompose.ServiceConfigArgs{
					Image: pulumi.String("nginx:latest"),
					Ports: gcpcompose.ServicePortConfigArray{
						gcpcompose.ServicePortConfigArgs{
							Target:      pulumi.Int(80),
							Mode:        pulumi.String("ingress"),
							AppProtocol: pulumi.String("http"),
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
			Location: pulumi.String("eastus"),
		})
		if err != nil {
			return err
		}

		azureProj, err := defangazure.NewProject(ctx, "azureProject", &defangazure.ProjectArgs{
			Services: azurecompose.ServiceConfigMap{
				"web": azurecompose.ServiceConfigArgs{
					Image: pulumi.String("nginx:latest"),
					Ports: azurecompose.ServicePortConfigArray{
						azurecompose.ServicePortConfigArgs{
							Target:      pulumi.Int(80),
							Mode:        pulumi.String("ingress"),
							AppProtocol: pulumi.String("http"),
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
