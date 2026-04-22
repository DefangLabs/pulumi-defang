package gcp

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudrunv2"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/secretmanager"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

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
	gcpConfig *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*CloudRunResult, error) {
	template, secretIds := buildTemplate(ctx, configProvider, serviceName, image, svc, sa, gcpConfig)

	// Grant the service account access to each referenced secret
	iamDeps := make([]pulumi.Resource, 0, len(secretIds))
	for _, sid := range secretIds {
		member, err := secretmanager.NewSecretIamMember(ctx, serviceName+"-secret-"+sid, &secretmanager.SecretIamMemberArgs{
			SecretId: pulumi.String(sid),
			Role:     pulumi.String("roles/secretmanager.secretAccessor"),
			Member:   pulumi.Sprintf("serviceAccount:%v", sa.Email),
		}, append(opts,
			pulumi.DeletedWith(sa),
			pulumi.DeleteBeforeReplace(true),
		)...)
		if err != nil {
			return nil, fmt.Errorf("granting secret access for %s: %w", sid, err)
		}
		iamDeps = append(iamDeps, member)
	}

	// Create Cloud Run service (depends on IAM bindings)
	serviceArgs := &cloudrunv2.ServiceArgs{
		Location:           pulumi.String(gcpConfig.Region),
		Ingress:            pulumi.String(Ingress.Get(ctx)),
		InvokerIamDisabled: pulumi.Bool(true),
		DeletionProtection: pulumi.Bool(DeletionProtection.Get(ctx)),
		Template:           template,
	}
	if launchStage, err := config.New(ctx, "defang").Try("launch-stage"); err == nil && launchStage != "" {
		serviceArgs.LaunchStage = pulumi.String(launchStage)
	}
	crService, err := cloudrunv2.NewService(ctx, serviceName, serviceArgs, append(opts, pulumi.DependsOn(iamDeps))...)
	if err != nil {
		return nil, fmt.Errorf("creating Cloud Run service: %w", err)
	}

	return &CloudRunResult{
		Service: crService,
	}, nil
}

// buildEnvVars constructs Cloud Run env vars, using SecretKeyRef for secret references
// (KEY=${KEY} pattern) and plaintext for everything else. Returns the env array and
// the list of Secret Manager secret IDs that need IAM binding.
func buildEnvVars(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	serviceName string,
	svc compose.ServiceConfig,
) (cloudrunv2.ServiceTemplateContainerEnvArray, []string) {
	// Multiple env vars can reference the same secret (e.g. FOO=${X}, BAR=${X});
	// the caller creates one SecretIamMember per ID so duplicates would cause a
	// URN collision. Track seen IDs to return each only once.
	seenSecretIds := make(map[string]struct{})
	var secretIds []string

	envs := cloudrunv2.ServiceTemplateContainerEnvArray{
		&cloudrunv2.ServiceTemplateContainerEnvArgs{
			Name:  pulumi.String("DEFANG_SERVICE"),
			Value: pulumi.String(serviceName),
		},
	}
	for k, v := range svc.Environment {
		if secretVar := compose.GetConfigName(v); secretVar != "" {
			secretId, _ := configProvider.GetSecretRef(ctx, secretVar)
			envs = append(envs, &cloudrunv2.ServiceTemplateContainerEnvArgs{
				Name: pulumi.String(k),
				ValueSource: &cloudrunv2.ServiceTemplateContainerEnvValueSourceArgs{
					SecretKeyRef: &cloudrunv2.ServiceTemplateContainerEnvValueSourceSecretKeyRefArgs{
						Secret:  pulumi.String(secretId),
						Version: pulumi.String("latest"),
					},
				},
			})
			if _, ok := seenSecretIds[secretId]; !ok {
				seenSecretIds[secretId] = struct{}{}
				secretIds = append(secretIds, secretId)
			}
		} else {
			value := compose.GetConfigOrEnvValue(ctx, configProvider, svc, k, v)
			envs = append(envs, &cloudrunv2.ServiceTemplateContainerEnvArgs{
				Name:  pulumi.String(k),
				Value: value,
			})
		}
	}
	return envs, secretIds
}

// buildTemplate returns the Cloud Run service template and a list of Secret Manager
// secret IDs that the service account needs access to (for IAM binding).
func buildTemplate(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	serviceName string,
	image pulumi.StringInput,
	svc compose.ServiceConfig,
	sa *serviceaccount.Account,
	gcpConfig *SharedInfra,
) (*cloudrunv2.ServiceTemplateArgs, []string) {
	envs, secretIds := buildEnvVars(ctx, configProvider, serviceName, svc)

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

	var resourceLimits pulumi.StringMap
	if svc.HasResourceReservations() {
		resourceLimits = make(pulumi.StringMap)
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

	// Only attach VpcAccess when a full project VPC has been provisioned.
	// Standalone GlobalConfig (NewStandaloneGlobalConfig) leaves PublicIP nil to
	// signal "no VPC, skip VpcAccess" — passing a zero VpcId/SubnetId to Cloud
	// Run would otherwise produce an invalid resource.
	if gcpConfig != nil && gcpConfig.PublicIP != nil {
		template.VpcAccess = buildVpcAccess(gcpConfig)
	}

	return template, secretIds
}

func buildVpcAccess(gcpConfig *SharedInfra) *cloudrunv2.ServiceTemplateVpcAccessArgs {
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
