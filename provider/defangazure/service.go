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

// ServiceInputs defines the inputs for a standalone Azure Container App.
// Build-from-source is deliberately unsupported — images must be pre-built and supplied
// via Image. Build orchestration belongs to the Project component.
type ServiceInputs struct {
	Image       string                      `pulumi:"image"`
	Platform    *string                     `pulumi:"platform,optional"`
	ProjectName string                      `pulumi:"projectName"`
	Ports       []compose.ServicePortConfig `pulumi:"ports,optional"`
	Deploy      *compose.DeployConfig       `pulumi:"deploy,optional"`
	Environment map[string]*string          `pulumi:"environment,optional"`
	Command     []string                    `pulumi:"command,optional"`
	Entrypoint  []string                    `pulumi:"entrypoint,optional"`
	HealthCheck *compose.HealthCheckConfig  `pulumi:"healthCheck,optional"`
	DomainName  string                      `pulumi:"domainName,optional"`

	// Infra is an optional shared Azure project infrastructure. When non-nil, the
	// Service reuses it (resource group, managed environment, networking, DNS,
	// Key Vault wiring). When nil, the Service runs standalone with a fresh
	// resource group and managed environment and no VNet/DNS/KV wiring. Untagged
	// because SharedInfra contains Pulumi Output and resource-pointer fields
	// that aren't schema-compatible; the project dispatcher passes it in Go.
	Infra *azure.SharedInfra
}

// ServiceOutputs holds the outputs of a Service component.
type ServiceOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
}

// ServiceComponentType is the Pulumi resource type token for the Service component.
const ServiceComponentType = "defang-azure:index:Service"

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

	infra := inputs.Infra
	if infra == nil {
		// Standalone path: build minimal infra (RG + ManagedEnvironment) since
		// there's no project-shared infra to reuse. No networking, DNS, log
		// analytics, or Key Vault wiring.
		var err error
		infra, err = newStandaloneInfra(ctx, name, pulumi.Parent(comp))
		if err != nil {
			return nil, err
		}
	}

	imageURI := pulumi.String(inputs.Image).ToStringOutput()
	if err := createContainerApp(ctx, comp, name, svc, infra, imageURI, nil, nil); err != nil {
		return nil, err
	}
	return comp, nil
}

// newStandaloneInfra builds a minimal SharedInfra (ResourceGroup +
// ManagedEnvironment) for a Service deployed without a Project.
func newStandaloneInfra(
	ctx *pulumi.Context, name string, parentOpt pulumi.ResourceOption,
) (*azure.SharedInfra, error) {
	rg, err := resources.NewResourceGroup(ctx, name, &resources.ResourceGroupArgs{
		// Location: pulumi.String(location),
	}, parentOpt)
	if err != nil {
		return nil, fmt.Errorf("creating resource group: %w", err)
	}
	env, err := azureapp.NewManagedEnvironment(ctx, name, &azureapp.ManagedEnvironmentArgs{
		ResourceGroupName: rg.Name,
		// Location:          pulumi.String(location),
	}, parentOpt)
	if err != nil {
		return nil, fmt.Errorf("creating managed environment: %w", err)
	}
	return &azure.SharedInfra{ResourceGroup: rg, Environment: env}, nil
}

// createContainerApp creates the Azure Container App under an already-registered
// Service component, populates its Endpoint, and registers its outputs. Shared
// between Construct and the project-level dispatcher.
func createContainerApp(
	ctx *pulumi.Context,
	comp *ServiceOutputs,
	serviceName string,
	svc compose.ServiceConfig,
	infra *azure.SharedInfra,
	imageURI pulumi.StringInput,
	managedEndpoints map[string]pulumi.StringOutput,
	serviceHosts map[string]pulumi.StringOutput,
) error {
	caResult, err := azure.CreateContainerApp(
		ctx, serviceName, svc, infra, imageURI, managedEndpoints, serviceHosts, pulumi.Parent(comp),
	)
	if err != nil {
		return fmt.Errorf("creating Container App %s: %w", serviceName, err)
	}
	comp.Endpoint = caResult.App.LatestRevisionFqdn.ApplyT(fqdnToHTTPS).(pulumi.StringOutput)
	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": comp.Endpoint,
	}); err != nil {
		return fmt.Errorf("registering outputs for %s: %w", serviceName, err)
	}
	return nil
}
