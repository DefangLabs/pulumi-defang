package main

import (
	"github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-gcp/defanggcp"
	"github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-gcp/shared"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		gcpYaml, err := defanggcp.NewProject(ctx, "gcp-yaml", &defanggcp.ProjectArgs{
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
