package provider

import (
	"fmt"

	provideraws "github.com/DefangLabs/pulumi-defang/provider/aws"
	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// AwsService is the controller struct for the defang:index:AwsService component.
type AwsService struct{}

// AwsServiceInputs defines the inputs for a standalone AWS service component.
type AwsServiceInputs struct {
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

	// AWS-specific configuration
	AWS *AWSConfigInput `pulumi:"aws,optional"`
}

// AwsServiceOutputs holds the outputs of an AwsService component.
type AwsServiceOutputs struct {
	pulumi.ResourceState

	// The service endpoint URL
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
}

// Construct implements the ComponentResource interface for AwsService.
// Creates all AWS resources for a standalone service (its own cluster, VPC, ALB, etc.).
func (*AwsService) Construct(ctx *pulumi.Context, name, typ string, inputs AwsServiceInputs, opts pulumi.ResourceOption) (*AwsServiceOutputs, error) {
	comp := &AwsServiceOutputs{}
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

	result, err := provideraws.BuildStandalone(ctx, name, svc, toAWSConfig(inputs.AWS), childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build AWS service: %w", err)
	}

	comp.Endpoint = result.Endpoint

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": result.Endpoint,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}
