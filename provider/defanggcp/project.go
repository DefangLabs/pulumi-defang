package defanggcp

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providergcp "github.com/DefangLabs/pulumi-defang/provider/defanggcp/gcp"
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
	args := common.BuildArgs{
		Services: inputs.Services,
		Domain:   inputs.Domain,
	}

	result, err := build(ctx, name, args, childOpt)
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
func build(
	ctx *pulumi.Context,
	projectName string,
	args common.BuildArgs,
	parentOpt pulumi.ResourceOption,
) (*common.BuildResult, error) {
	childOpts := []pulumi.ResourceOption{parentOpt}

	infra, err := providergcp.BuildGlobalConfig(ctx, projectName, args.Domain, args.Services, childOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to build GCP infrastructure: %w", err)
	}

	// Deploy each service, wrapped in a component resource for tree organization
	endpoints := pulumi.StringMap{}
	dependencies := map[string]pulumi.Resource{} // service name → component resource for dependees
	configProvider := providergcp.NewConfigProvider(projectName)
	var lbEntries []providergcp.LBServiceEntry

	for _, svcName := range common.TopologicalSort(args.Services) {
		svc := args.Services[svcName]

		// Collect dependency resources from services this one depends on
		var deps []pulumi.Resource
		for dep := range svc.DependsOn {
			if r, ok := dependencies[dep]; ok {
				deps = append(deps, r)
			}
		}

		endpoint, svcComp, lbEntry, err := buildService(ctx, configProvider, svcName, svc, infra, deps, childOpts)
		if err != nil {
			return nil, err
		}
		endpoints[svcName] = endpoint
		dependencies[svcName] = svcComp
		if lbEntry != nil {
			lbEntries = append(lbEntries, *lbEntry)
		}
	}

	if err := providergcp.CreateExternalLoadBalancer(
		ctx, projectName, infra, lbEntries, childOpts...,
	); err != nil {
		return nil, fmt.Errorf("creating external load balancer: %w", err)
	}

	return &common.BuildResult{
		Endpoints:       endpoints.ToStringMapOutput(),
		LoadBalancerDNS: pulumi.StringPtr("").ToStringPtrOutput(),
	}, nil
}

func buildService(
	ctx *pulumi.Context,
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

		sqlResult, err := providergcp.CreateCloudSQL(ctx, configProvider, svcName, svc, svcOpts...)
		if err != nil {
			return pulumi.StringOutput{}, nil, nil, fmt.Errorf("creating Cloud SQL for %s: %w", svcName, err)
		}
		endpoint = pulumi.Sprintf("%s:5432", sqlResult.Instance.PublicIpAddress)
	default:
		// Container service → Cloud Run
		if err := ctx.RegisterComponentResource("defang-gcp:index:Service", svcName, svcComp, svcChildOpts...); err != nil {
			return pulumi.StringOutput{}, nil, nil, fmt.Errorf("registering Cloud Run component %s: %w", svcName, err)
		}
		svcOpts := []pulumi.ResourceOption{pulumi.Parent(svcComp)}

		crResult, err := providergcp.CreateCloudRunService(ctx, configProvider, svcName, svc, infra, svcOpts...)
		if err != nil {
			return pulumi.StringOutput{}, nil, nil, fmt.Errorf("creating Cloud Run service %s: %w", svcName, err)
		}
		endpoint = crResult.Service.Uri
		lbEntry = &providergcp.LBServiceEntry{
			Name:    svcName,
			Service: crResult.Service,
			Config:  svc,
		}
	}

	if err := ctx.RegisterResourceOutputs(svcComp, pulumi.Map{
		"endpoint": endpoint,
	}); err != nil {
		return pulumi.StringOutput{}, nil, nil, fmt.Errorf("registering outputs for %s: %w", svcName, err)
	}

	return endpoint, svcComp, lbEntry, nil
}
