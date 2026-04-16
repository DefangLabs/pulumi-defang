package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azappconfig"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appconfiguration/armappconfiguration"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/DefangLabs/pulumi-defang/provider/defangazure/azure"
)

// fetchAzureUserConfig reads user-defined config secrets from the Azure App Configuration store
// that the defang CLI manages for this deployment.
//
// The defang CLI stores config values (e.g. from "defang config set POSTGRES_PASSWORD=…")
// under keys with the format "/{prefix}/{project}/{stack}/{name}". The CLI passes the project
// resource group via AZURE_RESOURCE_GROUP; the store name is derived deterministically from
// the RG + subscription ID.
//
// Returns nil, nil when no store exists yet (first deploy with no config values).
func fetchAzureUserConfig(ctx context.Context) (map[string]string, error) {
	if azureSubscription == "" || azureResourceGroup == "" {
		return nil, nil
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("creating Azure credential: %w", err)
	}

	storesClient, err := armappconfiguration.NewConfigurationStoresClient(azureSubscription, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("creating App Configuration management client: %w", err)
	}

	rg := azureResourceGroup
	storeName := azure.AppConfigStoreName(rg, azureSubscription)

	// Retrieve a connection string to access the data plane.
	// The MSI has Contributor on the subscription, so it can call ListKeys.
	var connString string
	keysPager := storesClient.NewListKeysPager(rg, storeName, nil)
	for keysPager.More() && connString == "" {
		page, err := keysPager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing App Configuration access keys: %w", err)
		}
		for _, key := range page.Value {
			if key.ConnectionString == nil {
				continue
			}
			if connString == "" {
				connString = *key.ConnectionString // take any key as fallback
			}
			if key.ReadOnly != nil && *key.ReadOnly {
				connString = *key.ConnectionString // prefer read-only key
				break
			}
		}
	}
	if connString == "" {
		return nil, fmt.Errorf("no access key found for App Configuration store %q", storeName)
	}

	cfgClient, err := azappconfig.NewClientFromConnectionString(connString, nil)
	if err != nil {
		return nil, fmt.Errorf("creating App Configuration data client: %w", err)
	}

	// Build the key prefix matching ByocBaseClient.StackDir(project, ""):
	//   "/{prefix}/{project}/{stack}/"  e.g.  "/Defang/nextjs-postgres/test/"
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

// fetchAzureUserConfigFromKeyVault reads user-defined config secrets from the Azure Key Vault
// that the defang CLI manages for this deployment.
//
// The defang CLI stores config values (from "defang config set KEY=val") as Key Vault secrets.
// The secret name is the StackDir key with slashes replaced by "--" and underscores by "-",
// and the original key is stored in the "original-key" tag.
//
// Returns nil, nil when no vault name is set (backward compatibility).
func fetchAzureUserConfigFromKeyVault(ctx context.Context) (map[string]string, error) {
	if azureKeyVaultName == "" {
		return nil, nil
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("creating Azure credential: %w", err)
	}

	vaultURL := "https://" + azureKeyVaultName + ".vault.azure.net"
	client, err := azsecrets.NewClient(vaultURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("creating Key Vault client: %w", err)
	}

	// Build the expected secret name prefix matching the StackDir convention:
	// "/Defang/project/stack/" -> "Defang--project--stack--"
	var keyPrefix string
	if prefix != "" {
		keyPrefix = prefix
	}
	keyPrefix += "--" + project + "--" + stack + "--"

	result := make(map[string]string)

	// List secrets and filter by prefix.
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
			// Recover the original key name from the tag.
			var originalKey string
			if props.Tags != nil {
				if orig, ok := props.Tags["original-key"]; ok && orig != nil {
					// The tag stores the full StackDir path; extract just the key name.
					parts := strings.Split(*orig, "/")
					originalKey = parts[len(parts)-1]
				}
			}
			if originalKey == "" {
				continue
			}
			// Fetch the actual secret value.
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
