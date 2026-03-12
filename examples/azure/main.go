package main

import (
	defangazure "github.com/DefangLabs/pulumi-defang/sdk/go/defang-azure/defangazure"
	"github.com/DefangLabs/pulumi-defang/sdk/go/defang-azure/shared"
	pulumiazurenative "github.com/pulumi/pulumi-azure-native-sdk/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create Azure provider — user controls location, auth
		azureProv, err := pulumiazurenative.NewProvider(ctx, "azure", &pulumiazurenative.ProviderArgs{
			Location: pulumi.StringPtr("eastus"),
		})
		if err != nil {
			return err
		}

		// Azure Project — services deployed to Container Apps + PostgreSQL Flexible Server
		azureProj, err := defangazure.NewProject(ctx, "azureProject", &defangazure.ProjectArgs{
			Services: shared.ServiceInputMap{
				"api": shared.ServiceInputArgs{
					Image: pulumi.StringPtr("nginx:latest"),
					Ports: shared.PortConfigArray{
						shared.PortConfigArgs{
							Target:      pulumi.Int(80),
							Mode:        pulumi.StringPtr("ingress"),
							AppProtocol: pulumi.StringPtr("http"),
						},
					},
				},
				"db": shared.ServiceInputArgs{
					Image:    pulumi.StringPtr("postgres:16"),
					Postgres: shared.PostgresInputArgs{},
					Environment: pulumi.StringMap{
						"POSTGRES_USER":     pulumi.String("azureuser"),
						"POSTGRES_PASSWORD": pulumi.String("azurepassword"),
						"POSTGRES_DB":       pulumi.String("azuredb"),
					},
				},
			},
		}, pulumi.Providers(azureProv))
		if err != nil {
			return err
		}

		ctx.Export("azureEndpoints", azureProj.Endpoints)

		return nil
	})
}
