package provider

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	providergcp "github.com/DefangLabs/pulumi-defang/provider/gcp"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// GcpService is the controller struct for the defang:index:GcpService component.
type GcpService struct{}

// GcpServiceInputs defines the inputs for a standalone GCP service component.
type GcpServiceInputs struct {
	// Build configuration
	Build *BuildInput `pulumi:"build,optional"`

	// Container image to deploy
	Image *string `pulumi:"image,optional"`

	// Target platform
	Platform *string `pulumi:"platform,optional"`

	// Port configurations
	Ports []PortConfig `pulumi:"ports,optional"`

	// Deployment configuration
	Deploy *DeployConfig `pulumi:"deploy,optional"`

	// Environment variables
	Environment map[string]string `pulumi:"environment,optional"`

	// Command to run
	Command []string `pulumi:"command,optional"`

	// Entrypoint override
	Entrypoint []string `pulumi:"entrypoint,optional"`

	// Managed Postgres configuration
	Postgres *PostgresInput `pulumi:"postgres,optional"`

	// Health check configuration
	HealthCheck *HealthCheckConfig `pulumi:"healthCheck,optional"`

	// Custom domain name
	DomainName *string `pulumi:"domainName,optional"`

	// GCP-specific configuration
	GCP *GCPConfigInput `pulumi:"gcp,optional"`
}

// GcpServiceOutputs holds the outputs of a GcpService component.
type GcpServiceOutputs struct {
	pulumi.ResourceState

	// The service endpoint URL
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
}

// Construct implements the ComponentResource interface for GcpService.
// Creates all GCP resources for a standalone service.
func (*GcpService) Construct(ctx *pulumi.Context, name, typ string, inputs GcpServiceInputs, opts pulumi.ResourceOption) (*GcpServiceOutputs, error) {
	comp := &GcpServiceOutputs{}
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
		Postgres:    toPostgres(inputs.Postgres, inputs.Image, inputs.Environment),
		HealthCheck: toHealthCheck(inputs.HealthCheck),
		DomainName:  inputs.DomainName,
	}
	if inputs.Platform != nil {
		svc.Platform = *inputs.Platform
	}

	result, err := providergcp.BuildStandalone(ctx, name, svc, toGCPConfig(inputs.GCP), childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build GCP service: %w", err)
	}

	comp.Endpoint = result.Endpoint

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": result.Endpoint,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}
