package azure

import (
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/app/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const defaultAzureLocation = "eastus"

// SharedInfra holds resources shared across all services in a project.
type SharedInfra struct {
	ResourceGroup      *resources.ResourceGroup
	Environment        *app.ManagedEnvironment
	BuildInfra         *BuildInfra            // nil when no services require image builds
	Networking         *NetworkingResult      // nil when no VNet-integrated services are present
	DNS                *DNSResult             // nil when no VNet-integrated services are present
	LLMInfra           *LLMInfra              // nil when no LLM services are present
	ConfigProvider     compose.ConfigProvider // reads project secrets (set via `defang config set`)
	KeyVaultURL        string                 // Key Vault URL for secret references (empty if no vault)
	KeyVaultIdentityID pulumi.StringOutput    // user-assigned identity for KV access (zero if no vault)
}

// KeyVaultName returns the Key Vault name from Pulumi stack config, or empty.
func KeyVaultName(ctx *pulumi.Context) string {
	return config.New(ctx, "defang-azure").Get("keyVaultName")
}

// Location reads the Azure location from Pulumi stack config, falling back to the default.
func Location(ctx *pulumi.Context) string {
	if l := config.New(ctx, "azure-native").Get("location"); l != "" {
		return l
	}
	return defaultAzureLocation
}

// ExistingResourceGroup returns the name of the project's Azure resource group,
// read from Pulumi stack config defang-azure:resourceGroup. The provider imports
// this RG instead of creating a new one. Empty if unset.
func ExistingResourceGroup(ctx *pulumi.Context) string {
	return config.New(ctx, "defang-azure").Get("resourceGroup")
}

// SubscriptionID returns the Azure subscription ID from azure-native:subscriptionId config.
func SubscriptionID(ctx *pulumi.Context) string {
	return config.New(ctx, "azure-native").Get("subscriptionId")
}
