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

var (
	errDependencyNotFound     = errors.New("service not found in dependencies map")
	errSidecarParentNotFound  = errors.New("sidecar parent service not found")
	errSidecarParentIsSidecar = errors.New("sidecar parent is itself a sidecar")
	errSidecarParentManaged   = errors.New("sidecar parent must be a container service, not managed Postgres/Redis")
)

// partitionSidecars splits services into standalone services and sidecar
// groups. A service with network_mode "service:<name>" is folded into
// <name>'s task definition as an additional container instead of being
// deployed as its own ECS service.
func partitionSidecars(
	services compose.Services,
) (compose.Services, map[string]map[string]compose.ServiceConfig, error) {
	standalone := compose.Services{}
	sidecars := map[string]map[string]compose.ServiceConfig{}
	for name, svc := range services {
		parent := svc.SidecarParent()
		if parent == "" {
			standalone[name] = svc
			continue
		}
		parentSvc, ok := services[parent]
		switch {
		case !ok:
			return nil, nil, fmt.Errorf("service %s: %w: %q", name, errSidecarParentNotFound, parent)
		case parentSvc.SidecarParent() != "":
			return nil, nil, fmt.Errorf("service %s: parent %q: %w", name, parent, errSidecarParentIsSidecar)
		case parentSvc.Postgres != nil || parentSvc.Redis != nil:
			return nil, nil, fmt.Errorf("service %s: parent %q: %w", name, parent, errSidecarParentManaged)
		}
		if sidecars[parent] == nil {
			sidecars[parent] = map[string]compose.ServiceConfig{}
		}
		sidecars[parent][name] = svc
	}
	return standalone, sidecars, nil
}

// Project is the controller struct for the defang-aws:index:Project component.
type Project struct{}

// ProjectInputs defines the top-level inputs for the AWS Project component.
type ProjectInputs struct {
	// Services map: name -> service config
	Services compose.Services `pulumi:"services"          yaml:"services"`
	Networks compose.Networks `pulumi:"networks,optional" yaml:"networks,omitempty"`

	AWS *AWSConfig `pulumi:"aws,optional" yaml:"x-defang-aws,omitempty"`

	// WaitForSteadyState makes every ECS service deployment wait until the
	// service reaches a steady state (in addition to services other services
	// depend on with condition: service_healthy, which always wait).
	WaitForSteadyState bool `pulumi:"waitForSteadyState,optional" yaml:"waitForSteadyState,omitempty"`

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

	// Load balancer ARN, for attaching externally managed resources (e.g. a
	// WAF web ACL). Unset when no service has ingress.
	LoadBalancerArn pulumix.Output[*string] `pulumi:"loadBalancerArn,optional"`

	// ECS cluster name, for externally managed alarms and dashboards.
	ClusterName pulumix.Output[string] `pulumi:"clusterName"`

	// CloudWatch log group name shared by all services.
	LogGroupName pulumix.Output[string] `pulumi:"logGroupName"`

	// ECS service names by compose service name (container services only).
	ServiceNames pulumix.Output[map[string]string] `pulumi:"serviceNames"`

	// Task role ARNs by compose service name (container services only), for
	// resource-based policies (e.g. KMS key policies) that must name the role.
	TaskRoleArns pulumix.Output[map[string]string] `pulumi:"taskRoleArns"`

	// DatastoreIds maps managed database service names to their physical
	// identifiers — the MemoryDB cluster name or ElastiCache replication
	// group ID for Redis, the RDS DBInstanceIdentifier for Postgres — so
	// consumers can attach externally managed alarms and dashboards. Mirrors
	// the standalone components' clusterId/instanceIdentifier outputs, which
	// are unreachable on Project children.
	DatastoreIds pulumix.Output[map[string]string] `pulumi:"datastoreIds"`
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
	comp.LoadBalancerArn = pulumix.Output[*string](result.LoadBalancerArn)
	comp.ClusterName = pulumix.Output[string](result.ClusterName)
	comp.LogGroupName = pulumix.Output[string](result.LogGroupName)
	comp.ServiceNames = pulumix.Output[map[string]string](result.ServiceNames)
	comp.TaskRoleArns = pulumix.Output[map[string]string](result.TaskRoleArns)
	comp.DatastoreIds = pulumix.Output[map[string]string](result.DatastoreIds)

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoints":       result.Endpoints,
		"loadBalancerDns": result.LoadBalancerDNS,
		"loadBalancerArn": result.LoadBalancerArn,
		"clusterName":     result.ClusterName,
		"logGroupName":    result.LogGroupName,
		"serviceNames":    result.ServiceNames,
		"taskRoleArns":    result.TaskRoleArns,
		"datastoreIds":    result.DatastoreIds,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}

