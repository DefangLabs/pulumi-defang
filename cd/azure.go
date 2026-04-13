package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azappconfig"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appconfiguration/armappconfiguration"
)

// cdResourceGroup returns the resource group name where the defang CD infrastructure lives.
// The naming convention (from ByocBaseClient / ContainerInstance.SetLocation) is:
//
//	"defang-cd" + location   e.g. "defang-cdwestus3"
func cdResourceGroup() string {
	return "defang-cd" + azureLocation
}

// fetchAzureUserConfig reads user-defined config secrets from the Azure App Configuration store
// that the defang CLI manages for this deployment.
//
// The defang CLI stores config values (e.g. from "defang config set POSTGRES_PASSWORD=…")
// under keys with the format "/{prefix}/{project}/{stack}/{name}" in a store whose name starts
// with "defangcfg" inside the CD resource group ("defang-cd{location}").
//
// Returns nil, nil when no store exists yet (first deploy with no config values).
func fetchAzureUserConfig(ctx context.Context) (map[string]string, error) {
	if azureSubscription == "" || azureLocation == "" {
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

	rg := cdResourceGroup()

	// Find the App Configuration store in the CD resource group.
	var storeName string
	storePager := storesClient.NewListByResourceGroupPager(rg, nil)
	for storePager.More() && storeName == "" {
		page, err := storePager.NextPage(ctx)
		if err != nil {
			// Resource group may not exist on a brand-new deployment.
			log.Printf("App Configuration store not found in %q (resource group may not exist yet): %v", rg, err)
			return nil, nil
		}
		for _, store := range page.Value {
			if store.Name != nil && strings.HasPrefix(*store.Name, "defangcfg") {
				storeName = *store.Name
				break
			}
		}
	}
	if storeName == "" {
		return nil, nil // No store yet; nothing to inject.
	}

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
