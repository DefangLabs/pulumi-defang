package azure

import (
	"github.com/pulumi/pulumi-azure-native-sdk/app/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const defaultAzureLocation = "eastus"

// SharedInfra holds resources shared across all services in a project.
type SharedInfra struct {
	ResourceGroup  *resources.ResourceGroup
	Environment    *app.ManagedEnvironment
	BuildInfra     *BuildInfra       // nil when no services require image builds
	Networking     *NetworkingResult // nil when no VNet-integrated services are present
	DNS            *DNSResult        // nil when no VNet-integrated services are present
	LLMInfra       *LLMInfra         // nil when no LLM services are present
	ConfigProvider *ConfigProvider   // reads project secrets (set via `defang config set`)
}

// Location reads the Azure location from Pulumi stack config, falling back to the default.
func Location(ctx *pulumi.Context) string {
	cfg := config.New(ctx, "azure-native")
	if l := cfg.Get("location"); l != "" {
		return l
	}
	return defaultAzureLocation
}

// AppConfigStore reads the Azure App Configuration store name and resource group
// from Pulumi stack config keys defang-azure:appConfigStore and
// defang-azure:appConfigResourceGroup.
func AppConfigStore(ctx *pulumi.Context) (storeName, resourceGroupName string) {
	cfg := config.New(ctx, "defang-azure")
	return cfg.Get("appConfigStore"), cfg.Get("appConfigResourceGroup")
}

// ExistingResourceGroup returns the name of an existing Azure resource group to import,
// read from Pulumi stack config defang-azure:resourceGroup. Empty if unset (create new).
func ExistingResourceGroup(ctx *pulumi.Context) string {
	return config.New(ctx, "defang-azure").Get("resourceGroup")
}

// SubscriptionID returns the Azure subscription ID from azure-native:subscriptionId config.
func SubscriptionID(ctx *pulumi.Context) string {
	return config.New(ctx, "azure-native").Get("subscriptionId")
}


