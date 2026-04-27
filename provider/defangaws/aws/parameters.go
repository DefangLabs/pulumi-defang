package aws

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ssm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ConfigProvider struct {
	projectName string
	// prefix is the leading namespace segment of every SSM parameter this
	// provider manages (e.g. "/Defang/<proj>/<stack>/<key>"). Set in
	// NewConfigProvider; kept private so the CLI and provider stay in sync.
	prefix  string
	cache   map[string]pulumi.StringOutput
	mu      sync.Mutex
	fetched bool
}

func NewConfigProvider(projectName string) *ConfigProvider {
	return &ConfigProvider{
		prefix:      "Defang", // TODO: customizable prefix
		projectName: projectName,
		cache:       make(map[string]pulumi.StringOutput),
	}
}

func (cp *ConfigProvider) GetConfigValue(
	ctx *pulumi.Context, key string, opts ...pulumi.InvokeOption,
) pulumi.StringOutput {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if !cp.fetched {
		values, err := cp.getParametersByPath(ctx, opts...)
		if err != nil {
			return common.ErrorOutput(errors.Join(&compose.ConfigNotFoundError{Key: key}, err))
		}
		cp.fetched = true
		for k, v := range values {
			// Mark as secret so downstream consumers (env vars, task
			// definitions) don't leak the value into Pulumi state or logs.
			cp.cache[k] = pulumi.ToSecret(pulumi.String(v)).(pulumi.StringOutput)
		}
	}

	if val, ok := cp.cache[key]; ok {
		return val
	}

	return compose.ConfigNotFoundOutput(key)
}

func (cp *ConfigProvider) getParametersByPath(
	ctx *pulumi.Context,
	opts ...pulumi.InvokeOption,
) (map[string]string, error) {
	path := cp.getSecretID(ctx.Stack(), "")
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

func (cp *ConfigProvider) getSecretID(stackName, service string) string {
	// Same as CLI
	return fmt.Sprintf("/%s/%s/%s/%s", cp.prefix, cp.projectName, stackName, service)
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
	id := cp.getSecretID(ctx.Stack(), key)
	return fmt.Sprintf("arn:aws:ssm:%s:%s:parameter%s", region, accountId, id), nil
}
