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

// ServiceInputs defines the inputs for a standalone Azure container service.
// Build-from-source is deliberately unsupported — images must be pre-built and supplied
// via Image. Build orchestration belongs to the Project component.
//
// TODO(azure-parity): mirror the AWS/GCP compose-shape / sidecars / triggers /
// taskRoleArn / secrets / securityGroupIds / waitForSteadyState / autoscaling
// inputs added in feat/defang-on-defang-inputs. Deferred until there is a
// concrete consumer — see CLAUDE.md § Compose-shape parity across providers.
type ServiceInputs struct {
	// Image is the pre-built container image URI; it may be an Output of an
	// image build (e.g. a Build resource or a caller-side pipeline).
	Image       pulumi.StringInput          `pulumi:"image"`
	Platform    *string                     `pulumi:"platform,optional"`
	ProjectName string                      `pulumi:"projectName,optional"`
	Ports       []compose.ServicePortConfig `pulumi:"ports,optional"`
	Deploy      *compose.DeployConfig       `pulumi:"deploy,optional"`
	Environment compose.Environment         `pulumi:"environment,optional"`
	Command     []string                    `pulumi:"command,optional"`
	Entrypoint  []string                    `pulumi:"entrypoint,optional"`
	HealthCheck *compose.HealthCheckConfig  `pulumi:"healthCheck,optional"`
	DomainName  string                      `pulumi:"domainName,optional"`
	// DnsZones maps each of the service's BYOD hostnames — its DomainName and any
	// aliases on its default network — to the ARM resource ID of the public Azure
	// DNS zone (in the current subscription) that hosts it, as resolved by the CD
	// task (cd/program/azure.go findByodZones). Hostnames present here get their
	// routing + asuid TXT records written into that customer-owned zone (BYOD),
	// enabling a managed cert; hostnames absent from it take the ACME /
	// delegate-domain path instead. Keyed by hostname, not by service, because one
	// service's hostnames need not share a zone.
	DnsZones map[string]string `pulumi:"dnsZones,optional"`

	// Infra is an optional shared Azure project infrastructure. When non-nil, the
	// Service reuses it (resource group, managed environment, networking, DNS,
	// Key Vault wiring). When nil, the Service creates the minimal infrastructure
	// for its selected runtime, without shared DNS or Key Vault wiring. Untagged
	// because SharedInfra contains Pulumi Output and resource-pointer fields
	// that aren't schema-compatible; the project dispatcher passes it in Go.
	Infra *azure.SharedInfra
}

// ServiceOutputs holds the outputs of a Service component.
type ServiceOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
	// AppID is the backing resource's ARM resource ID, surfaced through the
	// Project's serviceIds output. Untagged — not part of the SDK schema.
	AppID pulumi.StringOutput
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
	if inputs.Image == nil {
		return nil, fmt.Errorf("service %s: %w", name, common.ErrStandaloneServiceRequiresImage)
	}
	svc := compose.ServiceConfig{
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
		// Standalone path: build only the infrastructure needed by the selected
		// runtime. VM services need a VNet; Container Apps needs a managed
		// environment.
		var err error
		infra, err = newStandaloneInfra(ctx, name, azure.IsVirtualMachineService(&svc), pulumi.Parent(comp))
		if err != nil {
			return nil, err
		}
	}

	if err := createContainerService(ctx, comp, name, svc, infra, inputs.Image, nil, nil, inputs.DnsZones); err != nil {
		return nil, err
	}
	return comp, nil
}

// newStandaloneInfra builds minimal runtime-specific infrastructure for a
// Service deployed without a Project.
func newStandaloneInfra(
	ctx *pulumi.Context, name string, virtualMachine bool, parentOpt pulumi.ResourceOption,
) (*azure.SharedInfra, error) {
	rg, err := resources.NewResourceGroup(ctx, name, &resources.ResourceGroupArgs{
		// Location: pulumi.String(location),
	}, parentOpt)
	if err != nil {
		return nil, fmt.Errorf("creating resource group: %w", err)
	}
	infra := &azure.SharedInfra{ResourceGroup: rg}
	if virtualMachine {
		networking, err := azure.CreateNetworking(ctx, name, infra, parentOpt)
		if err != nil {
			return nil, fmt.Errorf("creating VM networking: %w", err)
		}
		infra.Networking = networking
		return infra, nil
	}
	env, err := azureapp.NewManagedEnvironment(ctx, name, &azureapp.ManagedEnvironmentArgs{
		ResourceGroupName: rg.Name,
		// Location:          pulumi.String(location),
	}, parentOpt)
	if err != nil {
		return nil, fmt.Errorf("creating managed environment: %w", err)
	}
	infra.Environment = env
	return infra, nil
}

// createContainerService selects the Azure runtime under an already-registered
// Service component, populates its Endpoint, and registers its outputs.
func createContainerService(
	ctx *pulumi.Context,
	comp *ServiceOutputs,
	serviceName string,
	svc compose.ServiceConfig,
	infra *azure.SharedInfra,
	imageURI pulumi.StringInput,
	managedEndpoints map[string]pulumi.StringOutput,
	serviceHosts map[string]pulumi.StringOutput,
	dnsZones map[string]string,
) error {
	// Entries a stack leaves empty ("${EXTRA:-}") normalize away, so a compose
	// file parameterized per stack still deploys here; one that is not an
	// Azure identifier gets the parse error (with the ${VAR} hint).
	policies, err := azure.ParsePolicies(svc.Policies)
	if err != nil {
		return fmt.Errorf("service %s: %w", serviceName, err)
	}
	policyIdentity, err := azure.CreatePolicyIdentity(ctx, serviceName, policies, infra, pulumi.Parent(comp))
	if err != nil {
		return fmt.Errorf("granting policies for service %s: %w", serviceName, err)
	}
	if azure.IsVirtualMachineService(&svc) {
		vmResult, err := azure.CreateVirtualMachineService(
			ctx, serviceName, imageURI, svc, infra, policyIdentity, pulumi.Parent(comp),
		)
		if err != nil {
			return fmt.Errorf("creating VM service %s: %w", serviceName, err)
		}
		comp.Endpoint = vmResult.PublicIPAddress.IpAddress.ApplyT(func(ip *string) string {
			if ip == nil {
				return ""
			}
			return *ip
		}).(pulumi.StringOutput)
		comp.AppID = vmResult.ScaleSet.ID().ToStringOutput()
	} else {
		caResult, err := azure.CreateContainerApp(
			ctx, serviceName, svc, infra, imageURI, managedEndpoints, serviceHosts, dnsZones, policyIdentity,
			pulumi.Parent(comp),
		)
		if err != nil {
			return fmt.Errorf("creating Container App %s: %w", serviceName, err)
		}
		comp.Endpoint = caResult.App.LatestRevisionFqdn.ApplyT(fqdnToHTTPS).(pulumi.StringOutput)
		comp.AppID = caResult.App.ID().ToStringOutput()
	}
	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": comp.Endpoint,
	}); err != nil {
		return fmt.Errorf("registering outputs for %s: %w", serviceName, err)
	}
	return nil
}
