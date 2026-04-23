package main

import (
	"fmt"

	defanggcp "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-gcp"
	gcpcompose "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-gcp/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v8/go/gcp"
	"github.com/pulumi/pulumi-gcp/sdk/v8/go/gcp/secretmanager"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func runGcp(ctx *pulumi.Context) error {
	gcpCfg := config.New(ctx, "gcp")
	gcpProvider, err := gcp.NewProvider(ctx, "gcp", &gcp.ProviderArgs{
		Project: pulumi.String(gcpCfg.Require("project")),
		Region:  pulumi.String(gcpCfg.Require("region")),
	})
	if err != nil {
		return err
	}

	// Secret ID matches defang-gcp's ConfigProvider.getSecretID:
	// Defang_<project>_<stack>_<KEY>. LookupSecretVersion resolves this at
	// Construct time.
	secretID := fmt.Sprintf("Defang_%s_%s_%s", defangProjectName, ctx.Stack(), configKey)
	secret, err := secretmanager.NewSecret(ctx, "config", &secretmanager.SecretArgs{
		SecretId: pulumi.String(secretID),
		Replication: &secretmanager.SecretReplicationArgs{
			Auto: &secretmanager.SecretReplicationAutoArgs{},
		},
	}, pulumi.Provider(gcpProvider))
	if err != nil {
		return err
	}

	version, err := secretmanager.NewSecretVersion(ctx, "config-v1", &secretmanager.SecretVersionArgs{
		Secret:     secret.ID(),
		SecretData: pulumi.String(configValue),
	}, pulumi.Provider(gcpProvider))
	if err != nil {
		return err
	}

	proj, err := defanggcp.NewProject(ctx, defangProjectName, &defanggcp.ProjectArgs{
		Services: gcpcompose.ServiceConfigMap{
			"web": gcpcompose.ServiceConfigArgs{
				Image:   pulumi.StringPtr("busybox"),
				Command: testCommand,
				Ports: gcpcompose.ServicePortConfigArray{
					gcpcompose.ServicePortConfigArgs{
						Target: pulumi.Int(8080),
						Mode:   pulumi.StringPtr("ingress"),
					},
				},
				Environment: testEnvironment,
			},
		},
	}, pulumi.Provider(gcpProvider), pulumi.DependsOn([]pulumi.Resource{version}))
	if err != nil {
		return err
	}

	ctx.Export("gcp-endpoints", proj.Endpoints)
	return nil
}
