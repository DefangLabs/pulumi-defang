package defangscaleway

import (
	"fmt"

	"github.com/DefangLabs/defang/src/pkg/deployment"
	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providerscaleway "github.com/DefangLabs/pulumi-defang/provider/defangscaleway/scaleway"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	scalewayconfig "github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/config"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/network"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/observability"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/registry"
)

// Project is the controller struct for the defang-scaleway:index:Project component.
type Project struct{}

// ProjectInputs defines the top-level inputs for the Scaleway Project component.
type ProjectInputs struct {
	Services compose.Services `pulumi:"services"          yaml:"services"`
	Networks compose.Networks `pulumi:"networks,optional" yaml:"networks,omitempty"`
	Etag     string           `pulumi:"etag,optional"     yaml:"etag,omitempty"`
}

// ProjectOutputs holds the outputs of the Project component.
type ProjectOutputs struct {
	pulumi.ResourceState

	Endpoints       pulumi.StringMapOutput `pulumi:"endpoints"`
	LoadBalancerDNS pulumi.StringPtrOutput `pulumi:"loadBalancerDns,optional"`
	Deployment      pulumi.StringOutput    `pulumi:"defangDeployment"`
}

// Construct implements the ComponentResource interface for Project.
func (*Project) Construct(
	ctx *pulumi.Context, name, typ string, inputs ProjectInputs, opts pulumi.ResourceOption,
) (*ProjectOutputs, error) {
	comp := &ProjectOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	result, err := buildProject(ctx, name, inputs, common.PluginIdentityFrom(opts), pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("failed to build Scaleway resources: %w", err)
	}

	comp.Endpoints = result.Endpoints
	comp.LoadBalancerDNS = result.LoadBalancerDNS
	comp.Deployment = result.Deployment
	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoints":           result.Endpoints,
		"loadBalancerDns":     result.LoadBalancerDNS,
		deployment.OutputName: result.Deployment,
	}); err != nil {
		return nil, err
	}
	return comp, nil
}

type projectResult struct {
	*common.BuildResult
	Deployment pulumi.StringOutput
}

func buildProject(
	ctx *pulumi.Context,
	projectName string,
	args ProjectInputs,
	pluginID common.PluginIdentity,
	parentOpts ...pulumi.ResourceOption,
) (*projectResult, error) {
	childOpts := append([]pulumi.ResourceOption{}, parentOpts...)
	needsBuild := false
	for _, service := range args.Services {
		if service.Build != nil {
			needsBuild = true
			break
		}
	}
	infra, err := buildSharedInfra(ctx, projectName, needsBuild, childOpts...)
	if err != nil {
		return nil, err
	}
	infra.Etag = args.Etag
	infra.PluginIdentity = pluginID

	configProvider := compose.ConfigProvider(&compose.PulumiConfigProvider{})
	if ctx.DryRun() {
		configProvider = &compose.DryRunConfigProvider{}
	}
	if infra.ConfigProvider != nil {
		configProvider = infra.ConfigProvider
	}

	endpoints := pulumi.StringMap{}
	dependencies := map[string]pulumi.Resource{}
	var serviceOutputs []any
	for _, svcName := range common.TopologicalSort(args.Services) {
		svc := args.Services[svcName]
		var deps []pulumi.Resource
		for dep := range svc.DependsOn {
			if r, ok := dependencies[dep]; ok {
				deps = append(deps, r)
			}
		}
		endpoint, component, err := buildService(ctx, configProvider, svcName, svc, infra, deps, childOpts)
		if err != nil {
			return nil, err
		}
		endpoints[svcName] = endpoint
		dependencies[svcName] = component
		switch service := component.(type) {
		case *ScalewayServiceOutputs:
			serviceOutputs = append(serviceOutputs, service.Deployment)
		case *ScalewayPostgresOutputs:
			serviceOutputs = append(serviceOutputs, service.Deployment)
		case *ScalewayRedisOutputs:
			serviceOutputs = append(serviceOutputs, service.Deployment)
		}
	}

	return &projectResult{BuildResult: &common.BuildResult{
		Endpoints:       endpoints.ToStringMapOutput(),
		LoadBalancerDNS: pulumi.StringPtr("").ToStringPtrOutput(),
	}, Deployment: projectDeployment(ctx, args.Etag, serviceOutputs)}, nil
}

