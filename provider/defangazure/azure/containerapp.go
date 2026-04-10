package azure

import (
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/app/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// postgresURLEndpoint checks whether v is a postgres(ql):// URL whose host matches a managed
// service in serviceEndpoints. If so it returns a new URL with:
//   - the hostname replaced by the managed service FQDN
//   - any ${VAR} credential/query references interpolated from configProvider
func postgresURLEndpoint(
	ctx *pulumi.Context,
	v string,
	serviceEndpoints map[string]pulumi.StringOutput,
	configProvider *ConfigProvider,
) (pulumi.StringOutput, bool) {
	var prefix string
	switch {
	case strings.HasPrefix(v, "postgresql://"):
		prefix = "postgresql://"
	case strings.HasPrefix(v, "postgres://"):
		prefix = "postgres://"
	default:
		return pulumi.StringOutput{}, false
	}

	rest := strings.TrimPrefix(v, prefix)
	// Find the last "@" to split userinfo from host
	atIdx := strings.LastIndex(rest, "@")
	if atIdx < 0 {
		return pulumi.StringOutput{}, false
	}

	hostAndRest := rest[atIdx+1:]
	// Extract hostname (stop at ':' or '/')
	host := hostAndRest
	if i := strings.IndexAny(host, ":/"); i >= 0 {
		host = host[:i]
	}

	ep, ok := serviceEndpoints[host]
	if !ok {
		return pulumi.StringOutput{}, false
	}

	// Split into: "scheme+userinfo@" and "afterhost" (port/path/query)
	beforeHost := prefix + rest[:atIdx+1]
	afterHost := hostAndRest[len(host):]

	// Interpolate ${VAR} in credentials and query parts if a config provider is available.
	var beforeOut, afterOut pulumi.StringOutput
	if configProvider != nil {
		beforeOut = compose.InterpolateEnvironmentVariable(ctx, configProvider, beforeHost)
		afterOut = compose.InterpolateEnvironmentVariable(ctx, configProvider, afterHost)
	} else {
		beforeOut = pulumi.String(beforeHost).ToStringOutput()
		afterOut = pulumi.String(afterHost).ToStringOutput()
	}

	// ep is "host:port" — use just the hostname portion.
	result := pulumi.All(beforeOut, ep, afterOut).ApplyT(func(args []any) string {
		before := args[0].(string)
		endpoint := args[1].(string)
		after := args[2].(string)
		epHost := strings.SplitN(endpoint, ":", 2)[0]
		return before + epHost + after
	}).(pulumi.StringOutput)

	return result, true
}

type containerAppResult struct {
	App *app.ContainerApp
}

// containerAppCpuMemory snaps requested CPU/memory to Azure Container Apps fixed tiers.
func containerAppCpuMemory(cpus float64, memMiB int) (float64, string) {
	memGi := float64(memMiB) / 1024.0
	options := []struct {
		cpu float64
		mem float64
	}{
		{0.25, 0.5},
		{0.5, 1.0},
		{0.75, 1.5},
		{1.0, 2.0},
		{1.25, 2.5},
		{1.5, 3.0},
		{1.75, 3.5},
		{2.0, 4.0},
	}
	for _, opt := range options {
		if cpus <= opt.cpu && memGi <= opt.mem {
			return opt.cpu, fmt.Sprintf("%.2fGi", opt.mem)
		}
	}
	return 2.0, "4.00Gi"
}

// buildEnvVars constructs the environment variable array for a Container App.
func buildEnvVars(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	infra *SharedInfra,
	serviceEndpoints map[string]pulumi.StringOutput,
) app.EnvironmentVarArray {
	envs := app.EnvironmentVarArray{
		app.EnvironmentVarArgs{
			Name:  pulumi.String("DEFANG_SERVICE"),
			Value: pulumi.String(serviceName),
		},
	}
	for k, v := range svc.Environment {
		var value pulumi.StringInput
		switch {
		case k == "OPENAI_API_KEY" && infra.LLMInfra != nil:
			// Replace the placeholder API key with the real Azure OpenAI key.
			value = infra.LLMInfra.APIKey
		case v == "" && infra.ConfigProvider != nil:
			// null/empty in compose means "read from config store" (set via `defang config set`).
			value = infra.ConfigProvider.GetConfig(ctx, k)
		default:
			if ep, ok := redisURLEndpoint(v, serviceEndpoints); ok {
				value = ep
			} else if ep, ok := llmURLEndpoint(v, serviceEndpoints); ok {
				value = ep
			} else if ep, ok := postgresURLEndpoint(ctx, v, serviceEndpoints, infra.ConfigProvider); ok {
				// Postgres URL: replace service hostname with managed FQDN and
				// interpolate ${VAR} credential references from config store.
				value = ep
			} else if infra.ConfigProvider != nil && strings.Contains(v, "${") {
				// Any other value with ${VAR} references: interpolate from config store.
				value = compose.InterpolateEnvironmentVariable(ctx, infra.ConfigProvider, v)
			} else {
				value = pulumi.String(v)
			}
		}
		envs = append(envs, app.EnvironmentVarArgs{
			Name:  pulumi.String(k),
			Value: value,
		})
	}
	return envs
}

// CreateContainerApp creates an Azure Container App.
//
// serviceEndpoints maps managed-service names (e.g. "redisx") to their connection URLs
// (e.g. "rediss://:<key>@host:10000"). Env var values that look like
// "redis://<serviceName>[:<port>]..." are automatically replaced with the real URL so
// that apps which reference a Redis or other managed service by its Compose service name
// continue to work without source changes.
func CreateContainerApp(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	infra *SharedInfra,
	imageURI pulumi.StringInput,
	serviceEndpoints map[string]pulumi.StringOutput,
	opts ...pulumi.ResourceOption,
) (*containerAppResult, error) {
	envs := buildEnvVars(ctx, serviceName, svc, infra, serviceEndpoints)

	// Resource limits
	cpu, mem := containerAppCpuMemory(svc.GetCPUs(), svc.GetMemoryMiB())

	// Scale config
	minReplicas := svc.GetReplicas()
	maxReplicas := minReplicas
	if mr := int32(MaxReplicas.Get(ctx)); mr > 0 { //nolint:gosec // config value is bounded
		maxReplicas = mr
	}

	// Ingress config
	var ingress *app.IngressArgs
	if len(svc.Ports) > 0 {
		external := false
		for _, p := range svc.Ports {
			if p.Mode == "ingress" {
				external = true
				break
			}
		}
		ingress = &app.IngressArgs{
			External:   pulumi.Bool(external),
			TargetPort: pulumi.Int(svc.Ports[0].Target),
		}
	}

	// Health check probes
	var probes app.ContainerAppProbeArray
	if svc.HealthCheck != nil && len(svc.HealthCheck.Test) > 0 && len(svc.Ports) > 0 {
		probe := app.ContainerAppProbeArgs{
			Type: pulumi.String("Liveness"),
			HttpGet: &app.ContainerAppProbeHttpGetArgs{
				Port: pulumi.Int(svc.Ports[0].Target),
				Path: pulumi.String("/"),
			},
		}
		if svc.HealthCheck.IntervalSeconds != 0 {
			probe.PeriodSeconds = pulumi.Int(svc.HealthCheck.IntervalSeconds)
		}
		if svc.HealthCheck.TimeoutSeconds != 0 {
			probe.TimeoutSeconds = pulumi.Int(svc.HealthCheck.TimeoutSeconds)
		}
		if svc.HealthCheck.Retries != 0 {
			probe.FailureThreshold = pulumi.Int(svc.HealthCheck.Retries)
		}
		if svc.HealthCheck.StartPeriodSeconds != 0 {
			probe.InitialDelaySeconds = pulumi.Int(svc.HealthCheck.StartPeriodSeconds)
		}
		probes = append(probes, probe)
	}

	// If there's a build infra (ACR), configure the Container App to pull images
	// using the pre-created user-assigned managed identity (AcrPull role).
	var registries app.RegistryCredentialsArray
	var identity *app.ManagedServiceIdentityArgs
	if infra.BuildInfra != nil {
		identityID := infra.BuildInfra.ManagedIdentityID()
		registries = app.RegistryCredentialsArray{
			app.RegistryCredentialsArgs{
				Server:   infra.BuildInfra.LoginServer(),
				Identity: identityID,
			},
		}
		identity = &app.ManagedServiceIdentityArgs{
			Type:                   pulumi.String("UserAssigned"),
			UserAssignedIdentities: pulumi.StringArray{identityID},
		}
	}

	containerApp, err := app.NewContainerApp(ctx, serviceName, &app.ContainerAppArgs{
		ResourceGroupName:    infra.ResourceGroup.Name,
		ContainerAppName:     pulumi.StringPtr(serviceName),
		ManagedEnvironmentId: infra.Environment.ID().ToStringOutput(),
		Identity:             identity,
		Configuration: &app.ConfigurationArgs{
			Ingress:    ingress,
			Registries: registries,
		},
		Template: &app.TemplateArgs{
			Scale: &app.ScaleArgs{
				MinReplicas: pulumi.Int(minReplicas),
				MaxReplicas: pulumi.Int(maxReplicas),
			},
			Containers: app.ContainerArray{
				app.ContainerArgs{
					Name:    pulumi.String(serviceName),
					Image:   imageURI,
					Command: compose.ToPulumiStringArray(svc.Entrypoint),
					Args:    compose.ToPulumiStringArray(svc.Command),
					Env:     envs,
					Probes:  probes,
					Resources: &app.ContainerResourcesArgs{
						Cpu:    pulumi.Float64(cpu),
						Memory: pulumi.String(mem),
					},
				},
			},
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Container App: %w", err)
	}

	return &containerAppResult{App: containerApp}, nil
}

// llmURLEndpoint checks whether v is an LLM gateway URL ("http://<name>/api/v1/") whose
// host matches a key in serviceEndpoints. If so, it returns the mapped (Azure OpenAI base)
// endpoint output and true.
func llmURLEndpoint(v string, serviceEndpoints map[string]pulumi.StringOutput) (pulumi.StringOutput, bool) {
	if !strings.HasPrefix(v, "http://") {
		return pulumi.StringOutput{}, false
	}
	rest := strings.TrimPrefix(v, "http://")
	host := rest
	if i := strings.IndexAny(host, ":/"); i >= 0 {
		host = host[:i]
	}
	if ep, ok := serviceEndpoints[host]; ok {
		return ep, true
	}
	return pulumi.StringOutput{}, false
}

// redisURLEndpoint checks whether v is a Redis URL ("redis://<name>...") whose host matches
// a key in serviceEndpoints. If so, it returns the mapped endpoint output and true.
// For rediss:// endpoints, ssl_cert_reqs=CERT_NONE is appended so that Celery 5.x and
// channels_redis can connect to Azure Redis Enterprise without certificate validation.
func redisURLEndpoint(v string, serviceEndpoints map[string]pulumi.StringOutput) (pulumi.StringOutput, bool) {
	if !strings.HasPrefix(v, "redis://") {
		return pulumi.StringOutput{}, false
	}
	// Extract the host from "redis://[userinfo@]host[:port][/path]"
	rest := strings.TrimPrefix(v, "redis://")
	if at := strings.LastIndex(rest, "@"); at >= 0 {
		rest = rest[at+1:]
	}
	host := rest
	if i := strings.IndexAny(host, ":/"); i >= 0 {
		host = host[:i]
	}
	if ep, ok := serviceEndpoints[host]; ok {
		ep = ep.ApplyT(func(u string) string {
			if strings.HasPrefix(u, "rediss://") {
				sep := "?"
				if strings.Contains(u, "?") {
					sep = "&"
				}
				return u + sep + "ssl_cert_reqs=none"
			}
			return u
		}).(pulumi.StringOutput)
		return ep, true
	}
	return pulumi.StringOutput{}, false
}
