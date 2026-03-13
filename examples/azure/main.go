package main

import (
	defangazure "github.com/DefangLabs/pulumi-defang/sdk/go/defang-azure/defangazure"
	"github.com/DefangLabs/pulumi-defang/sdk/go/defang-azure/shared"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		proj, err := defangazure.NewProject(ctx, "myProject", &defangazure.ProjectArgs{
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
		})
		if err != nil {
			return err
		}

		ctx.Export("endpoints", proj.Endpoints)

		return nil
	})
}
