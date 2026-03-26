package defangaws

import (
	"fmt"

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
	Build       *compose.BuildConfig        `pulumi:"build,optional"`
	Image       *string                     `pulumi:"image,optional"`
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
}

// Construct implements the ComponentResource interface for Service.
func (*Service) Construct(
	ctx *pulumi.Context, name, typ string, inputs ServiceInputs, opts pulumi.ResourceOption,
) (*ServiceOutputs, error) {
	comp := &ServiceOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	childOpt := pulumi.Parent(comp)
	svc := compose.ServiceConfig{
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
	infra := inputs.AWS
	// infra, err := provideraws.BuildSharedInfra(ctx, name, svc, inputs.AWS, childOpt)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to build AWS ECS infrastructure: %w", err)
	// }

	var imageInfra *provideraws.BuildInfra
	if infra != nil {
		imageInfra = infra.BuildInfra
	}
	imageURI, err := provideraws.GetServiceImage(ctx, name, svc, imageInfra, childOpt)
	if err != nil {
		return nil, fmt.Errorf("resolving image for %s: %w", name, err)
	}

	ecsResult, err := NewECSServiceComponent(ctx, configProvider, name, svc, &provideraws.ECSServiceArgs{
		Infra:    infra,
		ImageURI: imageURI,
	}, nil, childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to create ECS service: %w", err)
	}

	endpoint := ecsResult.Endpoint
	comp.Endpoint = pulumix.Output[string](endpoint)

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": endpoint,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}

// serviceComponent is a local component resource used to group per-service resources in the tree.
type serviceComponent struct {
	pulumi.ResourceState
}

type ECSResult struct {
	Endpoint   pulumi.StringOutput
	Dependency pulumi.Resource // the ECS service, for dependees
}

// NewECSServiceComponent registers a component resource for a container service,
// creates its ECS children, registers outputs, and returns the endpoint.
func NewECSServiceComponent(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	serviceName string,
	svc compose.ServiceConfig,
	args *provideraws.ECSServiceArgs,
	deps []pulumi.Resource,
	parentOpt pulumi.ResourceOption,
) (*ECSResult, error) {
	comp := &serviceComponent{}
	if err := ctx.RegisterComponentResource("defang-aws:index:Service", serviceName, comp, parentOpt); err != nil {
		return nil, fmt.Errorf("registering ECS service component %s: %w", serviceName, err)
	}
	opts := []pulumi.ResourceOption{pulumi.Parent(comp)}

	ecsResult, err := provideraws.CreateECSService(ctx, configProvider, serviceName, svc, args, deps, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ECS service %s: %w", serviceName, err)
	}

	endpoint := pulumi.StringOutput(ecsResult.Endpoint)

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{"endpoint": endpoint}); err != nil {
		return nil, fmt.Errorf("registering outputs for %s: %w", serviceName, err)
	}
	return &ECSResult{
		Endpoint:   endpoint,
		Dependency: ecsResult.Service,
	}, nil
}
