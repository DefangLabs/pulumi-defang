package defangazure

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providerazure "github.com/DefangLabs/pulumi-defang/provider/defangazure/azure"
	"github.com/pulumi/pulumi-azure-native-sdk/app/v2"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Project is the controller struct for the defang-azure:index:Project component.
type Project struct{}

// ProjectInputs defines the top-level inputs for the Azure Project component.
type ProjectInputs struct {
	// Services map: name -> service config
	Services map[string]compose.ServiceConfig       `pulumi:"services" yaml:"services"`
	Networks map[string]compose.NetworkConfigInput `pulumi:"networks,optional" yaml:"networks,omitempty"`
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

	env, err := app.NewManagedEnvironment(ctx, name, &app.ManagedEnvironmentArgs{
		ResourceGroupName: rg.Name,
		Location:          pulumi.String(location),
	}, childOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating managed environment: %w", err)
	}

	infra := &providerazure.SharedInfra{
		ResourceGroup: rg,
		Environment:   env,
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
			endpoints[svcName] = pulumi.Sprintf("%s:5432", pgResult.Server.FullyQualifiedDomainName)
		} else {
			if err := ctx.RegisterComponentResource(
				"defang-azure:index:AzureContainerApp", svcName, comp, childOpts...,
			); err != nil {
				return nil, fmt.Errorf("registering Container App component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

			caResult, err := providerazure.CreateContainerApp(ctx, svcName, svc, infra, svcOpts...)
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
