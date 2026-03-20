package defanggcp

import (
	"fmt"

	providergcp "github.com/DefangLabs/pulumi-defang/provider/defanggcp/gcp"
	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// GcpCloudRunService is the controller struct for the defang-gcp:index:GcpCloudRunService component.
type GcpCloudRunService struct{}

// GcpCloudRunServiceInputs defines the inputs for a standalone GCP Cloud Run service.
type GcpCloudRunServiceInputs struct {
	Build       *shared.BuildInput        `pulumi:"build,optional"`
	Image       *string                   `pulumi:"image,optional"`
	Platform    *string                   `pulumi:"platform,optional"`
	Ports       []shared.PortConfig       `pulumi:"ports,optional"`
	Deploy      *shared.DeployConfig      `pulumi:"deploy,optional"`
	Environment map[string]*string        `pulumi:"environment,optional"`
	Command     []string                  `pulumi:"command,optional"`
	Entrypoint  []string                  `pulumi:"entrypoint,optional"`
	HealthCheck *shared.HealthCheckConfig `pulumi:"healthCheck,optional"`
	DomainName  *string                   `pulumi:"domainName,optional"`
}

// GcpCloudRunServiceOutputs holds the outputs of a GcpCloudRunService component.
type GcpCloudRunServiceOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
}

// Construct implements the ComponentResource interface for GcpCloudRunService.
func (*GcpCloudRunService) Construct(ctx *pulumi.Context, name, typ string, inputs GcpCloudRunServiceInputs, opts pulumi.ResourceOption) (*GcpCloudRunServiceOutputs, error) {
	comp := &GcpCloudRunServiceOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	childOpt := pulumi.Parent(comp)
	svc := shared.ServiceInput{
		Build:       inputs.Build,
		Image:       inputs.Image,
		Platform:    inputs.Platform,
		Ports:       inputs.Ports,
		Deploy:      inputs.Deploy,
		Environment: inputs.Environment,
		Command:     inputs.Command,
		Entrypoint:  inputs.Entrypoint,
		HealthCheck: inputs.HealthCheck,
		DomainName:  inputs.DomainName,
	}

	region := providergcp.GcpRegion(ctx)
	recipe := providergcp.LoadRecipe(ctx)
	crResult, err := providergcp.CreateCloudRunService(ctx, name, svc, region, recipe, childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build GCP Cloud Run service: %w", err)
	}

	comp.Endpoint = crResult.Service.Uri

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": crResult.Service.Uri,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}
