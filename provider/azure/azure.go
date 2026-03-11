package azure

import (
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-azure-native-sdk/app/v2"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v2"
	azurenative "github.com/pulumi/pulumi-azure-native-sdk/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
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

// Build creates all Azure resources for the project.
func Build(ctx *pulumi.Context, projectName string, args common.BuildArgs, parentOpt pulumi.ResourceOption) (*common.BuildResult, error) {
	azProv, location, err := createAzureProvider(ctx, projectName, args.Azure, parentOpt)
	if err != nil {
		return nil, err
	}
	opts := []pulumi.ResourceOption{parentOpt, pulumi.Provider(azProv)}

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
			if err := ctx.RegisterComponentResource("defang:index:AzurePostgres", svcName, comp, opts[0]); err != nil {
				return nil, fmt.Errorf("registering Azure Postgres component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp), pulumi.Provider(azProv)}

			pgResult, err := createPostgresFlexible(ctx, svcName, svc, infra, recipe, svcOpts...)
			if err != nil {
				return nil, fmt.Errorf("creating PostgreSQL for %s: %w", svcName, err)
			}
			endpoints[svcName] = pulumi.Sprintf("%s:5432", pgResult.server.FullyQualifiedDomainName)
		} else {
			// Container service → Container App
			if err := ctx.RegisterComponentResource("defang:index:AzureContainerApp", svcName, comp, opts[0]); err != nil {
				return nil, fmt.Errorf("registering Container App component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp), pulumi.Provider(azProv)}

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
func BuildStandaloneContainerApp(ctx *pulumi.Context, serviceName string, svc common.ServiceConfig, azCfg *common.AzureConfig, opts ...pulumi.ResourceOption) (*ContainerAppResult, error) {
	azProv, location, err := createAzureProvider(ctx, serviceName, azCfg, opts...)
	if err != nil {
		return nil, err
	}
	provOpts := append(opts, pulumi.Provider(azProv))

	rg, err := resources.NewResourceGroup(ctx, serviceName+"-rg", &resources.ResourceGroupArgs{
		Location: pulumi.String(location),
	}, provOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating resource group: %w", err)
	}

	env, err := app.NewManagedEnvironment(ctx, serviceName+"-env", &app.ManagedEnvironmentArgs{
		ResourceGroupName: rg.Name,
		Location:          pulumi.String(location),
	}, provOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating managed environment: %w", err)
	}

	infra := &sharedInfra{resourceGroup: rg, environment: env}
	recipe := LoadRecipe(ctx)

	caResult, err := createContainerApp(ctx, serviceName, svc, infra, recipe, provOpts...)
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
func BuildStandalonePostgres(ctx *pulumi.Context, serviceName string, svc common.ServiceConfig, azCfg *common.AzureConfig, opts ...pulumi.ResourceOption) (*PostgresResult, error) {
	azProv, location, err := createAzureProvider(ctx, serviceName, azCfg, opts...)
	if err != nil {
		return nil, err
	}
	provOpts := append(opts, pulumi.Provider(azProv))

	rg, err := resources.NewResourceGroup(ctx, serviceName+"-rg", &resources.ResourceGroupArgs{
		Location: pulumi.String(location),
	}, provOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating resource group: %w", err)
	}

	// Postgres doesn't need Container Apps environment
	infra := &sharedInfra{resourceGroup: rg}
	recipe := LoadRecipe(ctx)

	pgResult, err := createPostgresFlexible(ctx, serviceName, svc, infra, recipe, provOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating PostgreSQL for %s: %w", serviceName, err)
	}

	return &PostgresResult{
		Endpoint: pulumi.Sprintf("%s:5432", pgResult.server.FullyQualifiedDomainName),
	}, nil
}

// createAzureProvider creates an Azure Native provider and returns the resolved location.
func createAzureProvider(ctx *pulumi.Context, projectName string, azCfg *common.AzureConfig, opts ...pulumi.ResourceOption) (*azurenative.Provider, string, error) {
	location := defaultAzureLocation
	provArgs := &azurenative.ProviderArgs{
		Location: pulumi.StringPtr(location),
	}
	if azCfg != nil {
		if azCfg.Location != "" {
			location = azCfg.Location
			provArgs.Location = pulumi.StringPtr(location)
		}
		if azCfg.SubscriptionID != "" {
			provArgs.SubscriptionId = pulumi.StringPtr(azCfg.SubscriptionID)
		}
	}

	provName := strings.ToLower(projectName)
	azProv, err := azurenative.NewProvider(ctx, provName+"-azure", provArgs, opts...)
	if err != nil {
		return nil, "", fmt.Errorf("creating Azure provider: %w", err)
	}
	return azProv, location, nil
}
