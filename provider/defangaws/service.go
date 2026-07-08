package defangaws

import (
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

// Service is the controller struct for the defang-aws:index:Service component.
type Service struct{}

// ServiceInputs defines the inputs for a standalone AWS ECS service.
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

	// Compose-shaped container settings
	ContainerName   *string                       `pulumi:"containerName,optional"`
	WorkingDir      *string                       `pulumi:"workingDir,optional"`
	StopGracePeriod *string                       `pulumi:"stopGracePeriod,optional"`
	Volumes         []compose.ServiceVolumeConfig `pulumi:"volumes,optional"`
	VolumesFrom     []string                      `pulumi:"volumesFrom,optional"`
	DependsOn       compose.DependsOnConfig       `pulumi:"dependsOn,optional"`
	Autoscaling     bool                          `pulumi:"autoscaling,optional"`
	// Policies are extra IAM policies (full ARN or customer-managed policy
	// name) attached to the task role created for this service. Cannot be
	// combined with TaskRoleArn — the caller owns that role's policies.
	Policies []string `pulumi:"policies,optional"`

	// Sidecars are additional containers deployed in the same ECS task.
	// volumesFrom / dependsOn may reference sidecar names.
	Sidecars map[string]compose.ServiceConfig `pulumi:"sidecars,optional"`
	// Secrets maps container environment variable names to SSM parameter or
	// Secrets Manager ARNs, injected via the ECS-native secrets mechanism.
	Secrets pulumi.StringMapInput `pulumi:"secrets,optional"`
	// TaskRoleArn reuses an existing IAM task role instead of creating one
	// per service; its policies are owned by the caller.
	TaskRoleArn pulumi.StringInput `pulumi:"taskRoleArn,optional"`
	// SecurityGroupIds are extra security groups attached to the service's ENI.
	SecurityGroupIds pulumi.StringArrayInput `pulumi:"securityGroupIds,optional"`
	// Triggers force a service redeployment when any value changes.
	Triggers pulumi.StringMapInput `pulumi:"triggers,optional"`
	// WaitForSteadyState makes the deployment wait until the ECS service is stable.
	WaitForSteadyState bool `pulumi:"waitForSteadyState,optional"`

	AWS *provideraws.SharedInfra `pulumi:"aws,optional"`
}

// ServiceOutputs holds the outputs of a Service component.
type ServiceOutputs struct {
	pulumi.ResourceState
	Endpoint pulumix.Output[string] `pulumi:"endpoint"`
	// ServiceName is the physical ECS service name — e.g. for wiring external
	// autoscaling policies.
	ServiceName pulumix.Output[string] `pulumi:"serviceName"`
	// ClusterName is the ECS cluster name the service runs in.
	ClusterName pulumix.Output[string] `pulumi:"clusterName"`
	// TaskRoleArn is the ARN of the task role (created or caller-supplied) —
	// e.g. for attaching extra IAM policies.
	TaskRoleArn pulumix.Output[string] `pulumi:"taskRoleArn"`
	// Dependency is an internal-only handle (the ECS service) used by downstream
	// services for ordering. Untagged — not part of the SDK schema.
	Dependency pulumi.Resource
}

// ServiceComponentType is the Pulumi resource type token for the Service component.
const ServiceComponentType = "defang-aws:index:Service"

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
		Platform:        inputs.Platform,
		Ports:           inputs.Ports,
		Deploy:          inputs.Deploy,
		Environment:     inputs.Environment,
		Command:         inputs.Command,
		Entrypoint:      inputs.Entrypoint,
		HealthCheck:     inputs.HealthCheck,
		DomainName:      inputs.DomainName,
		ContainerName:   inputs.ContainerName,
		WorkingDir:      inputs.WorkingDir,
		StopGracePeriod: inputs.StopGracePeriod,
		Volumes:         inputs.Volumes,
		VolumesFrom:     inputs.VolumesFrom,
		DependsOn:       inputs.DependsOn,
		Autoscaling:     inputs.Autoscaling,
		Policies:        inputs.Policies,
	}

	var configProvider compose.ConfigProvider
	if ctx.DryRun() {
		configProvider = &compose.DryRunConfigProvider{}
	} else {
		configProvider = provideraws.NewConfigProvider(inputs.ProjectName)
	}
	infra := inputs.AWS

	args := &provideraws.ECSServiceArgs{
		Infra:              infra,
		ImageURI:           inputs.Image,
		WaitForSteadyState: inputs.WaitForSteadyState,
		TaskRoleArn:        inputs.TaskRoleArn,
		SecurityGroupIds:   inputs.SecurityGroupIds,
		Secrets:            inputs.Secrets,
		Sidecars:           inputs.Sidecars,
		Triggers:           inputs.Triggers,
	}
	if err := createECSService(ctx, comp, configProvider, name, svc, args, nil); err != nil {
		return nil, err
	}
	return comp, nil
}

// createECSService creates the ECS Fargate service under an already-registered
// Service component, sets its Endpoint/Dependency, and registers its outputs.
// Shared between Construct and the project-level dispatcher.
func createECSService(
	ctx *pulumi.Context,
	comp *ServiceOutputs,
	configProvider compose.ConfigProvider,
	serviceName string,
	svc compose.ServiceConfig,
	args *provideraws.ECSServiceArgs,
	deps []pulumi.Resource,
) error {
	ecsResult, err := provideraws.CreateECSService(
		ctx, configProvider, serviceName, svc, args, deps, pulumi.Parent(comp),
	)
	if err != nil {
		return fmt.Errorf("creating ECS service %s: %w", serviceName, err)
	}

	endpoint := pulumi.StringOutput(ecsResult.Endpoint)
	comp.Endpoint = pulumix.Output[string](endpoint)
	comp.Dependency = ecsResult.Service

	ecsServiceName := ecsResult.Service.Name
	clusterName := ecsResult.Service.Cluster.ApplyT(func(arn string) string {
		// arn:aws:ecs:region:account:cluster/NAME → NAME
		return arn[strings.LastIndexByte(arn, '/')+1:]
	}).(pulumi.StringOutput)
	taskRoleArn := ecsResult.TaskRoleArn.ToStringOutput()
	comp.ServiceName = pulumix.Output[string](ecsServiceName)
	comp.ClusterName = pulumix.Output[string](clusterName)
	comp.TaskRoleArn = pulumix.Output[string](taskRoleArn)

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint":    endpoint,
		"serviceName": ecsServiceName,
		"clusterName": clusterName,
		"taskRoleArn": taskRoleArn,
	}); err != nil {
		return fmt.Errorf("registering outputs for %s: %w", serviceName, err)
	}
	return nil
}
