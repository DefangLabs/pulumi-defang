package defangazure

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/DefangLabs/pulumi-defang/provider/defangazure/azure"
	azureapp "github.com/pulumi/pulumi-azure-native-sdk/app/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Service is the controller struct for the defang-azure:index:Service component.
type Service struct{}

// AzureContainerAppInputs defines the inputs for a standalone Azure Container App.
// Build-from-source is deliberately unsupported — images must be pre-built and supplied
// via Image. Build orchestration belongs to the Project component.
type AzureContainerAppInputs struct {
	Image       string                      `pulumi:"image"`
	Platform    *string                     `pulumi:"platform,optional"`
	Ports       []compose.ServicePortConfig `pulumi:"ports,optional"`
	Deploy      *compose.DeployConfig       `pulumi:"deploy,optional"`
	Environment map[string]*string          `pulumi:"environment,optional"`
	Command     []string                    `pulumi:"command,optional"`
	Entrypoint  []string                    `pulumi:"entrypoint,optional"`
	HealthCheck *compose.HealthCheckConfig  `pulumi:"healthCheck,optional"`
	DomainName  string                      `pulumi:"domainName,optional"`
}

// AzureContainerAppOutputs holds the outputs of an AzureContainerApp component.
type AzureContainerAppOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
}

// Construct implements the ComponentResource interface for AzureContainerApp.
func (*Service) Construct(
	ctx *pulumi.Context, name, typ string, inputs AzureContainerAppInputs, opts pulumi.ResourceOption,
) (*AzureContainerAppOutputs, error) {
	comp := &AzureContainerAppOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	// Standalone Service is image-only — build belongs to Project.
	if inputs.Image == "" {
		return nil, fmt.Errorf("service %s: %w", name, common.ErrStandaloneServiceRequiresImage)
	}
	childOpt := pulumi.Parent(comp)
	svc := compose.ServiceConfig{
		Image:       &inputs.Image,
		Platform:    (inputs.Platform),
		Ports:       inputs.Ports,
		Deploy:      inputs.Deploy,
		Environment: inputs.Environment,
		Command:     inputs.Command,
		Entrypoint:  inputs.Entrypoint,
		HealthCheck: inputs.HealthCheck,
		DomainName:  inputs.DomainName,
	}

	location := azure.Location(ctx)

	rg, err := resources.NewResourceGroup(ctx, name, &resources.ResourceGroupArgs{
		Location: pulumi.String(location),
	}, childOpt)
	if err != nil {
		return nil, fmt.Errorf("creating resource group: %w", err)
	}

	env, err := azureapp.NewManagedEnvironment(ctx, name, &azureapp.ManagedEnvironmentArgs{
		ResourceGroupName: rg.Name,
		Location:          pulumi.String(location),
	}, childOpt)
	if err != nil {
		return nil, fmt.Errorf("creating managed environment: %w", err)
	}

	infra := &azure.SharedInfra{ResourceGroup: rg, Environment: env}

	imageURI := pulumi.String(inputs.Image).ToStringOutput()

	caResult, err := azure.CreateContainerApp(ctx, name, svc, infra, imageURI, nil, nil, childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build Azure Container App: %w", err)
	}

	comp.Endpoint = caResult.App.LatestRevisionFqdn.ApplyT(func(fqdn string) string {
		if fqdn != "" {
			return "https://" + fqdn
		}
		return ""
	}).(pulumi.StringOutput)

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": comp.Endpoint,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}
