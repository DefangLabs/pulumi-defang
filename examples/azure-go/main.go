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
					Image: pulumi.String("nginx"),
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
		return nil
	})
}
