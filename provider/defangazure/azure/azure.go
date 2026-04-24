package azure

import (
	"crypto/sha256"
	"encoding/hex"

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

// KeyVaultName returns the deterministic Key Vault name for the given Defang
// Compose project in this stack, derived from (subscription, resource group)
// per the defang CLI convention (see
// defang/src/pkg/clouds/azure/keyvault.VaultName). Empty if the subscription
// ID isn't available.
//
// composeProject is the Defang Compose project name (e.g. "crewai"), which
// may differ from ctx.Project() — a single Pulumi project can host multiple
// Defang Compose projects.
func KeyVaultName(ctx *pulumi.Context, composeProject string) string {
	subID := SubscriptionID(ctx)
	if subID == "" {
		return ""
	}
	rg := ExistingResourceGroup(ctx, composeProject)
	h := sha256.Sum256([]byte(subID + "|" + rg))
	return "kv-" + hex.EncodeToString(h[:])[:8]
}

// KeyVaultResourceGroup returns the name of the resource group that contains
// the user's Key Vault, from Pulumi stack config. Empty if unset, in which
// case the vault is assumed to live in the project's own resource group.
func KeyVaultResourceGroup(ctx *pulumi.Context) string {
	return config.New(ctx, "defang-azure").Get("keyVaultResourceGroup")
}

// Location reads the Azure location from Pulumi stack config, falling back to the default.
func Location(ctx *pulumi.Context) string {
	if l := config.New(ctx, "azure-native").Get("location"); l != "" {
		return l
	}
	return defaultAzureLocation
}

// ExistingResourceGroup returns the deterministic name of the Defang Compose
// project's Azure resource group, derived from (composeProject, stack,
// location) per the defang CLI convention (see
// defang/src/pkg/cli/client/byoc/azure.projectResourceGroupName). The CLI
// creates this RG before invoking the CD task; the provider imports it.
//
// composeProject is the Defang Compose project name (typically from the
// compose file's top-level `name:`), which may differ from ctx.Project() —
// a single Pulumi project can host multiple Defang Compose projects.
func ExistingResourceGroup(ctx *pulumi.Context, composeProject string) string {
	return "defang-" + composeProject + "-" + ctx.Stack() + "-" + Location(ctx)
}

// SubscriptionID returns the Azure subscription ID from azure-native:subscriptionId config.
func SubscriptionID(ctx *pulumi.Context) string {
	return config.New(ctx, "azure-native").Get("subscriptionId")
}
