package main

import (
	defangaws "github.com/DefangLabs/pulumi-defang/sdk/go/defang-aws/defangaws"
	"github.com/DefangLabs/pulumi-defang/sdk/go/defang-aws/shared"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		proj, err := defangaws.NewProject(ctx, "myProject", &defangaws.ProjectArgs{
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
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("endpoints", proj.Endpoints)

		return nil
	})
}
