package provider

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	providerazure "github.com/DefangLabs/pulumi-defang/provider/azure"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// AzureContainerApp is the controller struct for the defang:index:AzureContainerApp component.
type AzureContainerApp struct{}

// AzureContainerAppInputs defines the inputs for a standalone Azure Container App.
type AzureContainerAppInputs struct {
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
	Azure       *AzureConfigInput  `pulumi:"azure,optional"`
}

// AzureContainerAppOutputs holds the outputs of an AzureContainerApp component.
type AzureContainerAppOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
}

// Construct implements the ComponentResource interface for AzureContainerApp.
func (*AzureContainerApp) Construct(ctx *pulumi.Context, name, typ string, inputs AzureContainerAppInputs, opts pulumi.ResourceOption) (*AzureContainerAppOutputs, error) {
	comp := &AzureContainerAppOutputs{}
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

	result, err := providerazure.BuildStandaloneContainerApp(ctx, name, svc, toAzureConfig(inputs.Azure), childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build Azure Container App: %w", err)
	}

	comp.Endpoint = result.Endpoint

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": result.Endpoint,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}
