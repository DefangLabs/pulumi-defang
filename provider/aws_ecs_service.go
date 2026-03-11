package provider

import (
	"fmt"

	provideraws "github.com/DefangLabs/pulumi-defang/provider/aws"
	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// AwsEcsService is the controller struct for the defang:index:AwsEcsService component.
type AwsEcsService struct{}

// AwsEcsServiceInputs defines the inputs for a standalone AWS ECS service.
type AwsEcsServiceInputs struct {
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
	AWS         *AWSConfigInput    `pulumi:"aws,optional"`
}

// AwsEcsServiceOutputs holds the outputs of an AwsEcsService component.
type AwsEcsServiceOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
}

// Construct implements the ComponentResource interface for AwsEcsService.
func (*AwsEcsService) Construct(ctx *pulumi.Context, name, typ string, inputs AwsEcsServiceInputs, opts pulumi.ResourceOption) (*AwsEcsServiceOutputs, error) {
	comp := &AwsEcsServiceOutputs{}
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

	result, err := provideraws.BuildStandaloneECS(ctx, name, svc, toAWSConfig(inputs.AWS), childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build AWS ECS service: %w", err)
	}

	comp.Endpoint = result.Endpoint

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": result.Endpoint,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}
