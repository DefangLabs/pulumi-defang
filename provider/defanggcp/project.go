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
	infra *providergcp.SharedInfra,
	deps []pulumi.Resource,
	childOpts []pulumi.ResourceOption,
) (pulumi.StringOutput, pulumi.Resource, *providergcp.LBServiceEntry, error) {
	var endpoint pulumi.StringOutput
	var lbEntry *providergcp.LBServiceEntry
	var svcComp pulumi.Resource

	svcChildOpts := childOpts
	if len(deps) > 0 {
		svcChildOpts = append(svcChildOpts, pulumi.DependsOn(deps))
	}

	switch {
	case svc.Postgres != nil:
		// Managed Postgres → Cloud SQL
		pgComp := &PostgresOutputs{}
		if err := ctx.RegisterComponentResource(PostgresComponentType, svcName, pgComp, svcChildOpts...); err != nil {
			return pulumi.StringOutput{}, nil, nil, fmt.Errorf("registering Cloud SQL component %s: %w", svcName, err)
		}
		if err := createPostgres(ctx, pgComp, configProvider, svcName, svc, infra); err != nil {
			return pulumi.StringOutput{}, nil, nil, err
		}
		endpoint = pgComp.Endpoint
		lbEntry = &providergcp.LBServiceEntry{Name: svcName, PostgresInstance: pgComp.Instance, Config: svc}
		svcComp = pgComp
	case svc.Redis != nil:
		// Managed Redis → Memorystore
		redisComp := &RedisOutputs{}
		if err := ctx.RegisterComponentResource(RedisComponentType, svcName, redisComp, svcChildOpts...); err != nil {
			return pulumi.StringOutput{}, nil, nil, fmt.Errorf("registering Memorystore component %s: %w", svcName, err)
		}
		if err := createRedis(ctx, redisComp, svcName, svc, infra); err != nil {
			return pulumi.StringOutput{}, nil, nil, err
		}
		endpoint = redisComp.Endpoint
		lbEntry = &providergcp.LBServiceEntry{Name: svcName, RedisInstance: redisComp.Instance, Config: svc}
		svcComp = redisComp
	default:
		svcCompTyped := &ServiceOutputs{}
		if err := ctx.RegisterComponentResource(ServiceComponentType, svcName, svcCompTyped, svcChildOpts...); err != nil {
			return pulumi.StringOutput{}, nil, nil, fmt.Errorf("registering Service component %s: %w", svcName, err)
		}
		image, err := providergcp.GetServiceImage(ctx, svcName, svc, infra.Repos, infra.BuildInfra, svcChildOpts...)
		if err != nil {
			return pulumi.StringOutput{}, nil, nil, fmt.Errorf("resolving image for %s: %w", svcName, err)
		}
		if err := createService(ctx, svcCompTyped, projectName, configProvider, svcName, image, svc, infra); err != nil {
			return pulumi.StringOutput{}, nil, nil, err
		}
		endpoint = svcCompTyped.Endpoint
		lbEntry = svcCompTyped.LBEntry
		svcComp = svcCompTyped
	}
	// Managed Postgres/Redis still need the PrivateFqdn wired here because their
	// createX workers don't build the LBEntry themselves. createService sets
	// PrivateFqdn internally for the Service case.
	if lbEntry != nil && svc.HasHostPorts() && lbEntry.PrivateFqdn == "" {
		lbEntry.PrivateFqdn = fmt.Sprintf("%s.%s", svcName, "google.internal")
	}

	return endpoint, svcComp, lbEntry, nil
}

func createServiceAccount(
	ctx *pulumi.Context,
	projectName,
	svcName string,
	infra *providergcp.SharedInfra,
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
	sa, err := serviceaccount.NewAccount(ctx, projectName+"-"+svcName, &serviceaccount.AccountArgs{
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
	infra *providergcp.SharedInfra,
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
