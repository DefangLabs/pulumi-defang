package defangazure

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providerazure "github.com/DefangLabs/pulumi-defang/provider/defangazure/azure"
	"github.com/pulumi/pulumi-azure-native-sdk/app/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var ErrLLMInfraNotCreated = errors.New("LLM infra was not created")

// serviceComponent is a generic Pulumi component resource for a single service.
type serviceComponent struct{ pulumi.ResourceState }

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

// detectServiceTypes scans all services and returns feature flags and an llmModels map.
// llmModels maps each LLM service name to its model alias (e.g. "llm" → "chat-default").
// The CLI injects {UPPER(svcName)}_MODEL into dependent services; we reverse-look that up here.
func detectServiceTypes(services compose.Services) (bool, bool, bool, bool, map[string]string) {
	var hasPostgres, hasRedis, hasBuild, hasLLM bool
	llmModels := make(map[string]string)
	for svcName, svc := range services {
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
			llmModels[svcName] = llmModelAlias(svcName, services)
		}
	}
	return hasPostgres, hasRedis, hasBuild, hasLLM, llmModels
}

func createPostgresResources(
	ctx *pulumi.Context,
	svcName string,
	svc compose.ServiceConfig,
	infra *providerazure.SharedInfra,
	managedEndpoints map[string]pulumi.StringOutput,
	comp *serviceComponent,
	childOpts []pulumi.ResourceOption,
) (pulumi.StringOutput, error) {
	if err := ctx.RegisterComponentResource("defang-azure:index:Postgres", svcName, comp, childOpts...); err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("registering Azure Postgres component %s: %w", svcName, err)
	}
	svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

	pgResult, err := providerazure.CreatePostgresFlexible(ctx, infra.ConfigProvider, svcName, svc, infra, svcOpts...)
	if err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("creating PostgreSQL for %s: %w", svcName, err)
	}

	if infra.DNS != nil {
		if err := providerazure.AddPostgresDNSRecord(
			ctx, svcName, pgResult.Server.FullyQualifiedDomainName,
			infra.DNS, infra, svcOpts...,
		); err != nil {
			return pulumi.StringOutput{}, fmt.Errorf("adding DNS record for %s: %w", svcName, err)
		}
	}

	endpoint := pulumi.Sprintf("%s:5432", pgResult.Server.FullyQualifiedDomainName)
	managedEndpoints[svcName] = endpoint
	return endpoint, nil
}

func createRedisResources(
	ctx *pulumi.Context,
	svcName string,
	svc compose.ServiceConfig,
	infra *providerazure.SharedInfra,
	managedEndpoints map[string]pulumi.StringOutput,
	comp *serviceComponent,
	childOpts []pulumi.ResourceOption,
) (pulumi.StringOutput, error) {
	if err := ctx.RegisterComponentResource("defang-azure:index:Redis", svcName, comp, childOpts...); err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("registering Azure Redis component %s: %w", svcName, err)
	}
	svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

	redisResult, err := providerazure.CreateRedisEnterprise(ctx, svcName, svc, infra, svcOpts...)
	if err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("creating Redis for %s: %w", svcName, err)
	}

	if infra.DNS != nil {
		if err := providerazure.AddRedisDNSRecord(
			ctx, svcName, redisResult.Cluster.HostName,
			infra.DNS, infra, svcOpts...,
		); err != nil {
			return pulumi.StringOutput{}, fmt.Errorf("adding Redis DNS record for %s: %w", svcName, err)
		}
	}

	// ConnectionURL is the full redis:// or rediss:// URL including auth.
	// The Compose service name resolves via svc.internal CNAME → private endpoint.
	managedEndpoints[svcName] = redisResult.ConnectionURL
	return pulumi.Sprintf("%s:10000", redisResult.Cluster.HostName), nil
}

