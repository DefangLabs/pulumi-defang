package gcp

import (
	"path"
	"sync"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/secretmanager"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ConfigProvider struct {
	projectName string
	cache map[string]pulumi.StringOutput
	mu    sync.Mutex
}

func NewConfigProvider(projectName string) *ConfigProvider {
	return &ConfigProvider{projectName: projectName, cache: make(map[string]pulumi.StringOutput)}
}

// GetConfigValue fetches the decrypted secret value from GCP Secret Manager.
func (p *ConfigProvider) GetConfigValue(
	ctx *pulumi.Context, key string, opts ...pulumi.InvokeOption,
) pulumi.StringOutput {
	p.mu.Lock()
	defer p.mu.Unlock()

	if val, ok := p.cache[key]; ok {
		return val
	}

	secretId := getSecretID(p.projectName, ctx.Stack(), key)
	sv, err := secretmanager.LookupSecretVersion(ctx, &secretmanager.LookupSecretVersionArgs{
		Secret: secretId,
	}, opts...)
	if err != nil || sv == nil {
		out := compose.ConfigNotFoundOutput(key)
		p.cache[key] = out
		return out
	}

	out := pulumi.String(sv.SecretData).ToStringOutput()
	p.cache[key] = out
	return out
}

// GetSecretRef returns the GCP Secret Manager secret ID so Cloud Run can
// reference it via ValueSource.SecretKeyRef without decrypting.
func (p *ConfigProvider) GetSecretRef(
	ctx *pulumi.Context, key string, _ ...pulumi.InvokeOption,
) (string, error) {
	return getSecretID(p.projectName, ctx.Stack(), key), nil
}

// getSecretID returns the Secret Manager secret ID for a config key.
// Matches the old naming convention: Defang_{project}_{stack}_{key}
func getSecretID(project, stack, key string) string {
	return path.Join("Defang", project, stack, key)
}
