package gcp

import (
	"strings"
	"sync"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/secretmanager"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ConfigProvider struct {
	projectName string
	// prefix is the leading namespace segment of every Secret Manager secret
	// this provider manages (e.g. "Defang_<proj>_<stack>_<key>"). Set in
	// NewConfigProvider; kept private so the CLI and provider stay in sync.
	prefix string
	cache  map[string]pulumi.StringOutput
	mu     sync.Mutex
}

func NewConfigProvider(projectName string) *ConfigProvider {
	return &ConfigProvider{
		prefix:      "Defang", // TODO: customizable prefix
		projectName: projectName,
		cache:       make(map[string]pulumi.StringOutput),
	}
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

	secretId := p.getSecretID(ctx.Stack(), key)
	sv, err := secretmanager.LookupSecretVersion(ctx, &secretmanager.LookupSecretVersionArgs{
		Secret: secretId,
	}, opts...)
	if err != nil || sv == nil {
		return compose.ConfigNotFoundOutput(key)
	}

	// Mark as secret so downstream consumers (env vars, Cloud Run services)
	// don't leak the value into Pulumi state or logs.
	out := pulumi.ToSecret(pulumi.String(sv.SecretData)).(pulumi.StringOutput)
	p.cache[key] = out
	return out
}

// GetSecretRef returns the GCP Secret Manager secret ID so Cloud Run can
// reference it via ValueSource.SecretKeyRef without decrypting.
func (p *ConfigProvider) GetSecretRef(
	ctx *pulumi.Context, key string, _ ...pulumi.InvokeOption,
) (string, error) {
	return p.getSecretID(ctx.Stack(), key), nil
}

// getSecretID returns the Secret Manager secret ID for a config key.
// Matches the CLI's naming convention: <prefix>_<project>_<stack>_<key>.
func (p *ConfigProvider) getSecretID(stack, key string) string {
	return strings.Join([]string{p.prefix, p.projectName, stack, key}, "_")
}
