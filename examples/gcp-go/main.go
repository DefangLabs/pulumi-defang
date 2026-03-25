package main

import (
	"github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-gcp/defanggcp"
	"github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-gcp/shared"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "")
		domain := cfg.Get("domain")

		gcpYaml, err := defanggcp.NewProject(ctx, "gcp-yaml", &defanggcp.ProjectArgs{
			Domain: pulumi.StringInput(pulumi.String(domain)),
			Services: shared.ServiceInputMap{
				"app": &shared.ServiceInputArgs{
					Image: pulumi.String("nginx"),
					Ports: shared.ServicePortConfigArray{
						&shared.ServicePortConfigArgs{
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
		ctx.Export("endpoints", gcpYaml.Endpoints)
		return nil
	})
}
