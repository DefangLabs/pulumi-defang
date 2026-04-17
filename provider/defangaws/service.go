package defangaws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

// Service is the controller struct for the defang-aws:index:Service component.
type Service struct{}

// ServiceInputs defines the inputs for a standalone AWS ECS service.
// Scalar fields use pulumi.String / *pulumi.String so the generated SDK
// wraps them in pulumi.Input (matching the Node.js SDK behaviour).
type ServiceInputs struct {
	Image       string                      `pulumi:"image"`
	Platform    *string                     `pulumi:"platform,optional"`
	ProjectName string                      `pulumi:"project_name"`
	Ports       []compose.ServicePortConfig `pulumi:"ports,optional"`
	Deploy      *compose.DeployConfig       `pulumi:"deploy,optional"`
	Environment map[string]string           `pulumi:"environment,optional"`
	Command     []string                    `pulumi:"command,optional"`
	Entrypoint  []string                    `pulumi:"entrypoint,optional"`
	HealthCheck *compose.HealthCheckConfig  `pulumi:"healthCheck,optional"`
	DomainName  string                      `pulumi:"domainName,optional"`

	AWS *provideraws.SharedInfra `pulumi:"aws,optional"`
}

// ServiceOutputs holds the outputs of a Service component.
type ServiceOutputs struct {
	pulumi.ResourceState
	Endpoint pulumix.Output[string] `pulumi:"endpoint"`
	// Dependency is an internal-only handle (the ECS service) used by downstream
	// services for ordering. Untagged — not part of the SDK schema.
	Dependency pulumi.Resource
}

// ServiceComponentType is the Pulumi resource type token for the Service component.
const ServiceComponentType = "defang-aws:index:Service"

// Construct implements the ComponentResource interface for Service.
func (*Service) Construct(
	ctx *pulumi.Context, name, typ string, inputs ServiceInputs, opts pulumi.ResourceOption,
) (*ServiceOutputs, error) {
	comp := &ServiceOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	// Standalone Service is image-only — build belongs to Project.
	if inputs.Image == "" {
		return nil, fmt.Errorf("service %s: %w", name, common.ErrStandaloneServiceRequiresImage)
	}
	svc := compose.ServiceConfig{
		Image:       &inputs.Image,
		Platform:    inputs.Platform,
		Ports:       inputs.Ports,
		Deploy:      inputs.Deploy,
		Environment: inputs.Environment,
		Command:     inputs.Command,
		Entrypoint:  inputs.Entrypoint,
		HealthCheck: inputs.HealthCheck,
		DomainName:  inputs.DomainName,
	}

	var configProvider compose.ConfigProvider
	if ctx.DryRun() {
		configProvider = &compose.DryRunConfigProvider{}
	} else {
		configProvider = provideraws.NewConfigProvider(inputs.ProjectName)
	}
	infra := inputs.AWS
	imageURI := pulumi.String(inputs.Image).ToStringOutput()

	args := &provideraws.ECSServiceArgs{
		Infra:    infra,
		ImageURI: imageURI,
	}
	if err := createECSService(ctx, comp, configProvider, name, svc, args, nil); err != nil {
		return nil, err
	}
	return comp, nil
}

// createECSService creates the ECS Fargate service under an already-registered
// Service component, sets its Endpoint/Dependency, and registers its outputs.
// Shared between Construct and the project-level dispatcher.
func createECSService(
	ctx *pulumi.Context,
	comp *ServiceOutputs,
	configProvider compose.ConfigProvider,
	serviceName string,
	svc compose.ServiceConfig,
	args *provideraws.ECSServiceArgs,
	deps []pulumi.Resource,
) error {
	ecsResult, err := provideraws.CreateECSService(
		ctx, configProvider, serviceName, svc, args, deps, pulumi.Parent(comp),
	)
	if err != nil {
		return fmt.Errorf("creating ECS service %s: %w", serviceName, err)
	}

	endpoint := pulumi.StringOutput(ecsResult.Endpoint)
	comp.Endpoint = pulumix.Output[string](endpoint)
	comp.Dependency = ecsResult.Service

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{"endpoint": endpoint}); err != nil {
		return fmt.Errorf("registering outputs for %s: %w", serviceName, err)
	}
	return nil
}
