package compose

import (
	"errors"
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type ConfigProvider interface {
	GetConfigValue(ctx *pulumi.Context, key string, opts ...pulumi.InvokeOption) pulumi.StringOutput
	// GetSecretRef returns a native secret reference (e.g. SSM parameter ARN, GCP
	// Secret Manager ID) so container runtimes resolve the secret at startup instead
	// of receiving the decrypted value in the task definition.
	GetSecretRef(ctx *pulumi.Context, key string, opts ...pulumi.InvokeOption) (string, error)
}

type ConfigNotFoundError struct {
	Key string
}

func (e ConfigNotFoundError) Error() string {
	return fmt.Sprintf("config %q not found", e.Key)
}

// ConfigNotFoundOutput returns a StringOutput that fails the deployment with an
// error indicating the config key was not found. Use this instead of returning
// an empty string when a config value is required but missing.
func ConfigNotFoundOutput(key string) pulumi.StringOutput {
	return pulumi.String(key).ToStringOutput().ApplyT(func(k string) (string, error) {
		return "", &ConfigNotFoundError{Key: k}
	}).(pulumi.StringOutput)
}

type DryRunConfigProvider struct{}

// GetConfigValue returns a dummy config value for dry runs.
func (p *DryRunConfigProvider) GetConfigValue(
	ctx *pulumi.Context, key string, opts ...pulumi.InvokeOption,
) pulumi.StringOutput {
	return pulumi.ToSecret(pulumi.Sprintf("dry-run-%s", key)).(pulumi.StringOutput)
}

// GetSecretRef returns a placeholder secret reference for dry runs.
func (p *DryRunConfigProvider) GetSecretRef(
	_ *pulumi.Context, key string, _ ...pulumi.InvokeOption,
) (string, error) {
	return "dry-run-secret-" + key, nil
}

type PulumiConfigProvider struct{}

// GetConfigValue reads a secret config value from the Pulumi stack config.
func (p *PulumiConfigProvider) GetConfigValue(
	ctx *pulumi.Context, key string, opts ...pulumi.InvokeOption,
) pulumi.StringOutput {
	return config.New(ctx, "").RequireSecret(key)
}

// GetSecretRef is not supported by PulumiConfigProvider.
func (p *PulumiConfigProvider) GetSecretRef(
	_ *pulumi.Context, key string, _ ...pulumi.InvokeOption,
) (string, error) {
	return "", errors.ErrUnsupported
}
