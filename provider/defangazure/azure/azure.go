package azure

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/pulumi/pulumi-azure-native-sdk/app/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const defaultAzureLocation = "eastus"

// storeNameSuffixLen must match appcfg.storeNameSuffixLen in the defang CLI.
const storeNameSuffixLen = 12

// AppConfigStoreName returns the deterministic App Configuration store name for the
// given project resource group in the given subscription. Mirrors appcfg.StoreName in
// the defang CLI so both sides derive the same name without needing to pass it around.
//
// The 12-hex-char suffix is sha256(subscription_id + "|" + resource_group), which keeps
// the name globally unique even when the resource-group portion must be truncated to
// fit Azure's 50-char limit on App Configuration store names.
func AppConfigStoreName(resourceGroupName, subscriptionID string) string {
	h := sha256.Sum256([]byte(subscriptionID + "|" + resourceGroupName))
	suffix := hex.EncodeToString(h[:])[:storeNameSuffixLen]

	name := resourceGroupName + "-" + suffix
	if len(name) > 50 {
		name = name[:50-1-len(suffix)] + "-" + suffix
	}
	return name
}

// SharedInfra holds resources shared across all services in a project.
type SharedInfra struct {
	ResourceGroup  *resources.ResourceGroup
	Environment    *app.ManagedEnvironment
	BuildInfra     *BuildInfra       // nil when no services require image builds
	Networking     *NetworkingResult // nil when no VNet-integrated services are present
	DNS            *DNSResult        // nil when no VNet-integrated services are present
	LLMInfra       *LLMInfra         // nil when no LLM services are present
	ConfigProvider *ConfigProvider   // reads project secrets (set via `defang config set`)
	KeyVaultURL    string            // Key Vault URL for secret references (empty if no vault)
	KeyVaultIdentityID pulumi.StringOutput // user-assigned identity for KV access (zero if no vault)
}

// KeyVaultName returns the Key Vault name from Pulumi stack config, or empty.
func KeyVaultName(ctx *pulumi.Context) string {
	return config.New(ctx, "defang-azure").Get("keyVaultName")
}

// Location reads the Azure location from Pulumi stack config, falling back to the default.
func Location(ctx *pulumi.Context) string {
	cfg := config.New(ctx, "azure-native")
	if l := cfg.Get("location"); l != "" {
		return l
	}
	return defaultAzureLocation
}

// AppConfigStore returns the project's App Configuration store name and resource group.
// The resource group is read from Pulumi stack config (defang-azure:resourceGroup) and
// the store name is derived deterministically from it plus the subscription ID via
// AppConfigStoreName — the same derivation the defang CLI uses when creating the store.
func AppConfigStore(ctx *pulumi.Context) (storeName, resourceGroupName string) {
	resourceGroupName = config.New(ctx, "defang-azure").Get("resourceGroup")
	if resourceGroupName == "" {
		return "", ""
	}
	return AppConfigStoreName(resourceGroupName, SubscriptionID(ctx)), resourceGroupName
}

// ExistingResourceGroup returns the name of the project's Azure resource group,
// read from Pulumi stack config defang-azure:resourceGroup. This RG holds both the
// project's deployed resources and its App Configuration store. Empty if unset.
func ExistingResourceGroup(ctx *pulumi.Context) string {
	return config.New(ctx, "defang-azure").Get("resourceGroup")
}

// SubscriptionID returns the Azure subscription ID from azure-native:subscriptionId config.
func SubscriptionID(ctx *pulumi.Context) string {
	return config.New(ctx, "azure-native").Get("subscriptionId")
}


