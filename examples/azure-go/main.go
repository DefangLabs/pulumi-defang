package main

import (
	defangazure "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-azure"
	"github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-azure/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		azureDemo, err := defangazure.NewProject(ctx, "azure-demo", &defangazure.ProjectArgs{
			Services: compose.ServiceConfigMap{
				"app": &compose.ServiceConfigArgs{
					// Image: pulumi.String("nginx"),
					Build: compose.BuildConfigArgs{
						Context:    pulumi.String("https://edwdefangtest.blob.core.windows.net/build-contexts/context.tar.gz?sp=r&st=2026-03-27T21:00:58Z&se=2026-04-24T05:15:58Z&spr=https&sv=2024-11-04&sr=b&sig=bYmzacQ%2B1xhuyXxAB4CKHeab5n8v6Q8djoLJ4PBwtOo%3D"),
						Dockerfile: pulumi.String("Dockerfile"),
					},
					Ports: compose.ServicePortConfigArray{
						&compose.ServicePortConfigArgs{
							Target:      pulumi.Int(80),
							Mode:        pulumi.String("ingress"),
							AppProtocol: pulumi.String("http"),
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("endpoints", azureDemo.Endpoints)
		postgresImage := "postgres:17"
		pg, err := defangazure.NewPostgres(ctx, "postgres", &defangazure.PostgresArgs{
			Project_name: "azure-demo",
			Image:        &postgresImage,
			Postgres:     &compose.PostgresConfigArgs{},
			Environment: pulumi.StringMap{
				"POSTGRES_PASSWORD": pulumi.String("Ch4ng3M3pl3as3!"),
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("postgresEndpoint", pg.Endpoint)
		//
		// redis, err := defangazure.NewRedis(ctx, "redis", &defangazure.RedisArgs{
		// 	Project_name: "azure-demo",
		// })
		// if err != nil {
		// 	return err
		// }
		// ctx.Export("redisEndpoint", redis.Endpoint)
		return nil
	})
}
