package defangaws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

// AwsEcsService is the controller struct for the defang-aws:index:AwsEcsService component.
type AwsEcsService struct{}

// AwsEcsServiceInputs defines the inputs for a standalone AWS ECS service.
type AwsEcsServiceInputs struct {
	Build       *shared.BuildInput        `pulumi:"build,optional"`
	Image       *string                   `pulumi:"image,optional"`
	Platform    *string                   `pulumi:"platform,optional"`
	ProjectName string                    `pulumi:"project_name"`
	Ports       []shared.PortConfig       `pulumi:"ports,optional"`
	Deploy      *shared.DeployConfig      `pulumi:"deploy,optional"`
	Environment map[string]*string        `pulumi:"environment,optional"`
	Command     []string                  `pulumi:"command,optional"`
	Entrypoint  []string                  `pulumi:"entrypoint,optional"`
	HealthCheck *shared.HealthCheckConfig `pulumi:"healthCheck,optional"`
	DomainName  *string                   `pulumi:"domainName,optional"`
	AWS         *shared.AWSConfigInput    `pulumi:"aws,optional"`
}

// AwsEcsServiceOutputs holds the outputs of an AwsEcsService component.
type AwsEcsServiceOutputs struct {
	pulumi.ResourceState
	Endpoint pulumix.Output[string] `pulumi:"endpoint"`
}

// Construct implements the ComponentResource interface for AwsEcsService.
func (*AwsEcsService) Construct(ctx *pulumi.Context, name, typ string, inputs AwsEcsServiceInputs, opts pulumi.ResourceOption) (*AwsEcsServiceOutputs, error) {
	comp := &AwsEcsServiceOutputs{}
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

	configProvider := provideraws.NewConfigProvider(inputs.ProjectName)
	ecsArgs, err := provideraws.BuildECSArgs(ctx, name, svc, common.ToAWSConfig(inputs.AWS), childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build AWS ECS infrastructure: %w", err)
	}

	recipe := provideraws.LoadRecipe(ctx)
	endpoint, err := provideraws.NewECSServiceComponent(ctx, configProvider, name, svc, ecsArgs, recipe, childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to create ECS service: %w", err)
	}

	comp.Endpoint = pulumix.Output[string](endpoint)

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": endpoint,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}
