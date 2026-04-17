package azure

import (
	"sync"

	"errors"

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

// HasConfig reports whether the given key was set by the user. Synchronous —
// callers can use it to branch at program time (e.g., deciding whether to emit
// a Container App secret reference vs. a plain environment value).
func (p *ConfigProvider) HasConfig(key string) bool {
	if p == nil {
		return false
	}
	_, ok := p.values[key]
	return ok
}

// GetConfig returns a user-defined config value as a pulumi.StringOutput marked
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
	_ *pulumi.Context, key string, _ ...pulumi.InvokeOption,
) (string, error) {
	// TODO: return Azure Key Vault secret reference
	return "", errors.ErrUnsupported
}