// createServiceResources creates the component resource and underlying cloud resources
// for a single service. It updates managedEndpoints in-place for managed services and
// returns the service endpoint.
func createServiceResources(
	ctx *pulumi.Context,
	svcName string,
	svc compose.ServiceConfig,
	infra *providerazure.SharedInfra,
	managedEndpoints map[string]pulumi.StringOutput,
	llmModels map[string]string,
	childOpts []pulumi.ResourceOption,
) (pulumi.StringOutput, error) {
	comp := &serviceComponent{}
	var endpoint pulumi.StringOutput

	switch {
	case svc.Postgres != nil:
		var err error
		endpoint, err = createPostgresResources(ctx, svcName, svc, infra, managedEndpoints, comp, childOpts)
		if err != nil {
			return pulumi.StringOutput{}, err
		}

	case svc.Redis != nil:
		var err error
		endpoint, err = createRedisResources(ctx, svcName, svc, infra, managedEndpoints, comp, childOpts)
		if err != nil {
			return pulumi.StringOutput{}, err
		}

	case svc.LLM != nil:
		if infra.LLMInfra == nil {
			return pulumi.StringOutput{}, fmt.Errorf("service %s: %w", svcName, ErrLLMInfraNotCreated)
		}
		if err := ctx.RegisterComponentResource("defang-azure:index:LLM", svcName, comp, childOpts...); err != nil {
			return pulumi.StringOutput{}, fmt.Errorf("registering Azure LLM component %s: %w", svcName, err)
		}
		svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

		modelAlias := llmModels[svcName]
		if err := providerazure.CreateLLMDeployment(
			ctx, modelAlias, modelAlias, infra.LLMInfra, infra, svcOpts...,
		); err != nil {
			return pulumi.StringOutput{}, fmt.Errorf("creating LLM deployment for %s: %w", svcName, err)
		}

		managedEndpoints[svcName] = infra.LLMInfra.BaseURL
		endpoint = infra.LLMInfra.BaseURL

	default:
		if err := ctx.RegisterComponentResource(
			"defang-azure:index:AzureContainerApp", svcName, comp, childOpts...,
		); err != nil {
			return pulumi.StringOutput{}, fmt.Errorf("registering Container App component %s: %w", svcName, err)
		}
		svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

		imageURI, err := providerazure.GetServiceImage(ctx, svcName, svc, infra.BuildInfra, infra, svcOpts...)
		if err != nil {
			return pulumi.StringOutput{}, fmt.Errorf("resolving image for %s: %w", svcName, err)
		}

		caResult, err := providerazure.CreateContainerApp(ctx, svcName, svc, infra, imageURI, managedEndpoints, svcOpts...)
		if err != nil {
			return pulumi.StringOutput{}, fmt.Errorf("creating Container App %s: %w", svcName, err)
		}
		endpoint = caResult.App.LatestRevisionFqdn.ApplyT(fqdnToHTTPS).(pulumi.StringOutput)
	}

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": endpoint,
	}); err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("registering outputs for %s: %w", svcName, err)
	}

	return endpoint, nil
}

// Construct implements the ComponentResource interface for Project.
func (*Project) Construct(
	ctx *pulumi.Context, name, typ string, inputs ProjectInputs, opts pulumi.ResourceOption,
) (*ProjectOutputs, error) {
	comp := &ProjectOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	childOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

	location := providerazure.Location(ctx)

	rg, err := resources.NewResourceGroup(ctx, name, &resources.ResourceGroupArgs{
		Location: pulumi.String(location),
	}, childOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating resource group: %w", err)
	}

	// Create VNet and private DNS zones when any service uses PostgreSQL or Redis.
	// PostgreSQL requires VNet integration; Redis uses private endpoints within the VNet.
	hasPostgres, hasRedis, hasBuild, hasLLM, llmModels := detectServiceTypes(inputs.Services)

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
	envOpts := append([]pulumi.ResourceOption{pulumi.ReplaceOnChanges([]string{"vnetConfiguration"})}, childOpts...)
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

	endpoints := pulumi.StringMap{}

	// managedEndpoints accumulates connection URLs for managed services (Postgres, Redis, LLM)
	// as they are created. The CLI guarantees that a service appears before any service that
	// depends on it, so container apps will always find their managed endpoints already populated.
	managedEndpoints := make(map[string]pulumi.StringOutput, len(inputs.Services))

	for svcName, svc := range inputs.Services {
		endpoint, err := createServiceResources(ctx, svcName, svc, infra, managedEndpoints, llmModels, childOpts)
		if err != nil {
			return nil, err
		}
		endpoints[svcName] = endpoint
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

// fqdnToHTTPS converts a Container App FQDN to an https:// URL, or returns "" for empty FQDNs.
// Used as an ApplyT callback for LatestRevisionFqdn.
func fqdnToHTTPS(fqdn string) string {
	if fqdn == "" {
		return ""
	}
	return "https://" + fqdn
}
