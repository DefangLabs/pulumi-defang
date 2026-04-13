package azure

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type ConfigProvider struct {
	projectName string
}

func NewConfigProvider(projectName string) *ConfigProvider {
	return &ConfigProvider{projectName: projectName}
}

// GetConfig reads a secret config value from the Pulumi stack config.
// The defang CLI stores user-defined config (e.g. POSTGRES_PASSWORD) as Pulumi
// stack config secrets under the key name directly.
func (p *ConfigProvider) GetConfig(ctx *pulumi.Context, key string, opts ...pulumi.InvokeOption) pulumi.StringOutput {
	if ctx.DryRun() {
		return pulumi.Sprintf("dry-run-%s", key).ToStringOutput()
	}
	cfg := config.New(ctx, "")
	return cfg.GetSecret(key)
}
