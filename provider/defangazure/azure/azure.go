package azure

import (
	"github.com/pulumi/pulumi-azure-native-sdk/app/v2"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const defaultAzureLocation = "eastus"

// SharedInfra holds resources shared across all services in a project.
type SharedInfra struct {
	ResourceGroup *resources.ResourceGroup
	Environment   *app.ManagedEnvironment
	BuildInfra    *BuildInfra // nil when no services require image builds
}

// Location reads the Azure location from Pulumi stack config, falling back to the default.
func Location(ctx *pulumi.Context) string {
	cfg := config.New(ctx, "azure-native")
	if l := cfg.Get("location"); l != "" {
		return l
	}
	return defaultAzureLocation
}


