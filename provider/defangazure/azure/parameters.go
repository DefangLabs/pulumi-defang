package azure

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ErrNoKeyVaultConfigured is returned by GetSecretRef when the ConfigProvider
// was constructed without a keyVaultURL. Callers must gate on a non-empty vault
// URL before asking for a secret reference.
var ErrNoKeyVaultConfigured = errors.New("no Key Vault configured")

// ConfigProvider exposes user-defined config values (set via `defang config set`)
// to the Pulumi program. Values are fetched lazily from the project's Azure
// Key Vault on the first GetConfigValue call; subsequent calls are served from
// an in-memory cache.
type ConfigProvider struct {
	projectName string
	// prefix is the leading namespace segment of every Key Vault secret name
	// this provider manages (e.g. "Defang--<proj>--<stack>--<key>"). Set in
	// NewConfigProvider; kept private so the CLI and provider stay in sync.
	prefix string
	// keyVaultURL is the vault's base URL ("https://<vault>.vault.azure.net"),
	// used both to locate the vault for the fetch and to assemble ready-to-use
	// secret references in GetSecretRef. Empty when no vault is configured, in
	// which case the fetch is skipped and GetSecretRef must not be called.
	keyVaultURL string
	cache       map[string]pulumi.StringOutput
	mu          sync.Mutex
	fetched     bool
}

func NewConfigProvider(projectName, keyVaultURL string) *ConfigProvider {
	return &ConfigProvider{
		prefix:      "Defang", // TODO: customizable prefix
		projectName: projectName,
		keyVaultURL: strings.TrimRight(keyVaultURL, "/"),
		cache:       make(map[string]pulumi.StringOutput),
	}
}

// GetConfigValue returns a user-defined config value as a pulumi.StringOutput marked
// secret. Unknown keys produce a StringOutput that fails the deployment with a
// compose.ConfigNotFoundError — same contract as AWS/GCP. Never returns a
// zero-value pulumi.StringOutput{}, which would cause a nil-pointer dereference
// inside Pulumi's reflection walk.
//
// On first call, lazily fetches all project/stack secrets from Key Vault in
// one pager round-trip, matching the AWS provider's GetParametersByPath pattern.
func (p *ConfigProvider) GetConfigValue(ctx *pulumi.Context, key string, _ ...pulumi.InvokeOption) pulumi.StringOutput {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.fetched && p.keyVaultURL != "" {
		values, err := p.fetchFromKeyVault(ctx.Context(), ctx.Project(), ctx.Stack())
		if err != nil {
			return common.ErrorOutput(errors.Join(&compose.ConfigNotFoundError{Key: key}, err))
		}
		p.fetched = true
		for k, v := range values {
			p.cache[k] = pulumi.ToSecret(pulumi.String(v)).(pulumi.StringOutput)
		}
	}

	if val, ok := p.cache[key]; ok {
		return val
	}

	return compose.ConfigNotFoundOutput(key)
}

// GetSecretRef returns a ready-to-use Key Vault secret URL of the form
// "https://<vault>.vault.azure.net/secrets/<secret-name>" — callable directly
// from Container Apps' KeyVaultUrl field. Returns an error if the provider
// was constructed without a keyVaultURL (no vault configured on the project).
func (p *ConfigProvider) GetSecretRef(
	ctx *pulumi.Context, key string, _ ...pulumi.InvokeOption,
) (string, error) {
	if p.keyVaultURL == "" {
		return "", fmt.Errorf("cannot build secret ref for %q: %w", key, ErrNoKeyVaultConfigured)
	}
	// Mirror the CLI's ToSecretName convention:
	// "{prefix}/{project}/{stack}/{KEY}" with / -> -- and _ -> -
	safeKey := strings.ReplaceAll(key, "_", "-")
	secretName := p.prefix + "--" + ctx.Project() + "--" + ctx.Stack() + "--" + safeKey
	return p.keyVaultURL + "/secrets/" + secretName, nil
}

// fetchFromKeyVault lists secrets in the vault whose name begins with the
// project/stack prefix and reads their values. The secret's original key name
// is recovered from the "original-key" tag (the defang CLI stores the full
// StackDir path there).
//
// Uses the raw azsecrets data-plane client rather than Pulumi's
// keyvault.LookupSecret invoke because the latter hits ARM's management plane,
// which per the SDK comment on SecretProperties.Value will never return the
// secret value. Authentication flows through the Azure credential chain,
// independent of any Pulumi provider.
func (p *ConfigProvider) fetchFromKeyVault(ctx context.Context, project, stack string) (map[string]string, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("creating Azure credential: %w", err)
	}

	client, err := azsecrets.NewClient(p.keyVaultURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("creating Key Vault client: %w", err)
	}

	// Secret names follow the StackDir convention with slashes replaced by "--":
	//   "/<prefix>/<project>/<stack>/<KEY>" -> "<prefix>--<project>--<stack>--<sanitized-key>"
	keyPrefix := p.prefix + "--" + project + "--" + stack + "--"

	result := make(map[string]string)

	pager := client.NewListSecretPropertiesPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing Key Vault secrets: %w", err)
		}
		for _, props := range page.Value {
			if props.ID == nil {
				continue
			}
			secretName := props.ID.Name()
			if !strings.HasPrefix(secretName, keyPrefix) {
				continue
			}
			var originalKey string
			if props.Tags != nil {
				if orig, ok := props.Tags["original-key"]; ok && orig != nil {
					parts := strings.Split(*orig, "/")
					originalKey = parts[len(parts)-1]
				}
			}
			if originalKey == "" {
				continue
			}
			resp, err := client.GetSecret(ctx, secretName, "", nil)
			if err != nil {
				return nil, fmt.Errorf("getting Key Vault secret %s: %w", secretName, err)
			}
			if resp.Value != nil {
				result[originalKey] = *resp.Value
			}
		}
	}

	return result, nil
}
