package defangazure

import (
	"fmt"
	"strings"

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

	// Create VNet and private DNS zones when any service uses PostgreSQL or Redis.
	// PostgreSQL requires VNet integration; Redis uses private endpoints within the VNet.
	hasPostgres := false
	hasRedis := false
	hasBuild := false
	hasLLM := false
	for _, svc := range inputs.Services {
		if svc.Postgres != nil {
			hasPostgres = true
		}
		if svc.Redis != nil {
			hasRedis = true
		}
		if svc.Build != nil {
			hasBuild = true
		}
		if svc.LLM != nil {
			hasLLM = true
		}
	}

	// llmModels maps each LLM service name to its model alias (e.g. "llm" → "chat-default").
	// The CLI injects {UPPER(svcName)}_MODEL into dependent services; we reverse-look that up here.
	llmModels := make(map[string]string)
	if hasLLM {
		for svcName, svc := range inputs.Services {
			if svc.LLM == nil {
				continue
			}
			alias := llmModelAlias(svcName, inputs.Services)
			llmModels[svcName] = alias
		}
	}

	// Bootstrap a minimal SharedInfra (without Environment) so CreateNetworking can reference the RG.
	infra := &providerazure.SharedInfra{
		ResourceGroup:  rg,
		ConfigProvider: providerazure.NewConfigProvider(name),
	}

	if hasPostgres || hasRedis {
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

	// VnetConfiguration is immutable on Azure — adding/changing it requires replacement.
	envOpts := append(childOpts, pulumi.ReplaceOnChanges([]string{"vnetConfiguration"}))
	env, err := app.NewManagedEnvironment(ctx, name, envArgs, envOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating managed environment: %w", err)
	}
	infra.Environment = env

	if hasLLM {
		llmInfra, err := providerazure.CreateLLMInfra(ctx, name, infra, childOpts...)
		if err != nil {
			return nil, fmt.Errorf("creating LLM infra: %w", err)
		}
		infra.LLMInfra = llmInfra
	}

	if hasBuild {
		buildInfra, err := providerazure.CreateBuildInfra(ctx, name, infra, location, childOpts...)
		if err != nil {
			return nil, fmt.Errorf("creating build infrastructure: %w", err)
		}
		infra.BuildInfra = buildInfra
	}

	type serviceComponent struct{ pulumi.ResourceState }

	endpoints := pulumi.StringMap{}

	// managedEndpoints accumulates connection URLs for managed services (Postgres, Redis, LLM)
	// as they are created. The CLI guarantees that a service appears before any service that
	// depends on it, so container apps will always find their managed endpoints already populated.
	managedEndpoints := make(map[string]pulumi.StringOutput, len(inputs.Services))

	for svcName, svc := range inputs.Services {
		comp := &serviceComponent{}
		var endpoint pulumi.StringOutput

		if svc.Postgres != nil {
			if err := ctx.RegisterComponentResource("defang-azure:index:Postgres", svcName, comp, childOpts...); err != nil {
				return nil, fmt.Errorf("registering Azure Postgres component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

			pgResult, err := providerazure.CreatePostgresFlexible(ctx, infra.ConfigProvider, svcName, svc, infra, svcOpts...)
			if err != nil {
				return nil, fmt.Errorf("creating PostgreSQL for %s: %w", svcName, err)
			}

			if infra.DNS != nil {
				if err := providerazure.AddPostgresDNSRecord(ctx, svcName, pgResult.Server.FullyQualifiedDomainName, infra.DNS, infra, svcOpts...); err != nil {
					return nil, fmt.Errorf("adding DNS record for %s: %w", svcName, err)
				}
			}

			ep := pulumi.Sprintf("%s:5432", pgResult.Server.FullyQualifiedDomainName)
			managedEndpoints[svcName] = ep
			endpoint = ep

		} else if svc.Redis != nil {
			if err := ctx.RegisterComponentResource("defang-azure:index:Redis", svcName, comp, childOpts...); err != nil {
				return nil, fmt.Errorf("registering Azure Redis component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

			redisResult, err := providerazure.CreateRedisEnterprise(ctx, svcName, svc, infra, svcOpts...)
			if err != nil {
				return nil, fmt.Errorf("creating Redis for %s: %w", svcName, err)
			}

			if infra.DNS != nil {
				if err := providerazure.AddRedisDNSRecord(ctx, svcName, redisResult.Cluster.HostName, infra.DNS, infra, svcOpts...); err != nil {
					return nil, fmt.Errorf("adding Redis DNS record for %s: %w", svcName, err)
				}
			}

			// ConnectionURL is the full redis:// or rediss:// URL including auth.
			// The Compose service name resolves via svc.internal CNAME → private endpoint.
			managedEndpoints[svcName] = redisResult.ConnectionURL
			endpoint = pulumi.Sprintf("%s:10000", redisResult.Cluster.HostName)

		} else if svc.LLM != nil {
			if infra.LLMInfra == nil {
				return nil, fmt.Errorf("service %s uses LLM but LLM infra was not created", svcName)
			}
			if err := ctx.RegisterComponentResource("defang-azure:index:LLM", svcName, comp, childOpts...); err != nil {
				return nil, fmt.Errorf("registering Azure LLM component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

			modelAlias := llmModels[svcName]
			if err := providerazure.CreateLLMDeployment(ctx, modelAlias, modelAlias, infra.LLMInfra, infra, svcOpts...); err != nil {
				return nil, fmt.Errorf("creating LLM deployment for %s: %w", svcName, err)
			}

			managedEndpoints[svcName] = infra.LLMInfra.BaseURL
			endpoint = infra.LLMInfra.BaseURL

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

			caResult, err := providerazure.CreateContainerApp(ctx, svcName, svc, infra, imageURI, managedEndpoints, svcOpts...)
			if err != nil {
				return nil, fmt.Errorf("creating Container App %s: %w", svcName, err)
			}
			endpoint = caResult.App.LatestRevisionFqdn.ApplyT(func(fqdn string) string {
				if fqdn != "" {
					return "https://" + fqdn
				}
				return ""
			}).(pulumi.StringOutput)
		}

		endpoints[svcName] = endpoint
		if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
			"endpoint": endpoint,
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

// llmModelAlias finds the model alias for an LLM service by scanning dependent services'
// environments. The CLI injects {UPPER(svcName)}_MODEL into every service that depends on
// the LLM service. Returns the alias (e.g. "chat-default") or svcName as a fallback.
func llmModelAlias(svcName string, services compose.Services) string {
	envKey := strings.ToUpper(svcName) + "_MODEL"
	for _, svc := range services {
		if v, ok := svc.Environment[envKey]; ok && v != "" {
			return v
		}
	}
	return svcName // fallback: use service name as deployment name
}
