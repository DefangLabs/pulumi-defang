package defanggcp

import (
	"errors"
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providergcp "github.com/DefangLabs/pulumi-defang/provider/defanggcp/gcp"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var (
	errSidecarsRequireComputeEngine = errors.New(
		"sidecars and volumes are only supported on Compute Engine services (no single ingress port)")
	errPoliciesUnsupported = errors.New("x-defang-policies is not supported on GCP")
)

// Service is the controller struct for the defang-gcp:index:Service component.
type Service struct{}

// ServiceInputs defines the inputs for a standalone GCP Cloud Run / Compute Engine service.
// Build-from-source is deliberately unsupported — images must be pre-built and supplied
// via Image. Build orchestration belongs to the Project component, which provisions the
// shared BuildInfra (Artifact Registry + Cloud Build) needed to produce the image.
type ServiceInputs struct {
	// Image is the pre-built container image URI; it may be an Output of an
	// image build (e.g. a Build resource or a caller-side pipeline).
	Image       pulumi.StringInput          `pulumi:"image"`
	Platform    *string                     `pulumi:"platform,optional"`
	ProjectName string                      `pulumi:"projectName"`
	Ports       []compose.ServicePortConfig `pulumi:"ports,optional"`
	Deploy      *compose.DeployConfig       `pulumi:"deploy,optional"`
	Environment map[string]*string          `pulumi:"environment,optional"`
	Command     []string                    `pulumi:"command,optional"`
	Entrypoint  []string                    `pulumi:"entrypoint,optional"`
	HealthCheck *compose.HealthCheckConfig  `pulumi:"healthCheck,optional"`
	DomainName  string                      `pulumi:"domainName,optional"`
	LLM         *compose.LlmConfig          `pulumi:"llm,optional"`

	// Compose-shaped container settings (honored by the Compute Engine path)
	ContainerName   *string                       `pulumi:"containerName,optional"`
	WorkingDir      *string                       `pulumi:"workingDir,optional"`
	StopGracePeriod *string                       `pulumi:"stopGracePeriod,optional"`
	Volumes         []compose.ServiceVolumeConfig `pulumi:"volumes,optional"`
	VolumesFrom     []string                      `pulumi:"volumesFrom,optional"`
	DependsOn       compose.DependsOnConfig       `pulumi:"dependsOn,optional"`

	// Sidecars are additional containers run on the same Compute Engine
	// instance (as extra systemd units). volumesFrom / dependsOn may
	// reference sidecar names. Not supported on Cloud Run-shaped services.
	Sidecars map[string]compose.ServiceConfig `pulumi:"sidecars,optional"`
	// ServiceAccountEmail reuses an existing service account instead of
	// creating one per service; its IAM role grants are owned by the caller.
	ServiceAccountEmail pulumi.StringInput `pulumi:"serviceAccountEmail,optional"`
	// Triggers force a rolling replacement of Compute Engine instances when
	// any value changes.
	Triggers pulumi.StringMapInput `pulumi:"triggers,optional"`

	// Infra is an optional shared GCP project infrastructure. When non-nil, the
	// Service reuses it (region/VPC, build repos, LLM bindings). When nil, the
	// Service runs standalone with minimal defaults (region + project only):
	// Cloud Run without VPC access, or Compute Engine on the default network;
	// build-from-source is unsupported. Untagged
	// because SharedInfra contains Pulumi Output and resource-pointer fields
	// that aren't schema-compatible; the project dispatcher passes it in Go.
	Infra *providergcp.SharedInfra
}

// ServiceOutputs holds the outputs of a Service component.
type ServiceOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
	// ServiceAccountEmail is the email of the service account the service runs
	// as (created or caller-supplied) — e.g. for granting extra IAM roles.
	ServiceAccountEmail pulumi.StringOutput `pulumi:"serviceAccountEmail"`
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

	if inputs.Image == nil {
		return nil, fmt.Errorf("service %s: %w", name, common.ErrStandaloneServiceRequiresImage)
	}
	svc := compose.ServiceConfig{
		Platform:        inputs.Platform,
		Ports:           inputs.Ports,
		Deploy:          inputs.Deploy,
		Environment:     inputs.Environment,
		Command:         inputs.Command,
		Entrypoint:      inputs.Entrypoint,
		HealthCheck:     inputs.HealthCheck,
		DomainName:      inputs.DomainName,
		LLM:             inputs.LLM,
		ContainerName:   inputs.ContainerName,
		WorkingDir:      inputs.WorkingDir,
		StopGracePeriod: inputs.StopGracePeriod,
		Volumes:         inputs.Volumes,
		VolumesFrom:     inputs.VolumesFrom,
		DependsOn:       inputs.DependsOn,
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

	image := inputs.Image

	extras := &serviceExtras{
		ServiceAccountEmail: inputs.ServiceAccountEmail,
		Sidecars:            inputs.Sidecars,
		Triggers:            inputs.Triggers,
	}
	if err := createService(ctx, comp, projectName, configProvider, name, image, svc, infra, extras); err != nil {
		return nil, err
	}
	return comp, nil
}

// serviceExtras carries standalone-only Service inputs into the shared worker.
// The project dispatcher passes nil (compose files cannot express these).
type serviceExtras struct {
	ServiceAccountEmail pulumi.StringInput
	Sidecars            map[string]compose.ServiceConfig
	Triggers            pulumi.StringMapInput
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
	extras *serviceExtras,
) error {
	parentOpt := pulumi.Parent(comp)
	childOpts := []pulumi.ResourceOption{parentOpt}
	if extras == nil {
		extras = &serviceExtras{}
	}
	if len(svc.Policies) > 0 {
		return fmt.Errorf("service %s: %w", serviceName, errPoliciesUnsupported)
	}
	for name, sc := range extras.Sidecars {
		if sc.Image == nil || *sc.Image == "" {
			return fmt.Errorf("service %s sidecar %s: %w", serviceName, name, common.ErrStandaloneServiceRequiresImage)
		}
	}

	var identity *providergcp.ServiceIdentity
	if extras.ServiceAccountEmail != nil {
		identity = &providergcp.ServiceIdentity{Email: extras.ServiceAccountEmail}
	} else {
		sa, err := createServiceAccount(ctx, projectName, serviceName, infra, childOpts)
		if err != nil {
			return err
		}
		identity = &providergcp.ServiceIdentity{Account: sa, Email: sa.Email}
	}

	if svc.LLM != nil {
		if err := enableLLM(ctx, serviceName, &svc, identity, infra, childOpts); err != nil {
			return err
		}
	}

	var endpoint pulumi.StringOutput
	var lbEntry *providergcp.LBServiceEntry

	// Cloud Run for single-ingress-port services; everything else (portless
	// workers, host ports, multiple ports) runs on Compute Engine — unless
	// there's no shared project infra, where the legacy standalone behavior of
	// forcing Cloud Run is preserved. Sidecars/volumes are CE-only, so they
	// override the standalone forcing (CE then attaches to the default network)
	// but are an error on services whose port shape demands Cloud Run.
	needsCE := len(extras.Sidecars) > 0 || len(svc.Volumes) > 0 || len(svc.VolumesFrom) > 0
	useCloudRun := providergcp.IsCloudRunService(&svc)
	if !useCloudRun && infra.PublicIP == nil && !needsCE {
		useCloudRun = true // legacy: standalone CE was unsupported (no shared VPC)
	}
	if useCloudRun {
		if needsCE {
			return fmt.Errorf("service %s: %w", serviceName, errSidecarsRequireComputeEngine)
		}
		crResult, crErr := providergcp.CreateCloudRunService(
			ctx, configProvider, serviceName, image, svc, identity, infra, parentOpt,
		)
		if crErr != nil {
			return fmt.Errorf("creating Cloud Run service %s: %w", serviceName, crErr)
		}
		endpoint = crResult.Service.Uri
		lbEntry = &providergcp.LBServiceEntry{Name: serviceName, CloudRunService: crResult.Service, Config: svc}
	} else {
		ceArgs := &providergcp.ComputeEngineArgs{
			SA:       identity,
			Sidecars: extras.Sidecars,
			Triggers: extras.Triggers,
		}
		ceResult, ceErr := providergcp.CreateComputeEngine(
			ctx, serviceName, image, svc, ceArgs, infra, parentOpt,
		)
		if ceErr != nil {
			return fmt.Errorf("creating Compute Engine service %s: %w", serviceName, ceErr)
		}
		if infra.PublicIP != nil {
			endpoint = infra.PublicIP.Address.ToStringOutput()
		} else {
			// standalone CE has no load balancer; portless workers have no endpoint
			endpoint = pulumi.String("").ToStringOutput()
		}
		if svc.HasIngressPorts() || svc.HasHostPorts() {
			lbEntry = &providergcp.LBServiceEntry{Name: serviceName, InstanceGroup: ceResult.InstanceGroup, Config: svc}
		}
	}

	if lbEntry != nil && svc.HasHostPorts() {
		lbEntry.PrivateFqdn = fmt.Sprintf("%s.%s", common.ServiceLabel(serviceName), "google.internal")
	}

	saEmail := identity.Email.ToStringOutput()
	comp.Endpoint = endpoint
	comp.ServiceAccountEmail = saEmail
	comp.LBEntry = lbEntry

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint":            endpoint,
		"serviceAccountEmail": saEmail,
	}); err != nil {
		return fmt.Errorf("registering outputs for %s: %w", serviceName, err)
	}
	return nil
}
