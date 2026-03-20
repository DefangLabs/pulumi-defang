package main

import (
	"github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-azure/defangazure"
	"github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-azure/shared"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		azureYaml, err := defangazure.NewProject(ctx, "azure-yaml", &defangazure.ProjectArgs{
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
		ctx.Export("endpoints", azureYaml.Endpoints)
		return nil
	})
}