// projectResult extends the cross-provider BuildResult with AWS-specific
// infrastructure handles surfaced as Project outputs.
type projectResult struct {
	common.BuildResult
	LoadBalancerArn pulumi.StringPtrOutput
	ClusterName     pulumi.StringOutput
	LogGroupName    pulumi.StringOutput
	ServiceNames    pulumi.StringMapOutput
	TaskRoleArns    pulumi.StringMapOutput
	DatastoreIds    pulumi.StringMapOutput
}

// buildProject creates all AWS resources for the project.
// The AWS provider must be passed via the parent chain (pulumi.Providers on the parent component).
func buildProject(
	ctx *pulumi.Context,
	projectName string,
	args ProjectInputs,
	parentOpt pulumi.ResourceOrInvokeOption,
) (*projectResult, error) {
	awsConfig := (*provideraws.AWSConfig)(args.AWS)
	infra, err := provideraws.CreateProjectInfra(ctx, projectName, awsConfig, args.Services, parentOpt)
	if err != nil {
		return nil, fmt.Errorf("creating shared infrastructure: %w", err)
	}
	infra.Etag = args.Etag

	albDNS := pulumix.Val[*string](nil).Untyped().(pulumi.StringPtrOutput)
	albArn := pulumix.Val[*string](nil).Untyped().(pulumi.StringPtrOutput)
	if infra.Alb != nil {
		albDNS = infra.Alb.DnsName.ToStringPtrOutput()
		albArn = infra.Alb.Arn.ToStringPtrOutput()
	}

	// Deploy each service, wrapped in a component resource for tree organization
	endpoints := pulumi.StringMap{}
	serviceNames := pulumi.StringMap{}
	taskRoleArns := pulumi.StringMap{}
	datastoreIds := pulumi.StringMap{}
	dependencies := map[string]pulumi.Resource{} // service name → dependency resource for dependees

	var configProvider compose.ConfigProvider
	if ctx.DryRun() {
		configProvider = &compose.DryRunConfigProvider{}
	} else {
		configProvider = provideraws.NewConfigProvider(projectName)
	}

	standalone, sidecars, err := partitionSidecars(args.Services)
	if err != nil {
		return nil, err
	}

	// Pre-compute which services need waitForSteadyState: true if any other
	// service depends on them with condition: service_healthy (matches TS tenant_stack.ts)
	waitForSteady := map[string]bool{}
	for _, other := range standalone {
		for dep, val := range other.DependsOn {
			if val.Condition == "service_healthy" {
				waitForSteady[dep] = true
			}
		}
	}

	sortedNames := common.TopologicalSort(standalone)
	for _, svcName := range sortedNames {
		svc := standalone[svcName]

		// Collect dependency resources from services this one depends on;
		// depends_on entries naming this service's own sidecars are handled
		// as container dependencies inside the task definition.
		var deps []pulumi.Resource
		for dep, val := range svc.DependsOn {
			if _, isOwnSidecar := sidecars[svcName][dep]; isOwnSidecar {
				continue
			}
			if r, ok := dependencies[dep]; ok {
				deps = append(deps, r)
			} else if val.Required {
				return nil, fmt.Errorf("service %s requires %s: %w", svcName, dep, errDependencyNotFound)
			}
		}

		waitForHealthy := waitForSteady[svcName] || args.WaitForSteadyState
		endpoint, dependency, svcComp, datastoreID, err := newService(
			ctx, configProvider, svcName, svc, args.Networks, infra, sidecars[svcName], waitForHealthy, deps, parentOpt)
		if err != nil {
			return nil, fmt.Errorf("building service %s: %w", svcName, err)
		}

		endpoints[svcName] = endpoint
		if svcComp != nil {
			serviceNames[svcName] = svcComp.ServiceName.Untyped().(pulumi.StringOutput)
			taskRoleArns[svcName] = svcComp.TaskRoleArn.Untyped().(pulumi.StringOutput)
		}
		if datastoreID != nil {
			datastoreIds[svcName] = datastoreID
		}
		if dependency != nil {
			dependencies[svcName] = dependency
		}
	}

	return &projectResult{
		BuildResult: common.BuildResult{
			Endpoints:       endpoints.ToStringMapOutput(),
			LoadBalancerDNS: albDNS,
		},
		LoadBalancerArn: albArn,
		ClusterName:     infra.Cluster.Name,
		LogGroupName:    infra.LogGroup.Name,
		ServiceNames:    serviceNames.ToStringMapOutput(),
		TaskRoleArns:    taskRoleArns.ToStringMapOutput(),
		DatastoreIds:    datastoreIds.ToStringMapOutput(),
	}, nil
}

