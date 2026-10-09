package gcp

import (
	"fmt"
	"strconv"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudrunv2"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/secretmanager"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
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

// CreateCloudRunService creates a Cloud Run service. extraIAMDeps are IAM
// bindings created by the caller (e.g. x-defang-policies grants) that the
// service must wait for — the container may need them at startup.
func CreateCloudRunService(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	serviceName string,
	image pulumi.StringInput,
	svc compose.ServiceConfig,
	sa *ServiceIdentity,
	gcpConfig *SharedInfra,
	extraIAMDeps []pulumi.Resource,
	parentOpt pulumi.ResourceOrInvokeOption,
) (*CloudRunResult, error) {
	template, secretIds, composedSecrets, err := buildTemplate(
		ctx, configProvider, serviceName, image, svc, sa, gcpConfig, parentOpt)
	if err != nil {
		return nil, fmt.Errorf("building service template: %w", err)
	}

	// Grant the service account access to each referenced secret: pre-existing
	// config secrets (bare ${VAR} references) by their known ID, and secrets
	// newly created for composite values by their own SecretId output.
	iamDeps := make([]pulumi.Resource, 0, len(secretIds)+len(composedSecrets)+len(extraIAMDeps))
	iamDeps = append(iamDeps, extraIAMDeps...)
	for _, sid := range secretIds {
		opts := append([]pulumi.ResourceOption{parentOpt}, sa.deleteOpts()...)
		member, err := secretmanager.NewSecretIamMember(ctx, serviceName+"-secret-"+sid, &secretmanager.SecretIamMemberArgs{
			SecretId: pulumi.String(sid),
			Role:     pulumi.String("roles/secretmanager.secretAccessor"),
			Member:   pulumi.Sprintf("serviceAccount:%v", sa.Email),
		}, opts...)
		if err != nil {
			return nil, fmt.Errorf("granting secret access for %s: %w", sid, err)
		}
		iamDeps = append(iamDeps, member)
	}
	for _, cs := range composedSecrets {
		opts := append([]pulumi.ResourceOption{parentOpt}, sa.deleteOpts()...)
		resourceName := serviceName + "-env-secret-" + cs.key
		member, err := secretmanager.NewSecretIamMember(ctx, resourceName, &secretmanager.SecretIamMemberArgs{
			SecretId: cs.id,
			Role:     pulumi.String("roles/secretmanager.secretAccessor"),
			Member:   pulumi.Sprintf("serviceAccount:%v", sa.Email),
		}, opts...)
		if err != nil {
			return nil, fmt.Errorf("granting secret access for %s: %w", cs.key, err)
		}
		iamDeps = append(iamDeps, member)
	}

	// Create Cloud Run service (depends on IAM bindings)
	serviceArgs := &cloudrunv2.ServiceArgs{
		Location:           pulumi.String(gcpConfig.Region), // required
		Ingress:            pulumi.String(Ingress.Get(ctx)),
		InvokerIamDisabled: pulumi.Bool(true),
		DeletionProtection: pulumi.Bool(DeletionProtection.Get(ctx)),
		Template:           template,
	}
	if scaling := minInstanceScaling(svc); scaling != nil {
		serviceArgs.Scaling = scaling
	}
	if launchStage := LaunchStage.Get(ctx); launchStage != "" {
		serviceArgs.LaunchStage = pulumi.String(launchStage)
	}
	crService, err := cloudrunv2.NewService(ctx, serviceName, serviceArgs, parentOpt, pulumi.DependsOn(iamDeps))
	if err != nil {
		return nil, fmt.Errorf("creating Cloud Run service: %w", err)
	}

	return &CloudRunResult{
		Service: crService,
	}, nil
}

// minInstanceScaling keeps `deploy.replicas` instances warm, or returns nil to
// leave the service scaling to zero.
//
// `replicas` is a floor, not just a ceiling. The legacy GCP CD kept N instances
// running for `replicas: N` (defang-mvp pulumi/cd/gcp/gcpcd/cloudrun.go:76-81),
// and the other providers here still do: AWS ecs.go:970 (DesiredCount) and Azure
// containerapp.go:306 (MinReplicas), as does this provider's own Compute Engine
// path, compute.go:134 (TargetSize). Cloud Run had become the only place where
// `replicas` meant "at most", so a service could sit at zero and cold-start.
//
// Two deliberate choices, both matching the legacy CD:
//
//   - The floor is set at the SERVICE level, not on the revision template where
//     MaxInstanceCount lives. Google recommends applying minimum instances at the
//     service level and warns against combining the two levels. A service-level
//     minimum is also a pool shared across revisions, so a rolling deploy does not
//     transiently pay for a warm set per revision.
//   - It reads Deploy.Replicas directly rather than GetReplicas(), which clamps a
//     missing or zero value to 1. Only a user who actually asked for replicas gets
//     a warm instance; everyone else keeps scale-to-zero and its cost profile.
//     That also leaves `replicas: 0` scaling to zero, which is what #329 wants.
func minInstanceScaling(svc compose.ServiceConfig) *cloudrunv2.ServiceScalingArgs {
	if svc.Deploy == nil || svc.Deploy.Replicas == nil || *svc.Deploy.Replicas <= 0 {
		return nil
	}
	return &cloudrunv2.ServiceScalingArgs{
		MinInstanceCount: pulumi.Int(*svc.Deploy.Replicas),
	}
}

// composedEnvSecret is a new Secret Manager secret created for a composite
// env value (as opposed to a bare ${VAR} reference, which points at a
// pre-existing secret via ConfigProvider.GetSecretRef). id and version are
// the created resources' own Outputs rather than strings built by hand:
// referencing them gives Pulumi an automatic dependency edge (no explicit
// pulumi.DependsOn needed before Cloud Run or the IAM grant can use them),
// and referencing the concrete version — instead of the "latest" alias a bare
// ${VAR} reference uses — means a changed value (which creates a new,
// immutable version, since Secret Manager versions can't be updated in
// place) changes the Cloud Run template's own inputs, so the revision
// actually redeploys to pick it up.
type composedEnvSecret struct {
	key     string // env var key; used to name the IAM grant deterministically
	id      pulumi.StringOutput
	version pulumi.StringOutput
}

// newComposedEnvSecret stores a compose env value that embeds a config-
// provided secret (InterpolateEnvironmentVariable's hasSecret return — e.g. a
// DSN built from ${DB_PASSWORD}) as a new Secret Manager secret + version.
// The caller references it the same way a bare ${VAR} reference does
// (ValueSource.SecretKeyRef) and grants the service account IAM access to it,
// instead of the value landing as a plaintext Cloud Run env[].value, which is
// readable via a plain services.get. See DefangLabs/pulumi-defang#638
// (follow-up to the Azure fix for #637 / DefangLabs/station#198).
//
// No SecretId is set, so GCP assigns the secret's physical ID — the caller
// never needs to construct or know one.
func newComposedEnvSecret(
	ctx *pulumi.Context,
	serviceName, key string,
	value pulumi.StringOutput,
	opts ...pulumi.ResourceOption,
) (*composedEnvSecret, error) {
	resourceName := serviceName + "-" + key + "-env-secret"
	secret, err := secretmanager.NewSecret(ctx, resourceName, &secretmanager.SecretArgs{
		Replication: &secretmanager.SecretReplicationArgs{
			Auto: &secretmanager.SecretReplicationAutoArgs{},
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating secret: %w", err)
	}
	version, err := secretmanager.NewSecretVersion(ctx, resourceName+"-version", &secretmanager.SecretVersionArgs{
		Secret:     secret.Name,
		SecretData: value.ToStringPtrOutput(),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating secret version: %w", err)
	}
	return &composedEnvSecret{key: key, id: secret.SecretId, version: version.Version}, nil
}

// buildEnvVars constructs Cloud Run env vars, using SecretKeyRef for secret references
// (KEY=${KEY} pattern) and plaintext for everything else. Returns the env array, the
// list of pre-existing Secret Manager secret IDs that need IAM binding (bare ${VAR}
// references), and the secrets newly created for composite values (see
// newComposedEnvSecret), which also need IAM binding.
func buildEnvVars(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	serviceName, etag, fqdn string,
	svc compose.ServiceConfig,
	parentOpt pulumi.ResourceOrInvokeOption,
	opts ...pulumi.InvokeOption,
) (cloudrunv2.ServiceTemplateContainerEnvArray, []string, []*composedEnvSecret, error) {
	// Multiple env vars can reference the same secret (e.g. FOO=${X}, BAR=${X});
	// the caller creates one SecretIamMember per ID so duplicates would cause a
	// URN collision. Track seen IDs to return each only once.
	seenSecretIds := make(map[string]struct{})
	var secretIds []string
	var composedSecrets []*composedEnvSecret

	envs := cloudrunv2.ServiceTemplateContainerEnvArray{
		&cloudrunv2.ServiceTemplateContainerEnvArgs{
			Name:  pulumi.String("DEFANG_SERVICE"),
			Value: pulumi.String(serviceName),
		},
	}
	if etag != "" {
		envs = append(envs, &cloudrunv2.ServiceTemplateContainerEnvArgs{
			Name:  pulumi.String("DEFANG_ETAG"),
			Value: pulumi.String(etag),
		})
	}
	if fqdn != "" {
		envs = append(envs, &cloudrunv2.ServiceTemplateContainerEnvArgs{
			Name:  pulumi.String("DEFANG_FQDN"),
			Value: pulumi.String(fqdn),
		})
	}
	for k, v := range common.Sorted(svc.Environment) {
		// Cloud Run reserves the PORT env var (it injects the container port
		// itself) and rejects revisions that set it, so drop it here:
		// https://cloud.google.com/run/docs/configuring/services/containers#configure-port
		if k == "PORT" {
			sv, static := compose.StaticEnvValue(v)
			port := -1
			if static && sv != nil {
				port, _ = strconv.Atoi(*sv)
			}
			if len(svc.Ports) == 0 || port != int(svc.Ports[0].Target) {
				msg := fmt.Sprintf("service %q: the PORT environment variable is reserved on Cloud Run"+
					" and must match the container's port; the value has been ignored", serviceName)
				_ = ctx.Log.Warn(msg, nil)
			}
			continue
		}
		if secretVar := compose.GetConfigName2(k, v); secretVar != "" && configProvider != nil {
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
		} else if sv, static := compose.StaticEnvValue(v); static {
			// sv is guaranteed non-nil here: GetConfigName2(k, nil) returns k,
			// which would have taken the secret-ref branch above when a
			// configProvider is available.
			var raw string
			if sv != nil {
				raw = *sv
			}
			value, hasSecret := compose.InterpolateEnvironmentVariable(ctx, configProvider, raw, opts...)
			if hasSecret {
				// Composite value embeds a config-provided secret (e.g. a DSN
				// built from ${DB_PASSWORD}) — keep it out of the plaintext
				// Cloud Run env[].value (readable via a plain services.get)
				// by wrapping it in a new Secret Manager secret, the same way
				// a bare ${VAR} reference does. See newComposedEnvSecret and
				// DefangLabs/pulumi-defang#638.
				cs, err := newComposedEnvSecret(ctx, serviceName, k, value, parentOpt)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("creating composed secret for %q: %w", k, err)
				}
				envs = append(envs, &cloudrunv2.ServiceTemplateContainerEnvArgs{
					Name: pulumi.String(k),
					ValueSource: &cloudrunv2.ServiceTemplateContainerEnvValueSourceArgs{
						SecretKeyRef: &cloudrunv2.ServiceTemplateContainerEnvValueSourceSecretKeyRefArgs{
							Secret:  cs.id,
							Version: cs.version.ToStringPtrOutput(),
						},
					},
				})
				composedSecrets = append(composedSecrets, cs)
				continue
			}
			envs = append(envs, &cloudrunv2.ServiceTemplateContainerEnvArgs{
				Name:  pulumi.String(k),
				Value: value,
			})
		} else {
			// Dynamic (Output) values pass through as-is; interpolation and
			// secret detection only apply to static text.
			envs = append(envs, &cloudrunv2.ServiceTemplateContainerEnvArgs{
				Name:  pulumi.String(k),
				Value: v.ToStringOutput(),
			})
		}
	}
	return envs, secretIds, composedSecrets, nil
}

// buildTemplate returns the Cloud Run service template, a list of pre-existing
// Secret Manager secret IDs, and the secrets newly created for composite env
// values — both need IAM binding.
func buildTemplate(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	serviceName string,
	image pulumi.StringInput,
	svc compose.ServiceConfig,
	sa *ServiceIdentity,
	gcpConfig *SharedInfra,
	parentOpt pulumi.ResourceOrInvokeOption,
	opts ...pulumi.InvokeOption,
) (*cloudrunv2.ServiceTemplateArgs, []string, []*composedEnvSecret, error) {
	var etag string
	if gcpConfig != nil {
		etag = gcpConfig.Etag
	}
	// DEFANG_FQDN: custom domain, else public FQDN (ingress). GCP has no
	// private-FQDN fallback. See common.ServiceFQDN for the precedence.
	var domain string
	if gcpConfig != nil {
		domain = gcpConfig.Domain
	}
	fqdn := common.ServiceFQDN(serviceName, svc, domain, "")
	envs, secretIds, composedSecrets, err := buildEnvVars(
		ctx, configProvider, serviceName, etag, fqdn, svc, parentOpt, opts...)
	if err != nil {
		return nil, nil, nil, err
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
		ServiceAccount:                sa.Email.ToStringOutput(),
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

	return template, secretIds, composedSecrets, nil
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
