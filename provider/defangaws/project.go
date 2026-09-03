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
	errSidecarParentManaged   = errors.New("sidecar parent must be a container service, not a managed service")
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
		case isManagedService(parentSvc):
			return nil, nil, fmt.Errorf("service %s: parent %q: %w", name, parent, errSidecarParentManaged)
		}
		if sidecars[parent] == nil {
			sidecars[parent] = map[string]compose.ServiceConfig{}
		}
		sidecars[parent][name] = svc
	}
	return standalone, sidecars, nil
}

// isManagedService reports whether a service is dispatched to a managed AWS
// backend (RDS, ElastiCache, S3) instead of an ECS task. Kept local to this
// provider: GCP and Azure do not dispatch x-defang-s3 yet, so
// common.IsManagedService must keep its current meaning for them.
func isManagedService(svc compose.ServiceConfig) bool {
	return svc.Postgres != nil || svc.Redis != nil || svc.ObjectStore != nil
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

	// ServiceIds maps every service name to the physical identifier of its
	// primary backing resource — the ECS service name for container services
	// (same value as serviceNames), the MemoryDB cluster name or ElastiCache
	// replication group ID for Redis, the RDS DBInstanceIdentifier for
	// Postgres — so consumers can attach externally managed alarms and
	// dashboards. Child components' own outputs are unreachable on Project
	// children.
	ServiceIds pulumix.Output[map[string]string] `pulumi:"serviceIds"`
}

// Construct implements the ComponentResource interface for Project.
func (*Project) Construct(
	ctx *pulumi.Context, name, typ string, inputs ProjectInputs, opts pulumi.ResourceOption,
) (*ProjectOutputs, error) {
	comp := &ProjectOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	// The engine gives Construct the plugin identity for our own package;
	// children do not inherit it, so carry it to the Build registration.
	pluginID := common.PluginIdentityFrom(Version, opts)

	result, err := buildProject(ctx, name, inputs, pluginID, pulumi.Parent(comp))

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
	comp.ServiceIds = pulumix.Output[map[string]string](result.ServiceIds)

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoints":       result.Endpoints,
		"loadBalancerDns": result.LoadBalancerDNS,
		"loadBalancerArn": result.LoadBalancerArn,
		"clusterName":     result.ClusterName,
		"logGroupName":    result.LogGroupName,
		"serviceNames":    result.ServiceNames,
		"taskRoleArns":    result.TaskRoleArns,
		"serviceIds":      result.ServiceIds,
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
	ServiceIds      pulumi.StringMapOutput
}

// buildProject creates all AWS resources for the project.
// The AWS provider must be passed via the parent chain (pulumi.Providers on the parent component).
func buildProject(
	ctx *pulumi.Context,
	projectName string,
	args ProjectInputs,
	pluginID common.PluginIdentity,
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
	serviceIds := pulumi.StringMap{}
	dependencies := map[string]pulumi.Resource{} // service name → dependency resource for dependees
	// x-defang-s3 service name → bucket ARN, for the IAM grants attached to
	// the task roles of the services that depend on them. Filled as the loop
	// dispatches each store; the topological sort guarantees a store is
	// created before any service that depends_on it.
	objectStoreArns := map[string]pulumi.StringInput{}

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

	// Bind the project (apex) domain to a service only when the whole project
	// has exactly one ingress port — i.e. a single service with a single
	// ingress port. Otherwise the apex is left unbound (ALB default action).
	infra.ApexServiceName = apexServiceName(standalone)

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

		deps, err := collectDeps(svcName, svc, sidecars[svcName], dependencies)
		if err != nil {
			return nil, err
		}

		waitForHealthy := waitForSteady[svcName] || args.WaitForSteadyState
		res, err := newService(
			ctx, configProvider, svcName, svc, args.Networks, infra, sidecars[svcName], waitForHealthy, deps,
			objectStoreGrants(svc, sidecars[svcName], objectStoreArns), pluginID, parentOpt)
		if err != nil {
			return nil, fmt.Errorf("building service %s: %w", svcName, err)
		}

		endpoints[svcName] = res.Endpoint
		if res.Service != nil {
			serviceNames[svcName] = res.Service.ServiceName.Untyped().(pulumi.StringOutput)
			taskRoleArns[svcName] = res.Service.TaskRoleArn.Untyped().(pulumi.StringOutput)
			serviceIds[svcName] = res.Service.ServiceName.Untyped().(pulumi.StringOutput)
		}
		// For managed services, deliberately overwrite the ECS service name set
		// above with the backing resource's physical ID (RDS instance, MemoryDB/
		// ElastiCache cluster, S3 bucket). newService returns a non-nil
		// DatastoreID only for those, so container services keep their ECS
		// service name.
		if res.DatastoreID != nil {
			serviceIds[svcName] = res.DatastoreID
		}
		if res.Dependency != nil {
			dependencies[svcName] = res.Dependency
		}
		if res.BucketArn != nil {
			objectStoreArns[svcName] = res.BucketArn
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
		ServiceIds:      serviceIds.ToStringMapOutput(),
	}, nil
}

