package azure

import (
	"strings"
	"sync"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ConfigProvider exposes user-defined config values (set via `defang config set`)
// to the Pulumi program. Values are fetched up front from the project's Azure
// Key Vault (or legacy App Configuration store) by FetchUserConfig and passed in
// via NewConfigProvider — the provider then serves them from an in-memory map
// instead of going through Pulumi stack config.
type ConfigProvider struct {
	projectName string
	values      map[string]string
	cache       map[string]pulumi.StringOutput
	mu          sync.Mutex
}

func NewConfigProvider(projectName string, values map[string]string) *ConfigProvider {
	if values == nil {
		values = map[string]string{}
	}
	return &ConfigProvider{
		projectName: projectName,
		values:      values,
		cache:       make(map[string]pulumi.StringOutput),
	}
}

// GetConfigValue returns a user-defined config value as a pulumi.StringOutput marked
// secret. Unknown keys resolve to "" — same contract as the previous stack-config
// backed implementation. Never returns a zero-value pulumi.StringOutput{}, which
// would cause a nil-pointer dereference inside Pulumi's reflection walk.
func (p *ConfigProvider) GetConfigValue(ctx *pulumi.Context, key string, _ ...pulumi.InvokeOption) pulumi.StringOutput {
	p.mu.Lock()
	defer p.mu.Unlock()

	if val, ok := p.cache[key]; ok {
		return val
	}

	v := p.values[key]
	out := pulumi.ToSecret(pulumi.String(v).ToStringOutput()).(pulumi.StringOutput)
	p.cache[key] = out
	return out
}

// GetSecretRef returns a secret reference for Azure Key Vault.
func (p *ConfigProvider) GetSecretRef(
	ctx *pulumi.Context, key string, _ ...pulumi.InvokeOption,
) (string, error) {
	// Mirror the CLI's ToSecretName convention:
	// "{prefix}/{project}/{stack}/{KEY}" with / -> -- and _ -> -
	safeKey := strings.ReplaceAll(key, "_", "-")
	return "Defang--" + ctx.Project() + "--" + ctx.Stack() + "--" + safeKey, nil // TODO: customizable prefix
}
