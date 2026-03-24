package gcp

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudrunv2"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var nonLowerAlphaNumericOrDash = regexp.MustCompile(`[^a-z0-9-]`)

// sanitizeAccountId returns a GCP service account ID (6-30 chars, lowercase alphanumeric + hyphens).
func sanitizeAccountId(name string) string {
	id := strings.ToLower(name)
	id = nonLowerAlphaNumericOrDash.ReplaceAllLiteralString(id, "-")
	id = strings.Trim(id, "-")
	if len(id) < 6 {
		id += "-svcacc"
	}
	if len(id) > 30 {
		id = id[:30]
	}
	return strings.TrimRight(id, "-")
}

type CloudRunResult struct {
	Service *cloudrunv2.Service
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

// CreateCloudRunService creates a Cloud Run service.
func CreateCloudRunService(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	serviceName string,
	svc compose.ServiceConfig,
	location string,
	opts ...pulumi.ResourceOption,
) (*CloudRunResult, error) {
	// Create service account (AccountId max 30 chars, must be lowercase alphanumeric + hyphens)
	sa, err := serviceaccount.NewAccount(ctx, serviceName, &serviceaccount.AccountArgs{
		AccountId:   pulumi.String(sanitizeAccountId(serviceName)),
		DisplayName: pulumi.String("Service account for " + serviceName),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating service account: %w", err)
	}

	// Build environment variables
	envs := cloudrunv2.ServiceTemplateContainerEnvArray{
		&cloudrunv2.ServiceTemplateContainerEnvArgs{
			Name:  pulumi.String("DEFANG_SERVICE"),
			Value: pulumi.String(serviceName),
		},
	}
	for k, v := range svc.Environment {
		value := compose.GetConfigOrEnvValue(ctx, configProvider, svc, k, v)
		envs = append(envs, &cloudrunv2.ServiceTemplateContainerEnvArgs{
			Name:  pulumi.String(k),
			Value: value,
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
	commands := compose.ToPulumiStringArray(svc.Entrypoint)
	cmdArgs := compose.ToPulumiStringArray(svc.Command)

	// Cloud Run config from recipe
	maxInstances := svc.GetReplicas()
	if MaxReplicas.Get(ctx) > 0 {
		maxInstances = MaxReplicas.Get(ctx)
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
		Location:           pulumi.String(location),
		Ingress:            pulumi.String(Ingress.Get(ctx)),
		LaunchStage:        pulumi.String(LaunchStage.Get(ctx)),
		InvokerIamDisabled: pulumi.Bool(true),
		DeletionProtection: pulumi.Bool(DeletionProtection.Get(ctx)),
		Template: &cloudrunv2.ServiceTemplateArgs{
			Containers: cloudrunv2.ServiceTemplateContainerArray{
				&cloudrunv2.ServiceTemplateContainerArgs{
					Image:    pulumi.String(*svc.Image),
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

	return &CloudRunResult{
		Service: crService,
	}, nil
}
