package defangazure

import (
	"fmt"

	providerazure "github.com/DefangLabs/pulumi-defang/provider/defangazure/azure"
	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// AzureContainerApp is the controller struct for the defang-azure:index:AzureContainerApp component.
type AzureContainerApp struct{}

// AzureContainerAppInputs defines the inputs for a standalone Azure Container App.
type AzureContainerAppInputs struct {
	Build       *shared.BuildInput        `pulumi:"build,optional"`
	Image       *string                   `pulumi:"image,optional"`
	Platform    *string                   `pulumi:"platform,optional"`
	Ports       []shared.PortConfig       `pulumi:"ports,optional"`
	Deploy      *shared.DeployConfig      `pulumi:"deploy,optional"`
	Environment map[string]string         `pulumi:"environment,optional"`
	Command     []string                  `pulumi:"command,optional"`
	Entrypoint  []string                  `pulumi:"entrypoint,optional"`
	HealthCheck *shared.HealthCheckConfig `pulumi:"healthCheck,optional"`
	DomainName  *string                   `pulumi:"domainName,optional"`
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

	result, err := providerazure.BuildStandaloneContainerApp(ctx, name, svc, childOpt)
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
