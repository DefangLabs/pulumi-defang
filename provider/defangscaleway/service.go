package defangscaleway

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providerscaleway "github.com/DefangLabs/pulumi-defang/provider/defangscaleway/scaleway"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	scalewayconfig "github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/config"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/containers"
)

// Service is the controller struct for the defang-scaleway:index:Service component.
type Service struct{}

// ScalewayServiceInputs defines the inputs for a standalone Scaleway Serverless Container.
type ScalewayServiceInputs struct {
	Image       string                      `pulumi:"image"`
	Platform    *string                     `pulumi:"platform,optional"`
	Ports       []compose.ServicePortConfig `pulumi:"ports,optional"`
	Deploy      *compose.DeployConfig       `pulumi:"deploy,optional"`
	Environment map[string]*string          `pulumi:"environment,optional"`
	Command     []string                    `pulumi:"command,optional"`
	Entrypoint  []string                    `pulumi:"entrypoint,optional"`
	HealthCheck *compose.HealthCheckConfig  `pulumi:"healthCheck,optional"`
	DomainName  string                      `pulumi:"domainName,optional"`
	ProjectName string                      `pulumi:"projectName,optional"`
	// Scaleway is optional shared project infrastructure. It is untagged because
	// SharedInfra contains Pulumi resources and outputs that are not SDK-schema inputs.
	Scaleway *providerscaleway.SharedInfra
}

// ScalewayServiceOutputs holds the outputs of a Scaleway Service component.
type ScalewayServiceOutputs struct {
	pulumi.ResourceState
	Endpoint  pulumi.StringOutput `pulumi:"endpoint"`
	Container *containers.Container
	Domain    *containers.Domain
}

const ServiceComponentType = "defang-scaleway:index:Service"

// Construct implements the ComponentResource interface for Service.
func (*Service) Construct(
	ctx *pulumi.Context, name, typ string, inputs ScalewayServiceInputs, opts pulumi.ResourceOption,
) (*ScalewayServiceOutputs, error) {
	comp := &ScalewayServiceOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}
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
	infra := inputs.Scaleway
	if infra == nil {
		infra = providerscaleway.NewStandaloneInfra(ctx, inputs.ProjectName)
	}
	configProvider := compose.ConfigProvider(&compose.PulumiConfigProvider{})
	if ctx.DryRun() {
		configProvider = &compose.DryRunConfigProvider{}
	}
	if infra.ConfigProvider != nil {
		configProvider = infra.ConfigProvider
	}
	if err := createService(ctx, comp, configProvider, name, pulumi.String(inputs.Image), svc, infra); err != nil {
		return nil, err
	}
	return comp, nil
}

func createService(
	ctx *pulumi.Context,
	comp *ScalewayServiceOutputs,
	configProvider compose.ConfigProvider,
	serviceName string,
	image pulumi.StringInput,
	svc compose.ServiceConfig,
	infra *providerscaleway.SharedInfra,
) error {
	if infra.Namespace == nil {
		ns, err := providerscaleway.CreateContainerNamespace(ctx, serviceName, infra, pulumi.Parent(comp))
		if err != nil {
			return err
		}
		infra.Namespace = ns
	}
	if infra.Region == "" {
		infra.Region = scalewayconfig.GetRegion(ctx)
	}
	result, err := providerscaleway.CreateContainerService(
		ctx, configProvider, serviceName, image, svc, infra, pulumi.Parent(comp),
	)
	if err != nil {
		return fmt.Errorf("creating Scaleway service %s: %w", serviceName, err)
	}
	comp.Endpoint = result.Endpoint
	comp.Container = result.Container
	comp.Domain = result.Domain
	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": result.Endpoint,
	}); err != nil {
		return fmt.Errorf("registering outputs for %s: %w", serviceName, err)
	}
	return nil
}
