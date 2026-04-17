package azure

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azappconfig"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appconfiguration/armappconfiguration"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// FetchUserConfig reads user-defined config values for this project/stack from
// Azure. Tries Key Vault (new path) first, falls back to App Configuration
// (legacy path). Returns an empty map on first deploy when neither store has
// been populated yet — not an error.
//
// The vault name and legacy App Config store are derived from Pulumi stack
// config (defang-azure:keyVaultName, defang-azure:resourceGroup + the
// subscription ID). Project/stack/prefix come from the running context.
func FetchUserConfig(ctx *pulumi.Context) (map[string]string, error) {
	prefix := config.New(ctx, "defang").Get("prefix")
	project := ctx.Project()
	stack := ctx.Stack()

	goCtx := ctx.Context()

	if vaultName := KeyVaultName(ctx); vaultName != "" {
		vals, err := fetchFromKeyVault(goCtx, vaultName, prefix, project, stack)
		if err != nil {
			return nil, fmt.Errorf("reading user config from Key Vault: %w", err)
		}
		if len(vals) > 0 {
			return vals, nil
		}
	}

	storeName, storeRG := AppConfigStore(ctx)
	if storeName == "" {
		return map[string]string{}, nil
	}
	subscriptionID := SubscriptionID(ctx)
	vals, err := fetchFromAppConfig(goCtx, subscriptionID, storeRG, storeName, prefix, project, stack)
	if err != nil {
		return nil, fmt.Errorf("reading user config from App Configuration: %w", err)
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

// fetchFromAppConfig reads user config from the legacy Azure App Configuration
// store. The defang CLI stored config values (from "defang config set KEY=val")
// under keys "/{prefix}/{project}/{stack}/{name}".
func fetchFromAppConfig(ctx context.Context, subscriptionID, resourceGroup, storeName, prefix, project, stack string) (map[string]string, error) {
	if subscriptionID == "" || resourceGroup == "" || storeName == "" {
		return map[string]string{}, nil
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("creating Azure credential: %w", err)
	}

	storesClient, err := armappconfiguration.NewConfigurationStoresClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("creating App Configuration management client: %w", err)
	}

	var connString string
	keysPager := storesClient.NewListKeysPager(resourceGroup, storeName, nil)
	for keysPager.More() && connString == "" {
		page, err := keysPager.NextPage(ctx)
		if err != nil {
			// Store likely doesn't exist yet on first deploy — not an error.
			return map[string]string{}, nil
		}
		for _, key := range page.Value {
			if key.ConnectionString == nil {
				continue
			}
			if connString == "" {
				connString = *key.ConnectionString
			}
			if key.ReadOnly != nil && *key.ReadOnly {
				connString = *key.ConnectionString
				break
			}
		}
	}
	if connString == "" {
		return map[string]string{}, nil
	}

	cfgClient, err := azappconfig.NewClientFromConnectionString(connString, nil)
	if err != nil {
		return nil, fmt.Errorf("creating App Configuration data client: %w", err)
	}

	var keyPrefix string
	if prefix != "" {
		keyPrefix = "/" + prefix
	}
	keyPrefix += "/" + project + "/" + stack + "/"

	filter := keyPrefix + "*"
	settingsPager := cfgClient.NewListSettingsPager(azappconfig.SettingSelector{
		KeyFilter: &filter,
	}, nil)

	result := make(map[string]string)
	for settingsPager.More() {
		page, err := settingsPager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing App Configuration settings: %w", err)
		}
		for _, setting := range page.Settings {
			if setting.Key == nil || setting.Value == nil {
				continue
			}
			name := strings.TrimPrefix(*setting.Key, keyPrefix)
			if name != "" {
				result[name] = *setting.Value
			}
		}
	}

	return result, nil
}
