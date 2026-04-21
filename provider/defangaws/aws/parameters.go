package aws

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
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

func (cp *ConfigProvider) GetConfigValue(
	ctx *pulumi.Context, key string, opts ...pulumi.InvokeOption,
) pulumi.StringOutput {
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

	return compose.ConfigNotFoundOutput(key)
}

func getParametersByPath(
	ctx *pulumi.Context,
	projectName string,
	opts ...pulumi.InvokeOption,
) (map[string]string, error) {
	path := getSecretID(projectName, ctx.Stack(), "")
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

func getSecretID(projectName, stackName, service string) string {
	// Same as CLI
	return fmt.Sprintf("/Defang/%s/%s/%s", projectName, stackName, service) // TODO: customizable prefix
}

// GetSecretRef returns the full SSM parameter ARN for a config key, so ECS can
// resolve the secret at task startup via the Secrets field (valueFrom).
func (cp *ConfigProvider) GetSecretRef(ctx *pulumi.Context, key string, opts ...pulumi.InvokeOption) (string, error) {
	region, err := getCallerRegion(ctx, opts...)
	if err != nil {
		return "", fmt.Errorf("getting region for secret ARN: %w", err)
	}
	accountId, err := getCallerAccountId(ctx, opts...)
	if err != nil {
		return "", fmt.Errorf("getting account ID for secret ARN: %w", err)
	}
	id := getSecretID(cp.projectName, ctx.Stack(), key)
	return fmt.Sprintf("arn:aws:ssm:%s:%s:parameter%s", region, accountId, id), nil
}
