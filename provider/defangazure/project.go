package defangazure

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providerazure "github.com/DefangLabs/pulumi-defang/provider/defangazure/azure"
	"github.com/pulumi/pulumi-azure-native-sdk/app/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Project is the controller struct for the defang-azure:index:Project component.
type Project struct{}

// ProjectInputs defines the top-level inputs for the Azure Project component.
type ProjectInputs struct {
	// Services map: name -> service config
	Services compose.Services `pulumi:"services"          yaml:"services"`
	Networks compose.Networks `pulumi:"networks,optional" yaml:"networks,omitempty"`
}

// ProjectOutputs holds the outputs of the Project component.
type ProjectOutputs struct {
	pulumi.ResourceState

	// Per-service endpoint URLs (service name -> URL)
	Endpoints pulumi.StringMapOutput `pulumi:"endpoints"`

	// Load balancer DNS name (unused for Azure, kept for interface compat)
	LoadBalancerDNS pulumi.StringPtrOutput `pulumi:"loadBalancerDns,optional"`
}

// Construct implements the ComponentResource interface for Project.
func (*Project) Construct(
	ctx *pulumi.Context, name, typ string, inputs ProjectInputs, opts pulumi.ResourceOption,
) (*ProjectOutputs, error) {
	comp := &ProjectOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	childOpt := pulumi.Parent(comp)
	childOpts := []pulumi.ResourceOption{childOpt}

	location := providerazure.Location(ctx)

	rg, err := resources.NewResourceGroup(ctx, name, &resources.ResourceGroupArgs{
		Location: pulumi.String(location),
	}, childOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating resource group: %w", err)
	}

	// Create VNet and private DNS zones when any service uses PostgreSQL (requires VNet integration).
	hasPostgres := false
	hasBuild := false
	for _, svc := range inputs.Services {
		if svc.Postgres != nil {
			hasPostgres = true
		}
		if svc.Build != nil {
			hasBuild = true
		}
	}

	// Bootstrap a minimal SharedInfra (without Environment) so CreateNetworking can reference the RG.
	infra := &providerazure.SharedInfra{ResourceGroup: rg}

	if hasPostgres {
		networking, err := providerazure.CreateNetworking(ctx, name, infra, location, childOpts...)
		if err != nil {
			return nil, fmt.Errorf("creating networking: %w", err)
		}
		infra.Networking = networking

		dns, err := providerazure.CreateDNSZones(ctx, name, infra, networking, childOpts...)
		if err != nil {
			return nil, fmt.Errorf("creating DNS zones: %w", err)
		}
		infra.DNS = dns
	}

	// Build managed environment args; attach VNet infra subnet when networking is configured.
	envArgs := &app.ManagedEnvironmentArgs{
		ResourceGroupName: rg.Name,
		Location:          pulumi.String(location),
	}
	if infra.Networking != nil {
		envArgs.VnetConfiguration = &app.VnetConfigurationArgs{
			InfrastructureSubnetId: infra.Networking.AppsSubnet.ID().ToStringOutput(),
		}
	}

	env, err := app.NewManagedEnvironment(ctx, name, envArgs, childOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating managed environment: %w", err)
	}
	infra.Environment = env

	if hasBuild {
		buildInfra, err := providerazure.CreateBuildInfra(ctx, name, infra, location, childOpts...)
		if err != nil {
			return nil, fmt.Errorf("creating build infrastructure: %w", err)
		}
		infra.BuildInfra = buildInfra
	}

	type serviceComponent struct{ pulumi.ResourceState }
	endpoints := pulumi.StringMap{}

	for svcName, svc := range inputs.Services {
		comp := &serviceComponent{}

		if svc.Postgres != nil {
			if err := ctx.RegisterComponentResource("defang-azure:index:Postgres", svcName, comp, childOpts...); err != nil {
				return nil, fmt.Errorf("registering Azure Postgres component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

			configProvider := providerazure.NewConfigProvider(name)
			pgResult, err := providerazure.CreatePostgresFlexible(ctx, configProvider, svcName, svc, infra, svcOpts...)
			if err != nil {
				return nil, fmt.Errorf("creating PostgreSQL for %s: %w", svcName, err)
			}

			// Add a CNAME in the "internal" DNS zone so services can resolve the
			// postgres server by short name (e.g. "db" → full Azure FQDN).
			if infra.DNS != nil {
				if err := providerazure.AddPostgresDNSRecord(ctx, svcName, pgResult.Server.FullyQualifiedDomainName, infra.DNS, infra, svcOpts...); err != nil {
					return nil, fmt.Errorf("adding DNS record for %s: %w", svcName, err)
				}
			}

			endpoints[svcName] = pulumi.Sprintf("%s:5432", pgResult.Server.FullyQualifiedDomainName)
		} else if svc.Redis != nil {
			if err := ctx.RegisterComponentResource("defang-azure:index:Redis", svcName, comp, childOpts...); err != nil {
				return nil, fmt.Errorf("registering Azure Redis component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

			redisResult, err := providerazure.CreateRedisEnterprise(ctx, svcName, svc, infra, svcOpts...)
			if err != nil {
				return nil, fmt.Errorf("creating Redis for %s: %w", svcName, err)
			}
			endpoints[svcName] = pulumi.Sprintf("%s:10000", redisResult.Cluster.HostName)
		} else {
			if err := ctx.RegisterComponentResource(
				"defang-azure:index:AzureContainerApp", svcName, comp, childOpts...,
			); err != nil {
				return nil, fmt.Errorf("registering Container App component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

			imageURI, err := providerazure.GetServiceImage(ctx, svcName, svc, infra.BuildInfra, infra, svcOpts...)
			if err != nil {
				return nil, fmt.Errorf("resolving image for %s: %w", svcName, err)
			}

			caResult, err := providerazure.CreateContainerApp(ctx, svcName, svc, infra, imageURI, svcOpts...)
			if err != nil {
				return nil, fmt.Errorf("creating Container App %s: %w", svcName, err)
			}
			endpoints[svcName] = caResult.App.LatestRevisionFqdn.ApplyT(func(fqdn string) string {
				if fqdn != "" {
					return "https://" + fqdn
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

	loadBalancerDNS := pulumi.StringPtr("").ToStringPtrOutput()

	comp.Endpoints = endpoints.ToStringMapOutput()
	comp.LoadBalancerDNS = loadBalancerDNS

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoints":       endpoints.ToStringMapOutput(),
		"loadBalancerDns": loadBalancerDNS,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}