func buildSharedInfra(
	ctx *pulumi.Context,
	projectName string,
	needsBuild bool,
	opts ...pulumi.ResourceOption,
) (*providerscaleway.SharedInfra, error) {
	infra := providerscaleway.NewStandaloneInfra(ctx, projectName)
	infra.ConfigProvider = providerscaleway.NewConfigProvider(projectName)

	namespace, err := providerscaleway.CreateContainerNamespace(ctx, projectName, infra, opts...)
	if err != nil {
		return nil, err
	}
	infra.Namespace = namespace
	infra.BuildInfra = &providerscaleway.BuildInfra{}
	if needsBuild {
		// Serverless Containers no longer creates a registry alongside its namespace.
		buildRegistry, err := registry.NewNamespace(ctx, projectName+"-registry", &registry.NamespaceArgs{
			ProjectId: pulumi.StringPtr(infra.ProjectID),
			Region:    pulumi.StringPtr(infra.Region),
			IsPublic:  pulumi.Bool(false),
		}, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating Scaleway build registry: %w", err)
		}
		infra.BuildInfra.RegistryEndpoint = buildRegistry.Endpoint
	}
	infra.ManagedHosts = make(map[string]pulumi.StringOutput)
	infra.ManagedConnectionURLs = make(map[string]pulumi.StringOutput)

	pn, err := network.NewPrivateNetwork(ctx, projectName+"-private-network", &network.PrivateNetworkArgs{
		Name: pulumi.StringPtr(projectName),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Scaleway private network: %w", err)
	}
	infra.PrivateNetwork = pn
	if infra.Region == "" {
		infra.Region = scalewayconfig.GetRegion(ctx)
	}
	if infra.ProjectID == "" {
		infra.ProjectID = scalewayconfig.GetProjectId(ctx)
	}

	logsSource, err := observability.NewSource(ctx, projectName+"-build-logs", &observability.SourceArgs{
		ProjectId:     pulumi.StringPtr(infra.ProjectID),
		Name:          pulumi.StringPtr(projectName + "-build-logs"),
		Type:          pulumi.StringPtr("logs"),
		RetentionDays: pulumi.Int(providerscaleway.LogRetentionDays.Get(ctx)),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Cockpit build-logs data source: %w", err)
	}
	logsToken, err := observability.NewToken(ctx, projectName+"-build-logs", &observability.TokenArgs{
		ProjectId: pulumi.StringPtr(infra.ProjectID),
		Name:      pulumi.StringPtr(projectName + "-build-logs"),
		Scopes: &observability.TokenScopesArgs{
			QueryLogs: pulumi.Bool(true),
			WriteLogs: pulumi.Bool(true),
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Cockpit build-logs token: %w", err)
	}
	infra.BuildInfra.LogsEndpoint = logsSource.Url
	infra.BuildInfra.LogsToken = logsToken.SecretKey

	return infra, nil
}

func buildService(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	svcName string,
	svc compose.ServiceConfig,
	infra *providerscaleway.SharedInfra,
	deps []pulumi.Resource,
	childOpts []pulumi.ResourceOption,
) (pulumi.StringOutput, pulumi.Resource, error) {
	svcChildOpts := childOpts
	if len(deps) > 0 {
		svcChildOpts = append(svcChildOpts, pulumi.DependsOn(deps))
	}
	switch {
	case svc.Postgres != nil:
		pgComp := &ScalewayPostgresOutputs{}
		if err := ctx.RegisterComponentResource(PostgresComponentType, svcName, pgComp, svcChildOpts...); err != nil {
			return pulumi.StringOutput{}, nil, fmt.Errorf("registering Scaleway PostgreSQL component %s: %w", svcName, err)
		}
		if err := createPostgres(ctx, pgComp, configProvider, svcName, svc, infra); err != nil {
			return pulumi.StringOutput{}, nil, err
		}
		return pgComp.Endpoint, pgComp, nil
	case svc.Redis != nil:
		redisComp := &ScalewayRedisOutputs{}
		if err := ctx.RegisterComponentResource(RedisComponentType, svcName, redisComp, svcChildOpts...); err != nil {
			return pulumi.StringOutput{}, nil, fmt.Errorf("registering Scaleway Redis component %s: %w", svcName, err)
		}
		if err := createRedis(ctx, redisComp, configProvider, svcName, svc, infra); err != nil {
			return pulumi.StringOutput{}, nil, err
		}
		return redisComp.Endpoint, redisComp, nil
	default:
		svcComp := &ScalewayServiceOutputs{}
		if err := ctx.RegisterComponentResource(ServiceComponentType, svcName, svcComp, svcChildOpts...); err != nil {
			return pulumi.StringOutput{}, nil, fmt.Errorf("registering Scaleway Service component %s: %w", svcName, err)
		}
		imageURI, err := providerscaleway.GetServiceImage(ctx, svcName, svc, infra.BuildInfra, infra, pulumi.Parent(svcComp))
		if err != nil {
			return pulumi.StringOutput{}, nil, err
		}
		if err := createService(ctx, svcComp, configProvider, svcName, imageURI, svc, infra); err != nil {
			return pulumi.StringOutput{}, nil, err
		}
		return svcComp.Endpoint, svcComp, nil
	}
}
