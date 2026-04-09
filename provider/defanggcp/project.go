package defanggcp

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providergcp "github.com/DefangLabs/pulumi-defang/provider/defanggcp/gcp"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Project is the controller struct for the defang-gcp:index:Project component.
type Project struct{}

// ProjectInputs defines the top-level inputs for the GCP Project component.
type ProjectInputs struct {
	// Services map: name -> service config
	Services compose.Services `pulumi:"services"          yaml:"services"`
	Networks compose.Networks `pulumi:"networks,optional" yaml:"networks,omitempty"`
	// Domain is the delegate domain for the project (e.g. "example.com"). When non-empty,
	// a wildcard certificate and DNS zone are created.
	Domain string `pulumi:"domain,optional" yaml:"domain,omitempty"`
}

// ProjectOutputs holds the outputs of the Project component.
type ProjectOutputs struct {
	pulumi.ResourceState

	// Per-service endpoint URLs (service name -> URL)
	Endpoints pulumi.StringMapOutput `pulumi:"endpoints"`

	// Load balancer DNS name (unused for GCP, kept for interface compat)
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

	result, err := buildProject(ctx, name, inputs, childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build GCP resources: %w", err)
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

// Build creates all GCP resources for the project.
// The GCP provider must be passed via the parent chain (pulumi.Providers on the parent component).
func buildProject(
	ctx *pulumi.Context,
	projectName string,
	args ProjectInputs,
	parentOpt pulumi.ResourceOption,
) (*common.BuildResult, error) {
	childOpts := []pulumi.ResourceOption{parentOpt}

	config, err := providergcp.BuildGlobalConfig(ctx, projectName, args.Domain, args.Services, childOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to build GCP infrastructure: %w", err)
	}

	if err := providergcp.EnableGcpAPIs(ctx, config.GcpProject, childOpts...); err != nil {
		return nil, err
	}

	// Deploy each service, wrapped in a component resource for tree organization
	endpoints := pulumi.StringMap{}
	dependencies := map[string]pulumi.Resource{} // service name → component resource for dependees
	configProvider := providergcp.NewConfigProvider(projectName)
	var lbEntries []providergcp.LBServiceEntry

	if common.IsProjectUsingLLM(args.Services) {
		// FIXME: create dependency between this NewService and the services that need this API
		_, err := projects.NewService(ctx, projectName+"-defang-llm", &projects.ServiceArgs{
			Project: pulumi.StringPtr(config.GcpProject),
			Service: pulumi.String("aiplatform.googleapis.com"),
		}, pulumi.RetainOnDelete(true)) // Do not try disabling on compose down
		if err != nil {
			return nil, err
		}
	}

	for _, svcName := range common.TopologicalSort(args.Services) {
		svc := args.Services[svcName]

		// Collect dependency resources from services this one depends on
		var deps []pulumi.Resource
		for dep := range svc.DependsOn {
			if r, ok := dependencies[dep]; ok {
				deps = append(deps, r)
			}
		}

		endpoint, svcComp, lbEntry, err := buildService(
			ctx, projectName, configProvider, svcName, svc, config, deps, childOpts)
		if err != nil {
			return nil, err
		}
		endpoints[svcName] = endpoint
		dependencies[svcName] = svcComp
		if lbEntry != nil {
			lbEntries = append(lbEntries, *lbEntry)
		}
	}

	if providergcp.NeedNATGateway(args.Networks, args.Services) {
		if err := providergcp.CreateNAT(ctx, config.VpcId, *config, childOpts...); err != nil {
			return nil, err
		}
	}

	err = providergcp.CreateLoadBalancers(
		ctx,
		projectName,
		lbEntries,
		config,
	)
	if err != nil {
		return nil, err
	}

	return &common.BuildResult{
		Endpoints:       endpoints.ToStringMapOutput(),
		LoadBalancerDNS: pulumi.StringPtr("").ToStringPtrOutput(),
	}, nil
}

func buildService(
	ctx *pulumi.Context,
	projectName string,
	configProvider compose.ConfigProvider,
	svcName string,
	svc compose.ServiceConfig,
	infra *providergcp.GlobalConfig,
	deps []pulumi.Resource,
	childOpts []pulumi.ResourceOption,
) (pulumi.StringOutput, pulumi.Resource, *providergcp.LBServiceEntry, error) {
	svcComp := &struct{ pulumi.ResourceState }{}

	var endpoint pulumi.StringOutput
	var lbEntry *providergcp.LBServiceEntry

	svcChildOpts := childOpts
	if len(deps) > 0 {
		svcChildOpts = append(svcChildOpts, pulumi.DependsOn(deps))
	}

	switch {
	case svc.Postgres != nil:
		// Managed Postgres → Cloud SQL
		if err := ctx.RegisterComponentResource("defang-gcp:index:Postgres", svcName, svcComp, svcChildOpts...); err != nil {
			return pulumi.StringOutput{}, nil, nil, fmt.Errorf("registering Cloud SQL component %s: %w", svcName, err)
		}
		svcOpts := []pulumi.ResourceOption{pulumi.Parent(svcComp)}

		sqlResult, err := providergcp.CreateCloudSQL(ctx, configProvider, svcName, svc, infra, svcOpts...)
		if err != nil {
			return pulumi.StringOutput{}, nil, nil, fmt.Errorf("creating Cloud SQL for %s: %w", svcName, err)
		}

		lbEntry = &providergcp.LBServiceEntry{Name: svcName, PostgresInstance: sqlResult.Instance, Config: svc}
		port := firstPort(svc.Ports, defaultPostgresPort)
		endpoint = pulumi.Sprintf("%s:%d", sqlResult.Instance.PublicIpAddress, port)
	case svc.Redis != nil:
		// Managed Redis → Memorystore
		if err := ctx.RegisterComponentResource("defang-gcp:index:Redis", svcName, svcComp, svcChildOpts...); err != nil {
			return pulumi.StringOutput{}, nil, nil, fmt.Errorf("registering Memorystore component %s: %w", svcName, err)
		}
		svcOpts := []pulumi.ResourceOption{pulumi.Parent(svcComp)}

		redisResult, err := providergcp.CreateMemoryStore(ctx, svcName, svc, infra, svcOpts...)
		if err != nil {
			return pulumi.StringOutput{}, nil, nil, fmt.Errorf("creating Memorystore for %s: %w", svcName, err)
		}
		lbEntry = &providergcp.LBServiceEntry{Name: svcName, RedisInstance: redisResult.Instance, Config: svc}
		endpoint = pulumi.Sprintf("%s:%d", redisResult.Instance.Host, firstPort(svc.Ports, defaultRedisPort))
	default:
		if err := ctx.RegisterComponentResource("defang-gcp:index:Service", svcName, svcComp, svcChildOpts...); err != nil {
			return pulumi.StringOutput{}, nil, nil, fmt.Errorf("registering Service component %s: %w", svcName, err)
		}
		image, err := providergcp.GetServiceImage(ctx, svcName, svc, infra.BuildInfra, svcChildOpts...)
		if err != nil {
			return pulumi.StringOutput{}, nil, nil, fmt.Errorf("resolving image for %s: %w", svcName, err)
		}

		var extraDeps []pulumi.ResourceOption
		if len(deps) > 0 {
			extraDeps = []pulumi.ResourceOption{pulumi.DependsOn(deps)}
		}
		endpoint, lbEntry, err = BuildContainerService(
			ctx, projectName, configProvider, svcName, image, svc, infra, svcComp, extraDeps...)
		if err != nil {
			return pulumi.StringOutput{}, nil, nil, err
		}
	}
	if lbEntry != nil && svc.HasHostPorts() {
		lbEntry.PrivateFqdn = fmt.Sprintf("%s.%s", svcName, "google.internal")
	}

	if err := ctx.RegisterResourceOutputs(svcComp, pulumi.Map{
		"endpoint": endpoint,
	}); err != nil {
		return pulumi.StringOutput{}, nil, nil, fmt.Errorf("registering outputs for %s: %w", svcName, err)
	}

	return endpoint, svcComp, lbEntry, nil
}

// BuildContainerService creates Cloud Run or Compute Engine resources for a service.
// svcComp must already be registered as a component resource before calling this function.
func BuildContainerService(
	ctx *pulumi.Context,
	projectName string,
	configProvider compose.ConfigProvider,
	svcName string,
	image pulumi.StringInput,
	svc compose.ServiceConfig,
	infra *providergcp.GlobalConfig,
	svcComp pulumi.Resource,
	extraOpts ...pulumi.ResourceOption,
) (pulumi.StringOutput, *providergcp.LBServiceEntry, error) {
	childOpts := append([]pulumi.ResourceOption{pulumi.Parent(svcComp)}, extraOpts...)

	sa, err := createServiceAccount(ctx, projectName, svcName, infra, childOpts)
	if err != nil {
		return pulumi.StringOutput{}, nil, err
	}

	if svc.LLM != nil {
		if err := enableLLM(ctx, svcName, &svc, sa, infra, childOpts); err != nil {
			return pulumi.StringOutput{}, nil, err
		}
	}

	if providergcp.IsCloudRunService(&svc) {
		// Cloud Run: single ingress port
		crResult, err := providergcp.CreateCloudRunService(
			ctx, configProvider, svcName, image, svc, sa, infra, pulumi.Parent(svcComp))
		if err != nil {
			return pulumi.StringOutput{}, nil, fmt.Errorf("creating Cloud Run service %s: %w", svcName, err)
		}
		lbEntry := &providergcp.LBServiceEntry{Name: svcName, CloudRunService: crResult.Service, Config: svc}
		return crResult.Service.Uri, lbEntry, nil
	}

	// Compute Engine: portless workers or services with host-mode ports
	ceResult, err := providergcp.CreateComputeEngine(
		ctx, projectName, svcName, image, svc, sa, infra, pulumi.Parent(svcComp),
	)
	if err != nil {
		return pulumi.StringOutput{}, nil, fmt.Errorf("creating Compute Engine service %s: %w", svcName, err)
	}
	var lbEntry *providergcp.LBServiceEntry
	if svc.HasIngressPorts() || svc.HasHostPorts() {
		lbEntry = &providergcp.LBServiceEntry{Name: svcName, InstanceGroup: ceResult.InstanceGroup, Config: svc}
	}
	return infra.PublicIP.Address.ToStringOutput(), lbEntry, nil
}

func createServiceAccount(
	ctx *pulumi.Context,
	projectName,
	svcName string,
	infra *providergcp.GlobalConfig,
	svcChildOpts []pulumi.ResourceOption,
) (*serviceaccount.Account, error) {
	displayName := fmt.Sprintf("%v service %v stack %v Service Account", projectName, infra.Stack, svcName)
	description := fmt.Sprintf(
		"Service Account used by run services of %v project %v service in %v stack",
		projectName,
		svcName,
		infra.Stack,
	)
	// Create a service account for the services running in cloudrun or compute engine
	sa, err := serviceaccount.NewAccount(ctx, projectName+"-"+svcName+"-service-account", &serviceaccount.AccountArgs{
		DisplayName: pulumi.String(displayName),
		Description: pulumi.String(description),
	}, svcChildOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating service account %s: %w", svcName, err)
	}
	return sa, nil
}

func enableLLM(
	ctx *pulumi.Context,
	svcName string,
	svc *compose.ServiceConfig,
	sa *serviceaccount.Account,
	infra *providergcp.GlobalConfig,
	svcChildOpts []pulumi.ResourceOption,
) error {
	// TODO: add dependency to the member resource
	_, err := projects.NewIAMMember(ctx, svcName+"-defang-llm", &projects.IAMMemberArgs{
		Project: pulumi.String(infra.GcpProject),
		// for details see https://cloud.google.com/vertex-ai/docs/general/access-control
		Role:   pulumi.String("roles/aiplatform.user"),
		Member: pulumi.Sprintf("serviceAccount:%v", sa.Email),
	}, append(svcChildOpts,
		// prevent service account does not exist error when down, will be automatically removed when sa is removed
		pulumi.DeletedWith(sa),
		// membership is not a distinct resource, so we risk deleting the membership we are trying to create
		pulumi.DeleteBeforeReplace(true),
	)...,
	)
	if err != nil {
		return fmt.Errorf("failed to grant aiplatform access to service account %v: %w", sa.Email, err)
	}

	if svc.Environment == nil {
		svc.Environment = make(map[string]string)
	}

	// Inject environment variables for Vercel routing for GCP Vertex AI access
	// https://ai-sdk.dev/providers/ai-sdk-providers/google-vertex
	if val, ok := svc.Environment["GOOGLE_VERTEX_PROJECT"]; !ok || val == "" {
		svc.Environment["GOOGLE_VERTEX_PROJECT"] = infra.GcpProject
	}

	if val, ok := svc.Environment["GOOGLE_VERTEX_LOCATION"]; !ok || val == "" {
		svc.Environment["GOOGLE_VERTEX_LOCATION"] = infra.Region
	}

	if val, ok := svc.Environment["VERTEX_PROJECT"]; !ok || val == "" {
		svc.Environment["VERTEX_PROJECT"] = infra.GcpProject
	}

	if val, ok := svc.Environment["VERTEX_LOCATION"]; !ok || val == "" {
		svc.Environment["VERTEX_LOCATION"] = infra.Region
	}

	// Inject environment variables for Google ADK to have access to GCP Vertex AI
	if val, ok := svc.Environment["GOOGLE_CLOUD_PROJECT"]; !ok || val == "" {
		svc.Environment["GOOGLE_CLOUD_PROJECT"] = infra.GcpProject
	}

	if val, ok := svc.Environment["GOOGLE_CLOUD_LOCATION"]; !ok || val == "" {
		svc.Environment["GOOGLE_CLOUD_LOCATION"] = infra.Region
	}
	return nil
}
