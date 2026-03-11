package main

import (
	"github.com/DefangLabs/pulumi-defang/sdk/go/defang"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// AWS Project with ECS services + Postgres
		awsProj, err := defang.NewProject(ctx, "awsProject", &defang.ProjectArgs{
			ProviderId: "aws",
			Services: defang.ServiceInputMap{
				"web": defang.ServiceInputArgs{
					Image: pulumi.StringPtr("nginx:latest"),
					Ports: defang.PortConfigArray{
						defang.PortConfigArgs{
							Target:      pulumi.Int(80),
							Mode:        pulumi.StringPtr("ingress"),
							AppProtocol: pulumi.StringPtr("http"),
						},
					},
				},
				"app": defang.ServiceInputArgs{
					Build: defang.BuildInputArgs{
						Context: pulumi.String("s3://my-bucket/app-context.tar.gz"),
					},
					Ports: defang.PortConfigArray{
						defang.PortConfigArgs{
							Target:      pulumi.Int(8080),
							Mode:        pulumi.StringPtr("ingress"),
							AppProtocol: pulumi.StringPtr("http"),
						},
					},
				},
				"db": defang.ServiceInputArgs{
					Image:    pulumi.StringPtr("postgres:16"),
					Postgres: defang.PostgresInputArgs{},
					Environment: pulumi.StringMap{
						"POSTGRES_USER":     pulumi.String("myuser"),
						"POSTGRES_PASSWORD": pulumi.String("mypassword"),
						"POSTGRES_DB":       pulumi.String("mydb"),
					},
				},
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("awsEndpoints", awsProj.Endpoints)
		ctx.Export("awsLoadBalancerDns", awsProj.LoadBalancerDns)

		// GCP Project with Cloud Run service + Cloud SQL
		gcpProj, err := defang.NewProject(ctx, "gcpProject", &defang.ProjectArgs{
			ProviderId: "gcp",
			Services: defang.ServiceInputMap{
				"api": defang.ServiceInputArgs{
					Image: pulumi.StringPtr("gcr.io/my-project/api:latest"),
					Ports: defang.PortConfigArray{
						defang.PortConfigArgs{
							Target:      pulumi.Int(8080),
							Mode:        pulumi.StringPtr("ingress"),
							AppProtocol: pulumi.StringPtr("http"),
						},
					},
				},
				"db": defang.ServiceInputArgs{
					Image:    pulumi.StringPtr("postgres:17"),
					Postgres: defang.PostgresInputArgs{},
					Environment: pulumi.StringMap{
						"POSTGRES_USER":     pulumi.String("gcpuser"),
						"POSTGRES_PASSWORD": pulumi.String("gcppassword"),
						"POSTGRES_DB":       pulumi.String("gcpdb"),
					},
				},
			},
			Gcp: defang.GCPConfigInputArgs{
				Project: pulumi.StringPtr("my-gcp-project"),
				Region:  pulumi.StringPtr("us-central1"),
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("gcpEndpoints", gcpProj.Endpoints)

		// Azure Project with Container Apps + Postgres
		azureProj, err := defang.NewProject(ctx, "azureProject", &defang.ProjectArgs{
			ProviderId: "azure",
			Services: defang.ServiceInputMap{
				"api": defang.ServiceInputArgs{
					Image: pulumi.StringPtr("nginx:latest"),
					Ports: defang.PortConfigArray{
						defang.PortConfigArgs{
							Target:      pulumi.Int(80),
							Mode:        pulumi.StringPtr("ingress"),
							AppProtocol: pulumi.StringPtr("http"),
						},
					},
				},
				"db": defang.ServiceInputArgs{
					Image:    pulumi.StringPtr("postgres:16"),
					Postgres: defang.PostgresInputArgs{},
					Environment: pulumi.StringMap{
						"POSTGRES_USER":     pulumi.String("azureuser"),
						"POSTGRES_PASSWORD": pulumi.String("azurepassword"),
						"POSTGRES_DB":       pulumi.String("azuredb"),
					},
				},
			},
			Azure: defang.AzureConfigInputArgs{
				Location: pulumi.StringPtr("eastus"),
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("azureEndpoints", azureProj.Endpoints)

		return nil
	})
}
