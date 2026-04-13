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

// SanitizeAccountId returns a GCP service account ID (6-30 chars, lowercase alphanumeric + hyphens).
func SanitizeAccountId(name string) string {
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
	image pulumi.StringInput,
	svc compose.ServiceConfig,
	sa *serviceaccount.Account,
	gcpConfig *GlobalConfig,
	opts ...pulumi.ResourceOption,
) (*CloudRunResult, error) {
	template := buildTemplate(ctx, configProvider, serviceName, image, svc, sa, gcpConfig)
	// Create Cloud Run service
	crService, err := cloudrunv2.NewService(ctx, serviceName, &cloudrunv2.ServiceArgs{
		Location:           pulumi.String(gcpConfig.Region),
		Ingress:            pulumi.String(Ingress.Get(ctx)),
		LaunchStage:        pulumi.String(LaunchStage.Get(ctx)),
		InvokerIamDisabled: pulumi.Bool(true),
		DeletionProtection: pulumi.Bool(DeletionProtection.Get(ctx)),
		Template:           template,
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Cloud Run service: %w", err)
	}

	return &CloudRunResult{
		Service: crService,
	}, nil
}

func buildTemplate(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	serviceName string,
	image pulumi.StringInput,
	svc compose.ServiceConfig,
	sa *serviceaccount.Account,
	gcpConfig *GlobalConfig,
) *cloudrunv2.ServiceTemplateArgs {
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
	if mr := int32(MaxReplicas.Get(ctx)); mr > 0 { //nolint:gosec // config value is bounded
		maxInstances = mr
	}

	resourceLimits := pulumi.StringMap{}
	if svc.HasResourceReservations() {
		cpuLimit, memLimit := cloudRunLimits(svc.GetCPUs(), svc.GetMemoryMiB())
		resourceLimits["cpu"] = pulumi.String(cpuLimit)
		resourceLimits["memory"] = pulumi.String(memLimit)
	}

	// Build health check probes
	var startupProbe *cloudrunv2.ServiceTemplateContainerStartupProbeArgs
	if svc.HealthCheck != nil && len(svc.HealthCheck.Test) > 0 && len(svc.Ports) > 0 {
		startupProbe = &cloudrunv2.ServiceTemplateContainerStartupProbeArgs{
			HttpGet: &cloudrunv2.ServiceTemplateContainerStartupProbeHttpGetArgs{
				Path: pulumi.String("/"),
				Port: pulumi.Int(svc.Ports[0].Target),
			},
		}
		if svc.HealthCheck.IntervalSeconds != 0 {
			startupProbe.PeriodSeconds = pulumi.Int(svc.HealthCheck.IntervalSeconds)
		}
		if svc.HealthCheck.TimeoutSeconds != 0 {
			startupProbe.TimeoutSeconds = pulumi.Int(svc.HealthCheck.TimeoutSeconds)
		}
		if svc.HealthCheck.Retries != 0 {
			startupProbe.FailureThreshold = pulumi.Int(svc.HealthCheck.Retries)
		}
	}
	template := &cloudrunv2.ServiceTemplateArgs{
		Containers: cloudrunv2.ServiceTemplateContainerArray{
			&cloudrunv2.ServiceTemplateContainerArgs{
				Image:    image,
				Commands: commands,
				Args:     cmdArgs,
				Ports:    ports,
				Envs:     envs,
				Resources: &cloudrunv2.ServiceTemplateContainerResourcesArgs{
					Limits: resourceLimits,
				},
				StartupProbe: startupProbe,
			},
		},
		MaxInstanceRequestConcurrency: pulumi.Int(80),
		ServiceAccount:                sa.Email,
		Scaling: &cloudrunv2.ServiceTemplateScalingArgs{
			MaxInstanceCount: pulumi.Int(maxInstances),
		},
	}

	if gcpConfig != nil {
		template.VpcAccess = buildVpcAccess(gcpConfig)
	}

	return template
}

func buildVpcAccess(gcpConfig *GlobalConfig) *cloudrunv2.ServiceTemplateVpcAccessArgs {
	return &cloudrunv2.ServiceTemplateVpcAccessArgs{
		Egress: pulumi.String("PRIVATE_RANGES_ONLY"),
		NetworkInterfaces: cloudrunv2.ServiceTemplateVpcAccessNetworkInterfaceArray{
			&cloudrunv2.ServiceTemplateVpcAccessNetworkInterfaceArgs{
				Network:    gcpConfig.VpcId,
				Subnetwork: gcpConfig.SubnetId,
			},
		},
	}
}