// apexServiceName returns the service that should serve the bare project
// (apex) domain: the sole service that has exactly one ingress port, but only
// when it is the only ingress port in the whole project. Returns "" when the
// project has zero or multiple ingress ports, leaving the apex unbound. The
// result is independent of map iteration order (it depends only on the count
// of ingress ports and, when that count is 1, the single owner).
func apexServiceName(services compose.Services) string {
	owner := ""
	ingressPorts := 0
	for name, svc := range services {
		// Managed services are dispatched as Postgres/Redis/ObjectStore, never
		// as ALB ingress targets, so they can't own the apex even if a port is
		// set.
		if isManagedService(svc) {
			continue
		}
		for _, port := range svc.Ports {
			if port.IsIngress() {
				ingressPorts++
				owner = name
			}
		}
	}
	if ingressPorts != 1 {
		return ""
	}
	return owner
}

// collectDeps returns the resources svc must be created after: the dependency
// handles of the services it names in depends_on. Entries naming svc's own
// sidecars are skipped — those are container dependencies inside the shared
// task definition, not separate resources.
func collectDeps(
	svcName string,
	svc compose.ServiceConfig,
	sidecars map[string]compose.ServiceConfig,
	dependencies map[string]pulumi.Resource,
) ([]pulumi.Resource, error) {
	var deps []pulumi.Resource
	for dep, val := range svc.DependsOn {
		if _, isOwnSidecar := sidecars[dep]; isOwnSidecar {
			continue
		}
		if r, ok := dependencies[dep]; ok {
			deps = append(deps, r)
		} else if val.Required {
			return nil, fmt.Errorf("service %s requires %s: %w", svcName, dep, errDependencyNotFound)
		}
	}
	return deps, nil
}

// objectStoreGrants returns the bucket ARNs of the x-defang-s3 services that
// svc — or one of its sidecars, which share svc's task role — names in
// depends_on. depends_on is the wiring contract: the CLI injects <STORE>_BUCKET
// and <STORE>_REGION into a service off that same edge, so a service that can
// see a bucket name is exactly the one whose task role must reach the bucket.
// Returns nil when the service depends on no store.
func objectStoreGrants(
	svc compose.ServiceConfig,
	sidecars map[string]compose.ServiceConfig,
	stores map[string]pulumi.StringInput,
) map[string]pulumi.StringInput {
	if len(stores) == 0 {
		return nil
	}
	var grants map[string]pulumi.StringInput
	collect := func(dependsOn compose.DependsOnConfig) {
		for dep := range dependsOn {
			arn, ok := stores[dep]
			if !ok {
				continue
			}
			if grants == nil {
				grants = map[string]pulumi.StringInput{}
			}
			grants[dep] = arn
		}
	}
	collect(svc.DependsOn)
	for _, sidecar := range sidecars {
		collect(sidecar.DependsOn)
	}
	return grants
}

