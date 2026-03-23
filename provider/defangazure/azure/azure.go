package azure

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-azure-native-sdk/app/v2"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const defaultAzureLocation = "eastus"

// serviceComponent is a local component resource used to group per-service resources in the tree.
type serviceComponent struct {
	pulumi.ResourceState
}

// SharedInfra holds resources shared across all services in a project.
type SharedInfra struct {
	ResourceGroup *resources.ResourceGroup
	Environment   *app.ManagedEnvironment
}

// Location reads the Azure location from Pulumi stack config, falling back to the default.
func Location(ctx *pulumi.Context) string {
	cfg := config.New(ctx, "azure-native")
	if l := cfg.Get("location"); l != "" {
		return l
	}
	return defaultAzureLocation
}

// Build creates all Azure resources for the project.
// The Azure provider must be passed via the parent chain (pulumi.Providers on the parent component).
func Build(ctx *pulumi.Context, projectName string, args common.BuildArgs, parentOpt pulumi.ResourceOption) (*common.BuildResult, error) {
	location := Location(ctx)
	opts := []pulumi.ResourceOption{parentOpt}

	// Create resource group
	rg, err := resources.NewResourceGroup(ctx, projectName, &resources.ResourceGroupArgs{
		Location: pulumi.String(location),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating resource group: %w", err)
	}

	// Create Container Apps managed environment
	env, err := app.NewManagedEnvironment(ctx, projectName, &app.ManagedEnvironmentArgs{
		ResourceGroupName: rg.Name,
		Location:          pulumi.String(location),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating managed environment: %w", err)
	}

	infra := &SharedInfra{
		ResourceGroup: rg,
		Environment:   env,
	}

	endpoints := pulumi.StringMap{}

	for svcName, svc := range args.Services {
		comp := &serviceComponent{}

		if svc.Postgres != nil {
			// Managed Postgres → Flexible Server
			if err := ctx.RegisterComponentResource("defang-azure:index:Postgres", svcName, comp, opts...); err != nil {
				return nil, fmt.Errorf("registering Azure Postgres component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

			configProvider := NewConfigProvider(projectName)
			pgResult, err := CreatePostgresFlexible(ctx, configProvider, svcName, svc, infra, svcOpts...)
			if err != nil {
				return nil, fmt.Errorf("creating PostgreSQL for %s: %w", svcName, err)
			}
			endpoints[svcName] = pulumi.Sprintf("%s:5432", pgResult.Server.FullyQualifiedDomainName)
		} else {
			// Container service → Container App
			if err := ctx.RegisterComponentResource("defang-azure:index:AzureContainerApp", svcName, comp, opts...); err != nil {
				return nil, fmt.Errorf("registering Container App component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

			caResult, err := CreateContainerApp(ctx, svcName, svc, infra, svcOpts...)
			if err != nil {
				return nil, fmt.Errorf("creating Container App %s: %w", svcName, err)
			}
			endpoints[svcName] = caResult.App.LatestRevisionFqdn.ApplyT(func(fqdn string) string {
				if fqdn != "" {
					return fmt.Sprintf("https://%s", fqdn)
				}
				return ""
			}).(pulumi.StringOutput)
		}

		if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
			"endpoint": endpoints[svcName],
		}); err != nil {
			return nil, fmt.Errorf("registering outputs for %s: %w", svcName, err)
		}
	}

	return &common.BuildResult{
		Endpoints:       endpoints.ToStringMapOutput(),
		LoadBalancerDNS: pulumi.StringPtr("").ToStringPtrOutput(),
	}, nil
}


