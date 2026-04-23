package azure

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// FetchUserConfig reads user-defined config values for this project/stack from
// the project's Azure Key Vault. Returns an empty map when the vault is unset
// or contains no matching secrets (first deploy) — not an error.
//
// The vault name is derived deterministically from (subscription, resource group)
// via KeyVaultName; project/stack/prefix come from the running context.
func FetchUserConfig(ctx *pulumi.Context) (map[string]string, error) {
	vaultName := KeyVaultName(ctx)
	if vaultName == "" {
		return map[string]string{}, nil
	}

	prefix := config.New(ctx, "defang").Get("prefix")
	vals, err := fetchFromKeyVault(ctx.Context(), vaultName, prefix, ctx.Project(), ctx.Stack())
	if err != nil {
		return nil, fmt.Errorf("reading user config from Key Vault: %w", err)
	}
	return vals, nil
}

// fetchFromKeyVault lists secrets in the vault whose name begins with the
// project/stack prefix and reads their values. The secret's original key name
// is recovered from the "original-key" tag (the defang CLI stores the full
// StackDir path there).
func fetchFromKeyVault(ctx context.Context, vaultName, prefix, project, stack string) (map[string]string, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("creating Azure credential: %w", err)
	}

	vaultURL := "https://" + vaultName + ".vault.azure.net"
	client, err := azsecrets.NewClient(vaultURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("creating Key Vault client: %w", err)
	}

	// Secret names follow the StackDir convention with slashes replaced by "--":
	//   "/Defang/project/stack/KEY" -> "Defang--project--stack--<sanitized-key>"
	var keyPrefix string
	if prefix != "" {
		keyPrefix = prefix
	}
	keyPrefix += "--" + project + "--" + stack + "--"

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