// serviceResult is what dispatching one compose service yields: its endpoint,
// the handle dependees order against, and the backend-specific extras.
type serviceResult struct {
	// Endpoint is the service's address — an ALB/private DNS URL for a
	// container service, the datastore's connection endpoint, or the bucket's
	// regional endpoint for an object store.
	Endpoint pulumi.StringOutput
	// Dependency is the resource a dependent service waits on.
	Dependency pulumi.Resource
	// Service is the ECS component; nil for managed services.
	Service *ServiceOutputs
	// DatastoreID is the managed backend's physical identifier (RDS instance,
	// MemoryDB/ElastiCache cluster, S3 bucket name), surfaced through the
	// Project's serviceIds output. nil for container services, whose ECS
	// service name is used instead.
	DatastoreID pulumi.StringInput
	// BucketArn is the object store's bucket ARN; nil for every other kind of
	// service. The project loop hands it to the task-role grant of each
	// service that depends_on this one.
	BucketArn pulumi.StringInput
}

// newService dispatches one compose service to its AWS backend: RDS for
// x-defang-postgres, ElastiCache for x-defang-redis, S3 for x-defang-s3, and
// an ECS service for everything else.
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
	objectStoreArns map[string]pulumi.StringInput,
	pluginID common.PluginIdentity,
	parentOpt pulumi.ResourceOrInvokeOption,
) (*serviceResult, error) {
	res := &serviceResult{}
	var err error
	switch {
	case svc.Postgres != nil:
		// Managed Postgres → RDS
		pgComp := &PostgresOutputs{}
		if regErr := ctx.RegisterComponentResource(PostgresComponentType, svcName, pgComp, parentOpt); regErr != nil {
			return nil, fmt.Errorf("registering postgres component %s: %w", svcName, regErr)
		}
		if err = createPostgres(ctx, pgComp, configProvider, svcName, svc, infra, deps); err == nil {
			res.Endpoint = pgComp.Endpoint
			res.Dependency = pgComp.Dependency
			res.DatastoreID = pgComp.InstanceIdentifier
		}
	case svc.Redis != nil:
		// Managed Redis → ElastiCache
		redisComp := &RedisOutputs{}
		if regErr := ctx.RegisterComponentResource(RedisComponentType, svcName, redisComp, parentOpt); regErr != nil {
			return nil, fmt.Errorf("registering redis component %s: %w", svcName, regErr)
		}
		if err = createRedis(ctx, redisComp, svcName, svc, infra, deps); err == nil {
			res.Endpoint = redisComp.Endpoint
			res.Dependency = redisComp.Dependency
			res.DatastoreID = redisComp.ClusterId
		}
	case svc.ObjectStore != nil:
		// Managed object store → S3 bucket
		storeComp := &ObjectStoreOutputs{}
		if regErr := ctx.RegisterComponentResource(ObjectStoreComponentType, svcName, storeComp, parentOpt); regErr != nil {
			return nil, fmt.Errorf("registering object store component %s: %w", svcName, regErr)
		}
		if err = createObjectStore(ctx, storeComp, svcName, svc, infra, deps); err == nil {
			res.Endpoint = storeComp.Endpoint
			res.Dependency = storeComp.Dependency
			res.DatastoreID = storeComp.Bucket
			res.BucketArn = storeComp.Arn
		}
	default:
		// Container service → ECS
		svcComp := &ServiceOutputs{}
		res.Service = svcComp
		if regErr := ctx.RegisterComponentResource(ServiceComponentType, svcName, svcComp, parentOpt); regErr != nil {
			return nil, fmt.Errorf("registering service component %s: %w", svcName, regErr)
		}
		imageURI, imgErr := provideraws.GetServiceImage(ctx, svcName, svc, infra.BuildInfra, pluginID, pulumi.Parent(svcComp))
		if imgErr != nil {
			return nil, fmt.Errorf("resolving image for %s: %w", svcName, imgErr)
		}
		args := &provideraws.ECSServiceArgs{
			Infra:              infra,
			ImageURI:           imageURI,
			Networks:           networks,
			WaitForSteadyState: waitForSteadyState,
			Sidecars:           sidecars,
			ObjectStoreArns:    objectStoreArns,
		}
		if err = createECSService(ctx, svcComp, configProvider, svcName, svc, args, deps); err == nil {
			res.Endpoint = pulumi.StringOutput(svcComp.Endpoint)
			res.Dependency = svcComp.Dependency
		}
	}
	if err != nil {
		return nil, err
	}
	return res, nil
}
