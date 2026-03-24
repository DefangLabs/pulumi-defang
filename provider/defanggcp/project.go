package defanggcp

import (
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providergcp "github.com/DefangLabs/pulumi-defang/provider/defanggcp/gcp"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/artifactregistry"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Project is the controller struct for the defang-gcp:index:Project component.
type Project struct{}

// ProjectInputs defines the top-level inputs for the GCP Project component.
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

	childOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}
	region := providergcp.GcpRegion(ctx)

	// Create Artifact Registry repository for container images
	ar, err := artifactregistry.NewRepository(ctx, "repo", &artifactregistry.RepositoryArgs{
		RepositoryId: pulumi.String(strings.ToLower(name)),
		Description:  pulumi.String("Container images for " + name),
		Format:       pulumi.String("DOCKER"),
	}, childOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating artifact registry: %w", err)
	}
	_ = ar

	// Deploy each service, wrapped in a component resource for tree organization
	endpoints := pulumi.StringMap{}
	configProvider := providergcp.NewConfigProvider(name)

	for _, svcName := range common.TopologicalSort(inputs.Services) {
		svc := inputs.Services[svcName]
		endpoint, err := buildService(ctx, configProvider, svcName, svc, region, childOpts)
		if err != nil {
			return nil, err
		}
		endpoints[svcName] = endpoint
	}

	endpointsOutput := endpoints.ToStringMapOutput()
	loadBalancerDNS := pulumi.StringPtr("").ToStringPtrOutput()

	comp.Endpoints = endpointsOutput
	comp.LoadBalancerDNS = loadBalancerDNS

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoints":       endpointsOutput,
		"loadBalancerDns": loadBalancerDNS,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}

func buildService(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	svcName string,
	svc compose.ServiceConfig,
	region string,
	childOpts []pulumi.ResourceOption,
) (pulumi.StringOutput, error) {
	svcComp := &struct{ pulumi.ResourceState }{}

	var endpoint pulumi.StringOutput

	if svc.Postgres != nil {
		// Managed Postgres → Cloud SQL
		if err := ctx.RegisterComponentResource("defang-gcp:index:Postgres", svcName, svcComp, childOpts...); err != nil {
			return pulumi.StringOutput{}, fmt.Errorf("registering Cloud SQL component %s: %w", svcName, err)
		}
		svcOpts := []pulumi.ResourceOption{pulumi.Parent(svcComp)}

		sqlResult, err := providergcp.CreateCloudSQL(ctx, configProvider, svcName, svc, svcOpts...)
		if err != nil {
			return pulumi.StringOutput{}, fmt.Errorf("creating Cloud SQL for %s: %w", svcName, err)
		}
		endpoint = pulumi.Sprintf("%s:5432", sqlResult.Instance.PublicIpAddress)
	} else {
		// Container service → Cloud Run
		if err := ctx.RegisterComponentResource("defang-gcp:index:Service", svcName, svcComp, childOpts...); err != nil {
			return pulumi.StringOutput{}, fmt.Errorf("registering Cloud Run component %s: %w", svcName, err)
		}
		svcOpts := []pulumi.ResourceOption{pulumi.Parent(svcComp)}

		crResult, err := providergcp.CreateCloudRunService(ctx, svcName, svc, region, svcOpts...)
		if err != nil {
			return pulumi.StringOutput{}, fmt.Errorf("creating Cloud Run service %s: %w", svcName, err)
		}
		endpoint = crResult.Service.Uri
	}

	if err := ctx.RegisterResourceOutputs(svcComp, pulumi.Map{
		"endpoint": endpoint,
	}); err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("registering outputs for %s: %w", svcName, err)
	}

	return endpoint, nil
}
