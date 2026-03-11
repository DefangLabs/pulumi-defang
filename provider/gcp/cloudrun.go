package gcp

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-gcp/sdk/v8/go/gcp/cloudrunv2"
	"github.com/pulumi/pulumi-gcp/sdk/v8/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type cloudRunResult struct {
	service *cloudrunv2.Service
}

// cloudRunLimits returns CPU and memory limits for Cloud Run.
func cloudRunLimits(cpus float64, memMiB int) (string, string) {
	// Cloud Run requires at least 1 CPU for always-on
	cpu := cpus
	if cpu < 1 {
		cpu = 1
	}

	// Minimum 512Mi memory
	mem := memMiB
	if mem < 512 {
		mem = 512
	}

	return fmt.Sprintf("%g", cpu), fmt.Sprintf("%dMi", mem)
}

// createCloudRunService creates a Cloud Run service.
func createCloudRunService(
	ctx *pulumi.Context,
	serviceName string,
	svc common.ServiceConfig,
	opts ...pulumi.ResourceOption,
) (*cloudRunResult, error) {
	// Create service account
	sa, err := serviceaccount.NewAccount(ctx, serviceName, &serviceaccount.AccountArgs{
		DisplayName: pulumi.String(fmt.Sprintf("Service account for %s", serviceName)),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating service account: %w", err)
	}

	image := svc.GetImage()

	// Build environment variables
	envs := cloudrunv2.ServiceTemplateContainerEnvArray{
		&cloudrunv2.ServiceTemplateContainerEnvArgs{
			Name:  pulumi.String("DEFANG_SERVICE"),
			Value: pulumi.String(serviceName),
		},
	}
	for k, v := range svc.Environment {
		envs = append(envs, &cloudrunv2.ServiceTemplateContainerEnvArgs{
			Name:  pulumi.String(k),
			Value: pulumi.String(v),
		})
	}

	// Build port config
	var ports *cloudrunv2.ServiceTemplateContainerPortsArgs
	if len(svc.Ports) > 0 {
		ports = &cloudrunv2.ServiceTemplateContainerPortsArgs{
			ContainerPort: pulumi.Int(svc.Ports[0].Target),
		}
	}

	// Build command/args
	var commands, cmdArgs pulumi.StringArray
	for _, cmd := range svc.Entrypoint {
		commands = append(commands, pulumi.String(cmd))
	}
	for _, cmd := range svc.Command {
		cmdArgs = append(cmdArgs, pulumi.String(cmd))
	}

	// Resolve Cloud Run config with defaults
	ingress := "INGRESS_TRAFFIC_ALL"
	launchStage := "BETA"
	maxInstances := svc.GetReplicas()
	if svc.CloudRun != nil {
		ingress = svc.CloudRun.Ingress
		launchStage = svc.CloudRun.LaunchStage
		if svc.CloudRun.MaxReplicas > 0 {
			maxInstances = svc.CloudRun.MaxReplicas
		}
	}

	cpuLimit, memLimit := cloudRunLimits(svc.GetCPUs(), svc.GetMemoryMiB())

	// Build health check probes
	var startupProbe *cloudrunv2.ServiceTemplateContainerStartupProbeArgs
	if svc.HealthCheck != nil && len(svc.HealthCheck.Test) > 0 && len(svc.Ports) > 0 {
		startupProbe = &cloudrunv2.ServiceTemplateContainerStartupProbeArgs{
			HttpGet: &cloudrunv2.ServiceTemplateContainerStartupProbeHttpGetArgs{
				Path: pulumi.String("/"),
				Port: pulumi.Int(svc.Ports[0].Target),
			},
		}
		if svc.HealthCheck.IntervalSeconds != nil {
			startupProbe.PeriodSeconds = pulumi.Int(*svc.HealthCheck.IntervalSeconds)
		}
		if svc.HealthCheck.TimeoutSeconds != nil {
			startupProbe.TimeoutSeconds = pulumi.Int(*svc.HealthCheck.TimeoutSeconds)
		}
		if svc.HealthCheck.Retries != nil {
			startupProbe.FailureThreshold = pulumi.Int(*svc.HealthCheck.Retries)
		}
	}

	// Create Cloud Run service
	crService, err := cloudrunv2.NewService(ctx, serviceName, &cloudrunv2.ServiceArgs{
		Ingress:            pulumi.String(ingress),
		LaunchStage:        pulumi.String(launchStage),
		InvokerIamDisabled: pulumi.Bool(true),
		DeletionProtection: pulumi.Bool(false),
		Template: &cloudrunv2.ServiceTemplateArgs{
			Containers: cloudrunv2.ServiceTemplateContainerArray{
				&cloudrunv2.ServiceTemplateContainerArgs{
					Image:    pulumi.String(image),
					Commands: commands,
					Args:     cmdArgs,
					Ports:    ports,
					Envs:     envs,
					Resources: &cloudrunv2.ServiceTemplateContainerResourcesArgs{
						Limits: pulumi.StringMap{
							"cpu":    pulumi.String(cpuLimit),
							"memory": pulumi.String(memLimit),
						},
					},
					StartupProbe: startupProbe,
				},
			},
			ServiceAccount: sa.Email,
			Scaling: &cloudrunv2.ServiceTemplateScalingArgs{
				MaxInstanceCount: pulumi.Int(maxInstances),
			},
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Cloud Run service: %w", err)
	}

	return &cloudRunResult{
		service: crService,
	}, nil
}
