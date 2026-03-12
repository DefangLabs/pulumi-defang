package main

import (
	defangaws "github.com/DefangLabs/pulumi-defang/sdk/go/defang-aws/defangaws"
	"github.com/DefangLabs/pulumi-defang/sdk/go/defang-aws/shared"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create AWS provider — user controls auth, region, tags
		awsProv, err := aws.NewProvider(ctx, "aws", &aws.ProviderArgs{
			Region: pulumi.StringPtr("us-west-2"),
		})
		if err != nil {
			return err
		}

		// AWS Project — services deployed to ECS + RDS
		awsProj, err := defangaws.NewProject(ctx, "awsProject", &defangaws.ProjectArgs{
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
				"app": shared.ServiceInputArgs{
					Image: pulumi.StringPtr("nginx:latest"),
					Ports: shared.PortConfigArray{
						shared.PortConfigArgs{
							Target:      pulumi.Int(8080),
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
		}, pulumi.Providers(awsProv))
		if err != nil {
			return err
		}

		ctx.Export("awsEndpoints", awsProj.Endpoints)
		ctx.Export("awsLoadBalancerDns", awsProj.LoadBalancerDns)

		return nil
	})
}
