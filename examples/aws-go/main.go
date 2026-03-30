package main

import (
	defangaws "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws"
	"github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		awsDemo, err := defangaws.NewProject(ctx, "aws-demo", &defangaws.ProjectArgs{
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
		ctx.Export("endpoints", awsDemo.Endpoints)
		return nil
	})
}
