package provider

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/aws"
	providergcp "github.com/DefangLabs/pulumi-defang/provider/gcp"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Service is the controller struct for the defang:index:Service component.
type Service struct{}

// ServiceInputs defines the inputs for a standalone Service component.
type ServiceInputs struct {
	// Cloud provider: "aws" or "gcp"
	Provider string `pulumi:"providerId"`

	// Container image to deploy
	Image *string `pulumi:"image,optional"`

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
}

// ServiceOutputs holds the outputs of a Service component.
type ServiceOutputs struct {
	pulumi.ResourceState

	// The service endpoint URL
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
}

// Construct implements the ComponentResource interface for Service.
func (*Service) Construct(ctx *pulumi.Context, name, typ string, inputs ServiceInputs, opts pulumi.ResourceOption) (*ServiceOutputs, error) {
	comp := &ServiceOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	childOpt := pulumi.Parent(comp)
	svc := common.ServiceConfig{
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
	args := common.ServiceBuildArgs{
		Service: svc,
	}

	var result *common.ServiceBuildResult
	var err error

	switch inputs.Provider {
	case "aws":
		result, err = provideraws.BuildService(ctx, name, args, childOpt)
	case "gcp":
		result, err = providergcp.BuildService(ctx, name, args, childOpt)
	default:
		return nil, fmt.Errorf("unsupported provider %q: must be \"aws\" or \"gcp\"", inputs.Provider)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to build %s service: %w", inputs.Provider, err)
	}

	comp.Endpoint = result.Endpoint

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": result.Endpoint,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}
