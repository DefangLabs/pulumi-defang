package main

import (
	defangaws "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws"
	"github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws/compose"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
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

		proj, err := defangaws.NewProject(ctx, "config-example", &defangaws.ProjectArgs{
			Services: compose.ServiceConfigMap{
				"web": compose.ServiceConfigArgs{
					Image: pulumi.StringPtr("busybox"),
					Command: pulumi.StringArray{
						pulumi.String("sh"), pulumi.String("-c"),
						pulumi.String("mkdir -p /www && echo $GREETING > /www/index.html && httpd -f -p 8080 -h /www"),
					},
					Ports: compose.ServicePortConfigArray{
						compose.ServicePortConfigArgs{
							Target: pulumi.Int(8080),
							Mode:   pulumi.StringPtr("ingress"),
						},
					},
					Environment: pulumi.StringMap{
						// Resolved at deploy time by the ConfigProvider from SSM
						"GREETING": pulumi.String("${GREETING}"),
					},
				},
			},
		}, pulumi.Provider(awsProvider))
		if err != nil {
			return err
		}

		ctx.Export("endpoints", proj.Endpoints)
		return nil
	})
}
