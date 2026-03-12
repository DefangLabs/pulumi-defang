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

// sharedInfra holds resources shared across all services in a project.
type sharedInfra struct {
	resourceGroup *resources.ResourceGroup
	environment   *app.ManagedEnvironment
}

// ContainerAppResult holds the per-service outputs for a Container App.
type ContainerAppResult struct {
	Endpoint pulumi.StringOutput
}

// PostgresResult holds the per-service outputs for a PostgreSQL Flexible Server.
type PostgresResult struct {
	Endpoint pulumi.StringOutput
}

// azureLocation reads the Azure location from Pulumi stack config, falling back to the default.
func azureLocation(ctx *pulumi.Context) string {
	cfg := config.New(ctx, "azure-native")
	if l := cfg.Get("location"); l != "" {
		return l
	}
	return defaultAzureLocation
}

// Build creates all Azure resources for the project.
// The Azure provider must be passed via the parent chain (pulumi.Providers on the parent component).
func Build(ctx *pulumi.Context, projectName string, args common.BuildArgs, parentOpt pulumi.ResourceOption) (*common.BuildResult, error) {
	location := azureLocation(ctx)
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

	infra := &sharedInfra{
		resourceGroup: rg,
		environment:   env,
	}

	recipe := LoadRecipe(ctx)
	endpoints := pulumi.StringMap{}

	for svcName, svc := range args.Services {
		comp := &serviceComponent{}

		if svc.Postgres != nil {
			// Managed Postgres → Flexible Server
			if err := ctx.RegisterComponentResource("defang-azure:index:AzurePostgres", svcName, comp, opts[0]); err != nil {
				return nil, fmt.Errorf("registering Azure Postgres component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

			pgResult, err := createPostgresFlexible(ctx, svcName, svc, infra, recipe, svcOpts...)
			if err != nil {
				return nil, fmt.Errorf("creating PostgreSQL for %s: %w", svcName, err)
			}
			endpoints[svcName] = pulumi.Sprintf("%s:5432", pgResult.server.FullyQualifiedDomainName)
		} else {
			// Container service → Container App
			if err := ctx.RegisterComponentResource("defang-azure:index:AzureContainerApp", svcName, comp, opts[0]); err != nil {
				return nil, fmt.Errorf("registering Container App component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

			caResult, err := createContainerApp(ctx, svcName, svc, infra, recipe, svcOpts...)
			if err != nil {
				return nil, fmt.Errorf("creating Container App %s: %w", svcName, err)
			}
			endpoints[svcName] = caResult.app.LatestRevisionFqdn.ApplyT(func(fqdn string) string {
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

// BuildStandaloneContainerApp creates Azure resources for a single standalone Container App.
// The Azure provider must be passed via opts (pulumi.Providers on the parent component).
func BuildStandaloneContainerApp(ctx *pulumi.Context, serviceName string, svc common.ServiceConfig, opts ...pulumi.ResourceOption) (*ContainerAppResult, error) {
	location := azureLocation(ctx)

	rg, err := resources.NewResourceGroup(ctx, serviceName+"-rg", &resources.ResourceGroupArgs{
		Location: pulumi.String(location),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating resource group: %w", err)
	}

	env, err := app.NewManagedEnvironment(ctx, serviceName+"-env", &app.ManagedEnvironmentArgs{
		ResourceGroupName: rg.Name,
		Location:          pulumi.String(location),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating managed environment: %w", err)
	}

	infra := &sharedInfra{resourceGroup: rg, environment: env}
	recipe := LoadRecipe(ctx)

	caResult, err := createContainerApp(ctx, serviceName, svc, infra, recipe, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Container App %s: %w", serviceName, err)
	}

	endpoint := caResult.app.LatestRevisionFqdn.ApplyT(func(fqdn *string) string {
		if fqdn != nil && *fqdn != "" {
			return fmt.Sprintf("https://%s", *fqdn)
		}
		return ""
	}).(pulumi.StringOutput)

	return &ContainerAppResult{Endpoint: endpoint}, nil
}

// BuildStandalonePostgres creates Azure resources for a single standalone PostgreSQL Flexible Server.
// The Azure provider must be passed via opts (pulumi.Providers on the parent component).
func BuildStandalonePostgres(ctx *pulumi.Context, serviceName string, svc common.ServiceConfig, opts ...pulumi.ResourceOption) (*PostgresResult, error) {
	location := azureLocation(ctx)

	rg, err := resources.NewResourceGroup(ctx, serviceName+"-rg", &resources.ResourceGroupArgs{
		Location: pulumi.String(location),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating resource group: %w", err)
	}

	// Postgres doesn't need Container Apps environment
	infra := &sharedInfra{resourceGroup: rg}
	recipe := LoadRecipe(ctx)

	pgResult, err := createPostgresFlexible(ctx, serviceName, svc, infra, recipe, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating PostgreSQL for %s: %w", serviceName, err)
	}

	return &PostgresResult{
		Endpoint: pulumi.Sprintf("%s:5432", pgResult.server.FullyQualifiedDomainName),
	}, nil
}
