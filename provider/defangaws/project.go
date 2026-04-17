package defangaws

import (
	"errors"
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

var errDependencyNotFound = errors.New("service not found in dependencies map")

// Project is the controller struct for the defang-aws:index:Project component.
type Project struct{}

// ProjectInputs defines the top-level inputs for the AWS Project component.
type ProjectInputs struct {
	// compose.Project
	Services compose.Services `pulumi:"services"          yaml:"services"`
	Networks compose.Networks `pulumi:"networks,optional" yaml:"networks,omitempty"`

	AWS *AWSConfig `pulumi:"aws,optional" yaml:"x-defang-aws,omitempty"`
}

type AWSConfig provideraws.AWSConfig

// ProjectOutputs holds the outputs of the Project component.
type ProjectOutputs struct {
	pulumi.ResourceState

	// Per-service endpoint URLs (service name -> URL)
	Endpoints pulumix.Output[map[string]string] `pulumi:"endpoints"`

	// Load balancer DNS name (AWS ALB)
	LoadBalancerDNS pulumix.Output[*string] `pulumi:"loadBalancerDns,optional"`
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
		return nil, fmt.Errorf("failed to build AWS resources: %w", err)
	}

	comp.Endpoints = pulumix.Output[map[string]string](result.Endpoints)
	comp.LoadBalancerDNS = pulumix.Output[*string](result.LoadBalancerDNS)

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoints":       result.Endpoints,
		"loadBalancerDns": result.LoadBalancerDNS,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}

// buildProject creates all AWS resources for the project.
// The AWS provider must be passed via the parent chain (pulumi.Providers on the parent component).
func buildProject(
	ctx *pulumi.Context,
	projectName string,
	args ProjectInputs,
	parentOpt pulumi.ResourceOrInvokeOption,
) (*common.BuildResult, error) {
	awsConfig := (*provideraws.AWSConfig)(args.AWS)
	infra, err := provideraws.CreateProjectInfra(ctx, projectName, awsConfig, args.Services, parentOpt)
	if err != nil {
		return nil, fmt.Errorf("creating shared infrastructure: %w", err)
	}

	albDNS := pulumix.Val[*string](nil).Untyped().(pulumi.StringPtrOutput)
	if infra.Alb != nil {
		albDNS = infra.Alb.DnsName.ToStringPtrOutput()
	}

	// Deploy each service, wrapped in a component resource for tree organization
	endpoints := pulumi.StringMap{}
	dependencies := map[string]pulumi.Resource{} // service name → dependency resource for dependees

	configProvider := provideraws.NewConfigProvider(projectName)

	// Pre-compute which services need waitForSteadyState: true if any other
	// service depends on them with condition: service_healthy (matches TS tenant_stack.ts)
	waitForSteady := map[string]bool{}
	for _, other := range args.Services {
		for dep, val := range other.DependsOn {
			if val.Condition == "service_healthy" {
				waitForSteady[dep] = true
			}
		}
	}

	serviceNames := common.TopologicalSort(args.Services)
	for _, svcName := range serviceNames {
		svc := args.Services[svcName]

		// Collect dependency resources from services this one depends on
		var deps []pulumi.Resource
		for dep, val := range svc.DependsOn {
			if r, ok := dependencies[dep]; ok {
				deps = append(deps, r)
			} else if val.Required {
				return nil, fmt.Errorf("service %s requires %s: %w", svcName, dep, errDependencyNotFound)
			}
		}

		waitForHealthy := waitForSteady[svcName]
		endpoint, dependency, err := newService(
			ctx, configProvider, svcName, svc, args.Networks, infra, waitForHealthy, deps, parentOpt)
		if err != nil {
			return nil, fmt.Errorf("building service %s: %w", svcName, err)
		}

		endpoints[svcName] = endpoint
		if dependency != nil {
			dependencies[svcName] = dependency
		}
	}

	return &common.BuildResult{
		Endpoints:       endpoints.ToStringMapOutput(),
		LoadBalancerDNS: albDNS,
	}, nil
}

func newService(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	svcName string,
	svc compose.ServiceConfig,
	networks compose.Networks,
	infra *provideraws.SharedInfra,
	waitForSteadyState bool,
	deps []pulumi.Resource,
	parentOpt pulumi.ResourceOrInvokeOption,
) (pulumi.StringOutput, pulumi.Resource, error) {
	var endpoint pulumi.StringOutput
	var dependency pulumi.Resource
	var err error
	switch {
	case svc.Postgres != nil:
		// Managed Postgres → RDS
		pgComp := &PostgresOutputs{}
		if regErr := ctx.RegisterComponentResource(PostgresComponentType, svcName, pgComp, parentOpt); regErr != nil {
			return pulumi.StringOutput{}, nil, fmt.Errorf("registering postgres component %s: %w", svcName, regErr)
		}
		if err = createPostgres(ctx, pgComp, configProvider, svcName, svc, infra, deps); err == nil {
			endpoint = pgComp.Endpoint
			dependency = pgComp.Dependency
		}
	case svc.Redis != nil:
		// Managed Redis → ElastiCache
		redisComp := &RedisOutputs{}
		if regErr := ctx.RegisterComponentResource(RedisComponentType, svcName, redisComp, parentOpt); regErr != nil {
			return pulumi.StringOutput{}, nil, fmt.Errorf("registering redis component %s: %w", svcName, regErr)
		}
		if err = createRedis(ctx, redisComp, svcName, svc, infra, deps); err == nil {
			endpoint = redisComp.Endpoint
			dependency = redisComp.Dependency
		}
	default:
		// TODO: detect sidecar services (network_mode: "service:<name>", volumes_from)
		// and add them as additional containers in the parent's task definition
		// instead of creating a separate ECS service.

		// Container service → ECS
		svcComp := &ServiceOutputs{}
		if regErr := ctx.RegisterComponentResource(ServiceComponentType, svcName, svcComp, parentOpt); regErr != nil {
			return pulumi.StringOutput{}, nil, fmt.Errorf("registering service component %s: %w", svcName, regErr)
		}
		imageURI, imgErr := provideraws.GetServiceImage(ctx, svcName, svc, infra.BuildInfra, pulumi.Parent(svcComp))
		if imgErr != nil {
			return pulumi.StringOutput{}, nil, fmt.Errorf("resolving image for %s: %w", svcName, imgErr)
		}
		args := &provideraws.ECSServiceArgs{
			Infra:              infra,
			ImageURI:           imageURI,
			Networks:           networks,
			WaitForSteadyState: waitForSteadyState,
		}
		if err = createECSService(ctx, svcComp, configProvider, svcName, svc, args, deps); err == nil {
			endpoint = pulumi.StringOutput(svcComp.Endpoint)
			dependency = svcComp.Dependency
		}
	}
	if err != nil {
		return pulumi.StringOutput{}, nil, err
	}
	return endpoint, dependency, nil
}