func newService(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	svcName string,
	svc compose.ServiceConfig,
	networks compose.Networks,
	infra *provideraws.SharedInfra,
	sidecars map[string]compose.ServiceConfig,
	waitForSteadyState bool,
	deps []pulumi.Resource,
	parentOpt pulumi.ResourceOrInvokeOption,
) (pulumi.StringOutput, pulumi.Resource, *ServiceOutputs, pulumi.StringInput, error) {
	var endpoint pulumi.StringOutput
	var dependency pulumi.Resource
	var svcComp *ServiceOutputs
	// datastoreID is the managed database's physical identifier (nil for
	// container services), surfaced through the Project's datastoreIds output.
	var datastoreID pulumi.StringInput
	var err error
	switch {
	case svc.Postgres != nil:
		// Managed Postgres → RDS
		pgComp := &PostgresOutputs{}
		if regErr := ctx.RegisterComponentResource(PostgresComponentType, svcName, pgComp, parentOpt); regErr != nil {
			return pulumi.StringOutput{}, nil, nil, nil, fmt.Errorf("registering postgres component %s: %w", svcName, regErr)
		}
		if err = createPostgres(ctx, pgComp, configProvider, svcName, svc, infra, deps); err == nil {
			endpoint = pgComp.Endpoint
			dependency = pgComp.Dependency
			datastoreID = pgComp.InstanceIdentifier
		}
	case svc.Redis != nil:
		// Managed Redis → ElastiCache
		redisComp := &RedisOutputs{}
		if regErr := ctx.RegisterComponentResource(RedisComponentType, svcName, redisComp, parentOpt); regErr != nil {
			return pulumi.StringOutput{}, nil, nil, nil, fmt.Errorf("registering redis component %s: %w", svcName, regErr)
		}
		if err = createRedis(ctx, redisComp, svcName, svc, infra, deps); err == nil {
			endpoint = redisComp.Endpoint
			dependency = redisComp.Dependency
			datastoreID = redisComp.ClusterId
		}
	default:
		// Container service → ECS
		svcComp = &ServiceOutputs{}
		if regErr := ctx.RegisterComponentResource(ServiceComponentType, svcName, svcComp, parentOpt); regErr != nil {
			return pulumi.StringOutput{}, nil, nil, nil, fmt.Errorf("registering service component %s: %w", svcName, regErr)
		}
		imageURI, imgErr := provideraws.GetServiceImage(ctx, svcName, svc, infra.BuildInfra, pulumi.Parent(svcComp))
		if imgErr != nil {
			return pulumi.StringOutput{}, nil, nil, nil, fmt.Errorf("resolving image for %s: %w", svcName, imgErr)
		}
		args := &provideraws.ECSServiceArgs{
			Infra:              infra,
			ImageURI:           imageURI,
			Networks:           networks,
			WaitForSteadyState: waitForSteadyState,
			Sidecars:           sidecars,
		}
		if err = createECSService(ctx, svcComp, configProvider, svcName, svc, args, deps); err == nil {
			endpoint = pulumi.StringOutput(svcComp.Endpoint)
			dependency = svcComp.Dependency
		}
	}
	if err != nil {
		return pulumi.StringOutput{}, nil, nil, nil, err
	}
	return endpoint, dependency, svcComp, datastoreID, nil
}
