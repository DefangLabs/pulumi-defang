package defangscaleway

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providerscaleway "github.com/DefangLabs/pulumi-defang/provider/defangscaleway/scaleway"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	scalewayconfig "github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/config"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/network"
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
}

// Construct implements the ComponentResource interface for Project.
func (*Project) Construct(
	ctx *pulumi.Context, name, typ string, inputs ProjectInputs, opts pulumi.ResourceOption,
) (*ProjectOutputs, error) {
	comp := &ProjectOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	result, err := buildProject(ctx, name, inputs, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("failed to build Scaleway resources: %w", err)
	}

	comp.Endpoints = result.Endpoints
	comp.LoadBalancerDNS = result.LoadBalancerDNS
	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoints":       result.Endpoints,
		"loadBalancerDns": result.LoadBalancerDNS,
	}); err != nil {
		return nil, err
	}
	return comp, nil
}

func buildProject(
	ctx *pulumi.Context,
	projectName string,
	args ProjectInputs,
	parentOpts ...pulumi.ResourceOption,
) (*common.BuildResult, error) {
	childOpts := append([]pulumi.ResourceOption{}, parentOpts...)
	infra, err := buildSharedInfra(ctx, projectName, childOpts...)
	if err != nil {
		return nil, err
	}
	infra.Etag = args.Etag

	configProvider := compose.ConfigProvider(&compose.PulumiConfigProvider{})
	if ctx.DryRun() {
		configProvider = &compose.DryRunConfigProvider{}
	}
	if infra.ConfigProvider != nil {
		configProvider = infra.ConfigProvider
	}

	endpoints := pulumi.StringMap{}
	dependencies := map[string]pulumi.Resource{}
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
	}

	return &common.BuildResult{
		Endpoints:       endpoints.ToStringMapOutput(),
		LoadBalancerDNS: pulumi.StringPtr("").ToStringPtrOutput(),
	}, nil
}

func buildSharedInfra(
	ctx *pulumi.Context,
	projectName string,
	opts ...pulumi.ResourceOption,
) (*providerscaleway.SharedInfra, error) {
	infra := providerscaleway.NewStandaloneInfra(ctx, projectName)
	infra.ConfigProvider = providerscaleway.NewConfigProvider(projectName)

	namespace, err := providerscaleway.CreateContainerNamespace(ctx, projectName, infra, opts...)
	if err != nil {
		return nil, err
	}
	infra.Namespace = namespace
	infra.BuildInfra = &providerscaleway.BuildInfra{
		RegistryEndpoint: namespace.RegistryEndpoint,
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
