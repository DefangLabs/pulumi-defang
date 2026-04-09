package defanggcp

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providergcp "github.com/DefangLabs/pulumi-defang/provider/defanggcp/gcp"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Service is the controller struct for the defang-gcp:index:Service component.
type Service struct{}

// ServiceInputs defines the inputs for a standalone GCP Cloud Run service.
type ServiceInputs struct {
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
}

// ServiceOutputs holds the outputs of a Service component.
type ServiceOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
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

	configProvider := providergcp.NewConfigProvider(inputs.ProjectName)
	services := map[string]compose.ServiceConfig{name: svc}
	infra, err := providergcp.BuildGlobalConfig(ctx, inputs.ProjectName, "", services, childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build GCP infrastructure: %w", err)
	}
	image, err := providergcp.GetServiceImage(ctx, name, svc, infra.BuildInfra, childOpt)
	if err != nil {
		return nil, fmt.Errorf("resolving image for %s: %w", name, err)
	}
	sa, err := createServiceAccount(ctx, inputs.ProjectName, name, infra, []pulumi.ResourceOption{childOpt})
	if err != nil {
		return nil, fmt.Errorf("failed to create service account: %w", err)
	}
	_, _, err = BuildContainerService(
		ctx, inputs.ProjectName, configProvider, name, image, svc, infra, comp, []pulumi.ResourceOption{childOpt},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build container service: %w", err)
	}
	crResult, err := providergcp.CreateCloudRunService(ctx, configProvider, name, image, svc, sa, infra, childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build GCP Cloud Run service: %w", err)
	}

	comp.Endpoint = crResult.Service.Uri

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": crResult.Service.Uri,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}
