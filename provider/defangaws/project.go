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
	// Services map: name -> service config
	Services compose.Services `pulumi:"services"          yaml:"services"`
	Networks compose.Networks `pulumi:"networks,optional" yaml:"networks,omitempty"`

	AWS *AWSConfig `pulumi:"aws,optional" yaml:"x-defang-aws,omitempty"`

	// Etag is the deployment identifier supplied by the CD program; the
	// provider injects it as a DEFANG_ETAG env var on every service container
	// so application logs can be correlated with a specific deployment.
	Etag string `pulumi:"etag,optional" yaml:"etag,omitempty"`
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
	// Merge the default network's Compose labels into the Tags of all shared
	// project infrastructure. Attaching the transformation to the Project
	// component cascades it to every child (see provider/compose/labels.go),
	// including the awsx VPC component — awsx propagates its Tags to the subnets,
	// NAT gateways and route tables it creates internally, which an in-process
	// transformation cannot reach across the multi-language-component boundary.
	// The default network spans all services, so these labels also reach service
	// resources; per-service labels (in newService) win on key collision.
	netLabels := inputs.Networks[compose.DefaultNetwork].Labels
	if t := compose.LabelTagsTransformation(netLabels, "aws", "Tags", nil); t != nil {
		opts = pulumi.Composite(opts, pulumi.Transformations([]pulumi.ResourceTransformation{t}))
	}

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
	infra.Etag = args.Etag

	albDNS := pulumix.Val[*string](nil).Untyped().(pulumi.StringPtrOutput)
	if infra.Alb != nil {
		albDNS = infra.Alb.DnsName.ToStringPtrOutput()
	}

	// Deploy each service, wrapped in a component resource for tree organization
	endpoints := pulumi.StringMap{}
	dependencies := map[string]pulumi.Resource{} // service name → dependency resource for dependees

	var configProvider compose.ConfigProvider
	if ctx.DryRun() {
		configProvider = &compose.DryRunConfigProvider{}
	} else {
		configProvider = provideraws.NewConfigProvider(projectName)
	}

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

	// Merge this service's Compose labels into the Tags of every AWS resource it
	// creates. Attaching the transformation to the service component cascades it
	// to all children (see provider/compose/labels.go). AWS tags accept the full
	// Compose label character set, so labels are applied verbatim (nil normalize).
	// Prefix "aws" matches both aws: and awsx: type tokens.
	svcOpts := []pulumi.ResourceOption{parentOpt}
	if t := compose.LabelTagsTransformation(svc.Labels, "aws", "Tags", nil); t != nil {
		svcOpts = append(svcOpts, pulumi.Transformations([]pulumi.ResourceTransformation{t}))
	}

	switch {
	case svc.Postgres != nil:
		// Managed Postgres → RDS
		pgComp := &PostgresOutputs{}
		if regErr := ctx.RegisterComponentResource(PostgresComponentType, svcName, pgComp, svcOpts...); regErr != nil {
			return pulumi.StringOutput{}, nil, fmt.Errorf("registering postgres component %s: %w", svcName, regErr)
		}
		if err = createPostgres(ctx, pgComp, configProvider, svcName, svc, infra, deps); err == nil {
			endpoint = pgComp.Endpoint
			dependency = pgComp.Dependency
		}
	case svc.Redis != nil:
		// Managed Redis → ElastiCache
		redisComp := &RedisOutputs{}
		if regErr := ctx.RegisterComponentResource(RedisComponentType, svcName, redisComp, svcOpts...); regErr != nil {
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
		if regErr := ctx.RegisterComponentResource(ServiceComponentType, svcName, svcComp, svcOpts...); regErr != nil {
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
