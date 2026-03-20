package main

import (
	defangaws "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws"
	"github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws/shared"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		awsYaml, err := defangaws.NewProject(ctx, "aws-yaml", &defangaws.ProjectArgs{
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
		ctx.Export("endpoints", awsYaml.Endpoints)
		return nil
	})
}
