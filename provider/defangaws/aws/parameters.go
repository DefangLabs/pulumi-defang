package aws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/shared"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ssm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func getConfigOrEnvValue(ctx *pulumi.Context, s shared.ServiceInput, projectName string, key string, opts ...pulumi.ResourceOption) pulumi.StringOutput {
	if s.Environment == nil {
		return pulumi.StringOutput{}
	}

	value, exists := s.Environment[key]
	if !exists {
		return pulumi.StringOutput{}
	}

	if value == nil {
		// If the value is explicitly set to nil, treat it as a sensitive config reference
		return getParameterValue(ctx, projectName, key)
	}

	v := *value
	if v == "" {
		return pulumi.StringOutput{}
	}

	return pulumi.Sprintf("%s", v).ApplyT(func(v string) pulumi.StringOutput {
		return interpolateEnvironmentVariable(ctx, projectName, v)
	}).(pulumi.StringOutput)
}

func interpolateEnvironmentVariable(ctx *pulumi.Context, projectName, value string) pulumi.StringOutput {
	parsed := shared.ParseInterpolatedString(value)

	parts := make([]pulumi.StringOutput, len(parsed))
	for i, match := range parsed {
		if !match.IsVar {
			parts[i] = pulumi.String(match.Literal).ToStringOutput()
		} else {
			parts[i] = getParameterValue(ctx, projectName, match.Variable)
		}
	}

	// Fold over parts, joining with ApplyT since pulumi.Concat doesn't exist in Go
	result := parts[0]
	for _, part := range parts[1:] {
		p := part // capture loop var
		result = result.ApplyT(func(acc string) pulumi.StringOutput {
			return p.ApplyT(func(s string) string {
				return acc + s
			}).(pulumi.StringOutput)
		}).(pulumi.StringOutput)
	}
	return result
}

func getParameterValue(ctx *pulumi.Context, projectName string, sourceName string) pulumi.StringOutput {
	// In dry-run mode, return a placeholder value
	if ctx.DryRun() {
		return pulumi.Sprintf("dry-run-%s", sourceName).ApplyT(func(v string) string {
			return v
		}).(pulumi.StringOutput)
	}

	gpr := ssm.GetParametersByPathOutput(ctx, ssm.GetParametersByPathOutputArgs{
		Path:           pulumi.String(getSecretID(sourceName, projectName)),
		WithDecryption: pulumi.Bool(true),
	})

	return pulumi.StringOutput(gpr.Values())
}

func getSecretID(sourceName, projectName string) string {
	return fmt.Sprintf("/%s/%s", projectName, sourceName)
}
