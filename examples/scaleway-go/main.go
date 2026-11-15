package main

import (
	defangscaleway "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-scaleway"
	"github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-scaleway/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "")
		postgresPassword := cfg.RequireSecret("POSTGRES_PASSWORD")
		databaseUrl := cfg.RequireSecret("DATABASE_URL")
		apiUrl := cfg.Require("API_URL")
		apiImage := cfg.Require("API_IMAGE")
		webImage := cfg.Require("WEB_IMAGE")
		db, err := defangscaleway.NewPostgres(ctx, "db", &defangscaleway.PostgresArgs{
			ProjectName: "scaleway-e2e",
			Postgres:    &compose.PostgresConfigArgs{},
			Environment: pulumi.StringMap{
				"POSTGRES_DB":       pulumi.String("defang"),
				"POSTGRES_USER":     pulumi.String("defang"),
				"POSTGRES_PASSWORD": pulumi.String("${POSTGRES_PASSWORD}"),
			},
		})
		if err != nil {
			return err
		}
		api, err := defangscaleway.NewService(ctx, "api", &defangscaleway.ServiceArgs{
			ProjectName: "scaleway-e2e-api",
			Image:       pulumi.String(apiImage),
			Ports: compose.ServicePortConfigArray{
				&compose.ServicePortConfigArgs{
					Target:      pulumi.Int(8080),
					Mode:        pulumi.String("ingress"),
					AppProtocol: pulumi.String("http"),
				},
			},
			Environment: pulumi.StringMap{
				"DATABASE_URL": pulumi.String("${DATABASE_URL}"),
			},
		})
		if err != nil {
			return err
		}
		web, err := defangscaleway.NewService(ctx, "web", &defangscaleway.ServiceArgs{
			ProjectName: "scaleway-e2e-web",
			Image:       pulumi.String(webImage),
			Ports: compose.ServicePortConfigArray{
				&compose.ServicePortConfigArgs{
					Target:      pulumi.Int(8080),
					Mode:        pulumi.String("ingress"),
					AppProtocol: pulumi.String("http"),
				},
			},
			Environment: pulumi.StringMap{
				"API_URL": pulumi.String("${API_URL}"),
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("dbEndpoint", db.Endpoint)
		ctx.Export("databaseUrl", db.ConnectionUrl)
		ctx.Export("apiEndpoint", api.Endpoint)
		ctx.Export("webEndpoint", web.Endpoint)
		return nil
	})
}
