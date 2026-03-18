package aws

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ssm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
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
		return pulumi.Sprintf("dry-run-%s", sourceName).ApplyT(func(v string) string {
			return v
		}).(pulumi.StringOutput)
	}

	path := getSecretID("", projectName, ctx.Stack())

	gpr := ssm.GetParametersByPathOutput(ctx, ssm.GetParametersByPathOutputArgs{
		Path:           pulumi.String(path),
		WithDecryption: pulumi.Bool(true),
	})

	return gpr.Names().ApplyT(func(names []string) pulumi.StringOutput {
		return gpr.Values().ApplyT(func(vals []string) pulumi.StringOutput {
			return findValueForName(names, vals, sourceName)
		}).(pulumi.StringOutput)
	}).(pulumi.StringOutput)
}

func getSecretID(sourceName, projectName, stackName string) string {
	return fmt.Sprintf("/Defang/%s/%s/%s", projectName, stackName, sourceName)
}

func findValueForName(names, vals []string, sourceName string) pulumi.StringOutput {
	for i, name := range names {
		if name == getSecretID(sourceName, "", "") {
			return pulumi.String(vals[i]).ToStringOutput()
		}
	}
	return pulumi.String("").ToStringOutput()
}
