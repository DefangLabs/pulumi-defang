package aws

import (
	"fmt"
	"path"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ssm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

type ConfigProvider struct {
	projectName string
}

func NewConfigProvider(projectName string) *ConfigProvider {
	return &ConfigProvider{projectName: projectName}
}

func (cp *ConfigProvider) GetConfig(ctx *pulumi.Context, key string) pulumi.StringOutput {
	return getParameterValue(ctx, cp.projectName, key)
}

func getParameterValue(ctx *pulumi.Context, projectName string, sourceName string) pulumi.StringOutput {
	// In dry-run mode, return a placeholder value
	if ctx.DryRun() {
		return pulumi.Sprintf("dry-run-%s", sourceName)
	}

	path := getSecretPath(projectName, ctx.Stack())

	gpr := ssm.GetParametersByPathOutput(ctx, ssm.GetParametersByPathOutputArgs{
		Path:           pulumi.String(path),
		WithDecryption: pulumi.Bool(true),
	})

	return pulumi.StringOutput(pulumix.Apply2Err(gpr.Names(), gpr.Values(), func(names, vals []string) (string, error) {
		return findValueForName(names, vals, sourceName)
	}))
}

func getSecretPath(projectName, stackName string) string {
	return fmt.Sprintf("/Defang/%s/%s/", projectName, stackName)
}

func findValueForName(names, vals []string, sourceName string) (string, error) {
	for i, name := range names {
		if sourceName == path.Base(name) {
			return vals[i], nil
		}
	}
	return "", fmt.Errorf("value not found for name: %s", sourceName)
}
