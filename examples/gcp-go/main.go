package main

import (
	defanggcp "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-gcp"
	"github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-gcp/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "")
		domain := cfg.Get("domain")

		project, err := defanggcp.NewProject(ctx, "gcp-go", &defanggcp.ProjectArgs{
			Domain: &domain,
			Services: compose.ServiceConfigMap{
				"app": compose.ServiceConfigArgs{
					Image: pulumi.StringPtr("nginx:latest"),
					Ports: compose.ServicePortConfigArray{
						compose.ServicePortConfigArgs{
							Target:      pulumi.Int(80),
							Mode:        pulumi.StringPtr("ingress"),
							AppProtocol: pulumi.StringPtr("http"),
						},
					},
				},
				"db": compose.ServiceConfigArgs{
					Image:    pulumi.StringPtr("postgres:17"),
					Postgres: &compose.PostgresConfigArgs{},
					Environment: pulumi.StringMap{
						"POSTGRES_PASSWORD": pulumi.String("supersecret"),
					},
					Ports: compose.ServicePortConfigArray{
						compose.ServicePortConfigArgs{
							Target: pulumi.Int(5432),
							Mode:   pulumi.StringPtr("host"),
						},
					},
				},
				"redis": compose.ServiceConfigArgs{
					Image: pulumi.StringPtr("redis:7"),
					Redis: &compose.RedisConfigArgs{},
					Ports: compose.ServicePortConfigArray{
						compose.ServicePortConfigArgs{
							Target: pulumi.Int(6379),
							Mode:   pulumi.StringPtr("host"),
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("endpoints", project.Endpoints)
		return nil
	})
}
