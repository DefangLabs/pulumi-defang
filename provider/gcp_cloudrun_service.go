package provider

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	providergcp "github.com/DefangLabs/pulumi-defang/provider/gcp"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// GcpCloudRunService is the controller struct for the defang:index:GcpCloudRunService component.
type GcpCloudRunService struct{}

// GcpCloudRunServiceInputs defines the inputs for a standalone GCP Cloud Run service.
type GcpCloudRunServiceInputs struct {
	Build       *BuildInput        `pulumi:"build,optional"`
	Image       *string            `pulumi:"image,optional"`
	Platform    *string            `pulumi:"platform,optional"`
	Ports       []PortConfig       `pulumi:"ports,optional"`
	Deploy      *DeployConfig      `pulumi:"deploy,optional"`
	Environment map[string]string  `pulumi:"environment,optional"`
	Command     []string           `pulumi:"command,optional"`
	Entrypoint  []string           `pulumi:"entrypoint,optional"`
	HealthCheck *HealthCheckConfig `pulumi:"healthCheck,optional"`
	DomainName  *string            `pulumi:"domainName,optional"`
	GCP         *GCPConfigInput    `pulumi:"gcp,optional"`
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
	svc := common.ServiceConfig{
		Build:       toBuild(inputs.Build),
		Image:       inputs.Image,
		Ports:       toPorts(inputs.Ports),
		Deploy:      toDeploy(inputs.Deploy),
		Environment: inputs.Environment,
		Command:     inputs.Command,
		Entrypoint:  inputs.Entrypoint,
		HealthCheck: toHealthCheck(inputs.HealthCheck),
		DomainName:  inputs.DomainName,
	}
	if inputs.Platform != nil {
		svc.Platform = *inputs.Platform
	}

	result, err := providergcp.BuildStandaloneCloudRun(ctx, name, svc, toGCPConfig(inputs.GCP), childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build GCP Cloud Run service: %w", err)
	}

	comp.Endpoint = result.Endpoint

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": result.Endpoint,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}
