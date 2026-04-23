package main

import (
	defangaws "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws"
	awscompose "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws/compose"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ssm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func runAws(ctx *pulumi.Context) error {
	awsProvider, err := aws.NewProvider(ctx, "aws", &aws.ProviderArgs{
		Region: pulumi.String(config.New(ctx, "aws").Require("region")),
	})
	if err != nil {
		return err
	}

	// SSM parameter path matches defang-aws's ConfigProvider.getSecretID:
	// /Defang/<project>/<stack>/<KEY>. GetParametersByPath reads everything
	// under /Defang/<project>/<stack>/ at Construct time.
	param, err := ssm.NewParameter(ctx, "config", &ssm.ParameterArgs{
		Type:  pulumi.String("String"),
		Name:  pulumi.Sprintf("/Defang/%s/%s/%s", defangProjectName, ctx.Stack(), configKey),
		Value: pulumi.String(configValue),
	}, pulumi.Provider(awsProvider))
	if err != nil {
		return err
	}

	proj, err := defangaws.NewProject(ctx, defangProjectName, &defangaws.ProjectArgs{
		Services: awscompose.ServiceConfigMap{
			"web": awscompose.ServiceConfigArgs{
				Image:   pulumi.StringPtr("busybox"),
				Command: testCommand,
				Ports: awscompose.ServicePortConfigArray{
					awscompose.ServicePortConfigArgs{
						Target: pulumi.Int(8080),
						Mode:   pulumi.StringPtr("ingress"),
					},
				},
				Environment: testEnvironment,
			},
		},
	}, pulumi.Provider(awsProvider), pulumi.DependsOn([]pulumi.Resource{param}))
	if err != nil {
		return err
	}

	ctx.Export("aws-endpoints", proj.Endpoints)
	return nil
}
