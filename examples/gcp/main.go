package main

import (
	defanggcp "github.com/DefangLabs/pulumi-defang/sdk/go/defang-gcp/defanggcp"
	"github.com/DefangLabs/pulumi-defang/sdk/go/defang-gcp/shared"
	"github.com/pulumi/pulumi-gcp/sdk/v8/go/gcp"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create GCP provider — user controls project, region, auth
		gcpProv, err := gcp.NewProvider(ctx, "gcp", &gcp.ProviderArgs{
			Project: pulumi.StringPtr("my-gcp-project"),
			Region:  pulumi.StringPtr("us-central1"),
		})
		if err != nil {
			return err
		}

		// GCP Project — services deployed to Cloud Run + Cloud SQL
		gcpProj, err := defanggcp.NewProject(ctx, "gcpProject", &defanggcp.ProjectArgs{
			Services: shared.ServiceInputMap{
				"web": shared.ServiceInputArgs{
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
						"POSTGRES_USER":     pulumi.String("defang"),
						"POSTGRES_PASSWORD": pulumi.String("secret"),
						"POSTGRES_DB":       pulumi.String("mydb"),
					},
				},
			},
		}, pulumi.Providers(gcpProv))
		if err != nil {
			return err
		}

		ctx.Export("gcpEndpoints", gcpProj.Endpoints)

		return nil
	})
}
