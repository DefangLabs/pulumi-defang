package main

import (
	defangscaleway "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-scaleway"
	"github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-scaleway/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		scalewayDemo, err := defangscaleway.NewProject(ctx, "scaleway-demo", &defangscaleway.ProjectArgs{
			Services: compose.ServiceConfigMap{
				"app": compose.ServiceConfigArgs{
					Image: pulumi.StringPtr("nginx"),
					Ports: compose.ServicePortConfigArray{
						compose.ServicePortConfigArgs{
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
		ctx.Export("endpoints", scalewayDemo.Endpoints)
		return nil
	})
}
