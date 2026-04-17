package main

import (
	defangaws "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws"
	"github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws/compose"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ssm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		awsConfig := config.New(ctx, "aws")
		awsProvider, err := aws.NewProvider(ctx, "aws", &aws.ProviderArgs{
			Region: pulumi.String(awsConfig.Require("region")),
		})
		if err != nil {
			return err
		}

		// Create SSM param /Defang/config-example/dev/CONFIG with value "secr3t"
		param, err := ssm.NewParameter(ctx, "config", &ssm.ParameterArgs{
			Type:  pulumi.String("String"),
			Name:  pulumi.String("/Defang/config-example/dev/CONFIG"),
			Value: pulumi.String("secr3t"),
		})
		if err != nil {
			return err
		}

		proj, err := defangaws.NewProject(ctx, "config-example", &defangaws.ProjectArgs{
			Services: compose.ServiceConfigMap{
				"web": compose.ServiceConfigArgs{
					Image: pulumi.StringPtr("busybox"),
					Command: pulumi.StringArray{
						pulumi.String("sh"), pulumi.String("-c"),
						pulumi.String("mkdir -p /www && env > /www/index.html && httpd -f -p 8080 -h /www"),
					},
					Ports: compose.ServicePortConfigArray{
						compose.ServicePortConfigArgs{
							Target: pulumi.Int(8080),
							Mode:   pulumi.StringPtr("ingress"),
						},
					},
					Environment: pulumi.StringMap{
						"LITERAL":      pulumi.String("verbatim"),
						"CONFIG":       pulumi.String("${CONFIG}"), // secret
						"OTHER":        pulumi.String("${CONFIG}"), // secret
						"INTERPOLATED": pulumi.String("prefix${CONFIG}suffix"),
						"EMPTY":        pulumi.String(""), // empty literal
					},
				},
			},
		}, pulumi.Provider(awsProvider), pulumi.DependsOn([]pulumi.Resource{param}))
		if err != nil {
			return err
		}

		ctx.Export("endpoints", proj.Endpoints)
		return nil
	})
}
