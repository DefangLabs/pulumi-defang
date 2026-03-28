package aws

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ssm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ConfigProvider struct {
	projectName string
	cache       map[string]pulumi.StringOutput
	mu          sync.Mutex
	fetched     bool
}

func NewConfigProvider(projectName string) *ConfigProvider {
	return &ConfigProvider{projectName: projectName, cache: make(map[string]pulumi.StringOutput)}
}

func (cp *ConfigProvider) GetConfig(ctx *pulumi.Context, key string, opts ...pulumi.InvokeOption) pulumi.StringOutput {
	// In dry-run mode, return a placeholder value
	if ctx.DryRun() {
		return pulumi.Sprintf("dry-run-%s", key).ToStringOutput()
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()

	if !cp.fetched {
		values, err := getParametersByPath(ctx, cp.projectName, opts...)
		if err == nil {
			cp.fetched = true
			for k, v := range values {
				cp.cache[k] = pulumi.String(v).ToStringOutput()
			}
		}
	}

	if val, ok := cp.cache[key]; ok {
		return val
	}

	return pulumi.StringOutput{}
}

func getParametersByPath(
	ctx *pulumi.Context,
	projectName string,
	opts ...pulumi.InvokeOption,
) (map[string]string, error) {
	path := getSecretPath(projectName, ctx.Stack())
	withDecryption := true

	gpr, err := ssm.GetParametersByPath(ctx, &ssm.GetParametersByPathArgs{
		Path:           path,
		WithDecryption: &withDecryption,
	}, opts...)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for i, name := range gpr.Names {
		baseName := filepath.Base(name)
		result[baseName] = gpr.Values[i]
	}

	return result, nil
}

func getSecretPath(projectName, stackName string) string {
	return fmt.Sprintf("/Defang/%s/%s/", projectName, stackName)
}
