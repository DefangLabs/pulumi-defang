package defanggcp

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providergcp "github.com/DefangLabs/pulumi-defang/provider/defanggcp/gcp"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Service is the controller struct for the defang-gcp:index:Service component.
type Service struct{}

// ServiceInputs defines the inputs for a standalone GCP Cloud Run / Compute Engine service.
// Build-from-source is deliberately unsupported — images must be pre-built and supplied
// via Image. Build orchestration belongs to the Project component, which provisions the
// shared BuildInfra (Artifact Registry + Cloud Build) needed to produce the image.
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
	LLM         *compose.LlmConfig          `pulumi:"llm,optional"`
	// Infra is an optional shared GCP project infrastructure. When non-nil, the
	// Service reuses it (region/VPC, build repos, LLM bindings). When nil, the
	// Service runs standalone with minimal defaults (region + project only) and
	// without VPC access, Compute Engine, or build-from-source support. Untagged
	// because SharedInfra contains Pulumi Output and resource-pointer fields
	// that aren't schema-compatible; the project dispatcher passes it in Go.
	Infra *providergcp.SharedInfra
}

// ServiceOutputs holds the outputs of a Service component.
type ServiceOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
	// LBEntry is an internal-only handle used by the project dispatcher to wire
	// services into the external load balancer. Untagged — not part of the SDK schema.
	LBEntry *providergcp.LBServiceEntry
}

// ServiceComponentType is the Pulumi resource type token for the Service component.
const ServiceComponentType = "defang-gcp:index:Service"

// Construct implements the ComponentResource interface for Service.
func (*Service) Construct(
	ctx *pulumi.Context, name, typ string, inputs ServiceInputs, opts pulumi.ResourceOption,
) (*ServiceOutputs, error) {
	comp := &ServiceOutputs{}
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
		LLM:         inputs.LLM,
	}

	projectName := inputs.ProjectName
	if projectName == "" {
		projectName = name
	}
	var configProvider compose.ConfigProvider
	if ctx.DryRun() {
		configProvider = &compose.DryRunConfigProvider{}
	} else {
		configProvider = providergcp.NewConfigProvider(projectName)
	}

	infra := inputs.Infra
	if infra == nil {
		infra = providergcp.NewStandaloneGlobalConfig(ctx)
	}

	image := pulumi.String(inputs.Image)

	if err := createService(ctx, comp, projectName, configProvider, name, image, svc, infra); err != nil {
		return nil, err
	}
	return comp, nil
}

// createService creates the service account, optional LLM IAM bindings, and either
// the Cloud Run service or Compute Engine MIG under an already-registered Service
// component, populates its Endpoint/LBEntry, and registers its outputs. Shared
// between Construct and the project-level dispatcher.
func createService(
	ctx *pulumi.Context,
	comp *ServiceOutputs,
	projectName string,
	configProvider compose.ConfigProvider,
	serviceName string,
	image pulumi.StringInput,
	svc compose.ServiceConfig,
	infra *providergcp.SharedInfra,
) error {
	parentOpt := pulumi.Parent(comp)
	childOpts := []pulumi.ResourceOption{parentOpt}

	sa, err := createServiceAccount(ctx, projectName, serviceName, infra, childOpts)
	if err != nil {
		return err
	}

	if svc.LLM != nil {
		if err := enableLLM(ctx, serviceName, &svc, sa, infra, childOpts); err != nil {
			return err
		}
	}

	var endpoint pulumi.StringOutput
	var lbEntry *providergcp.LBServiceEntry

	// Compute Engine requires a project VPC/subnet/public IP; standalone (no
	// shared infra) can only deploy via Cloud Run. Force Cloud Run when no VPC
	// is available, regardless of port configuration.
	useCloudRun := providergcp.IsCloudRunService(&svc) || infra.PublicIP == nil
	if useCloudRun {
		crResult, crErr := providergcp.CreateCloudRunService(
			ctx, configProvider, serviceName, image, svc, sa, infra, parentOpt,
		)
		if crErr != nil {
			return fmt.Errorf("creating Cloud Run service %s: %w", serviceName, crErr)
		}
		endpoint = crResult.Service.Uri
		lbEntry = &providergcp.LBServiceEntry{Name: serviceName, CloudRunService: crResult.Service, Config: svc}
	} else {
		ceResult, ceErr := providergcp.CreateComputeEngine(
			ctx, projectName, serviceName, image, svc, sa, infra, parentOpt,
		)
		if ceErr != nil {
			return fmt.Errorf("creating Compute Engine service %s: %w", serviceName, ceErr)
		}
		endpoint = infra.PublicIP.Address.ToStringOutput()
		if svc.HasIngressPorts() || svc.HasHostPorts() {
			lbEntry = &providergcp.LBServiceEntry{Name: serviceName, InstanceGroup: ceResult.InstanceGroup, Config: svc}
		}
	}

	if lbEntry != nil && svc.HasHostPorts() {
		lbEntry.PrivateFqdn = fmt.Sprintf("%s.%s", serviceName, "google.internal")
	}

	comp.Endpoint = endpoint
	comp.LBEntry = lbEntry

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": endpoint,
	}); err != nil {
		return fmt.Errorf("registering outputs for %s: %w", serviceName, err)
	}
	return nil
}
