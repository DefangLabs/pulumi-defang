package azure

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	armappcontainers "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v3"
	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/app/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/v3/commontypesv5"
	"github.com/pulumi/pulumi-azure-native-sdk/v3/config"
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
	configProvider compose.ConfigProvider,
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

// envResult holds the environment variables and secrets for a Container App.
type envResult struct {
	Envs    app.EnvironmentVarArray
	Secrets app.SecretArray
}

// toContainerAppSecretName converts an env var name to a Container App secret
// name (lowercase, hyphens instead of underscores).
func toContainerAppSecretName(envKey string) string {
	return strings.ToLower(strings.ReplaceAll(envKey, "_", "-"))
}

// buildEnvVars constructs the environment variable array for a Container App.
// Pure config vars (empty value in compose) get Key Vault-backed secret references
// when vault info is available; otherwise they fall back to inline values.
func buildEnvVars(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	infra *SharedInfra,
	serviceEndpoints map[string]pulumi.StringOutput,
	serviceHosts map[string]pulumi.StringOutput,
	opts ...pulumi.InvokeOption,
) envResult {
	envs := app.EnvironmentVarArray{
		app.EnvironmentVarArgs{
			Name:  pulumi.String("DEFANG_SERVICE"),
			Value: pulumi.String(serviceName),
		},
	}
	if infra.Etag != "" {
		envs = append(envs, app.EnvironmentVarArgs{
			Name:  pulumi.String("DEFANG_ETAG"),
			Value: pulumi.String(infra.Etag),
		})
	}

	var appSecrets app.SecretArray
	// Multiple env vars can reference the same secret (FOO=${X}, BAR=${X}); we
	// need one Secret entry per unique secret but one EnvironmentVar per env
	// var, so dedupe on the secret var name.
	seenSecrets := make(map[string]struct{})
	for k, v := range common.Sorted(svc.Environment) {
		if k == "OPENAI_API_KEY" && infra.LLMInfra != nil {
			envs = append(envs, app.EnvironmentVarArgs{
				Name:  pulumi.String(k),
				Value: infra.LLMInfra.APIKey,
			})
		} else if secretVar := compose.GetConfigName2(k, v); secretVar != "" && infra.ConfigProvider != nil {
			secretURL, _ := infra.ConfigProvider.GetSecretRef(ctx, secretVar, opts...)
			// If we fail to get a secret ref, fall back to an inline value so the app can still deploy.
			appSecretName := toContainerAppSecretName(secretVar)
			if _, ok := seenSecrets[appSecretName]; !ok {
				seenSecrets[appSecretName] = struct{}{}
				// Container Apps requires both KeyVaultUrl and Identity when
				// referencing a Key Vault secret — the identity is what the
				// runtime uses to authenticate the vault fetch.
				appSecrets = append(appSecrets, app.SecretArgs{
					Name:        pulumi.String(appSecretName),
					KeyVaultUrl: pulumi.String(secretURL),
					Identity:    infra.KeyVaultIdentityID,
				})
			}
			envs = append(envs, app.EnvironmentVarArgs{
				Name:      pulumi.String(k),
				SecretRef: pulumi.String(appSecretName),
			})
		} else {
			// v is guaranteed non-nil here: GetConfigName2(k, nil) returns k,
			// which took the secret-ref branch above when a ConfigProvider is
			// available. The only way to land here with nil v is
			// ConfigProvider == nil — treat as empty literal.
			var raw string
			if v != nil {
				raw = *v
			}
			var value pulumi.StringInput
			if ep, ok := redisURLEndpoint(raw, serviceEndpoints); ok {
				value = ep
			} else if ep, ok := llmURLEndpoint(raw, serviceEndpoints); ok {
				value = ep
			} else if ep, ok := postgresURLEndpoint(ctx, raw, serviceEndpoints, infra.ConfigProvider); ok {
				value = ep
			} else if host, ok := serviceHosts[raw]; ok {
				value = host
			} else {
				value = compose.InterpolateEnvironmentVariable(ctx, infra.ConfigProvider, raw)
			}
			envs = append(envs, app.EnvironmentVarArgs{
				Name:  pulumi.String(k),
				Value: value,
			})
		}
	}
	return envResult{Envs: envs, Secrets: appSecrets}
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
	serviceHosts map[string]pulumi.StringOutput,
	opts ...pulumi.ResourceOption,
) (*containerAppResult, error) {
	result := buildEnvVars(ctx, serviceName, svc, infra, serviceEndpoints, serviceHosts)

	// Resource limits
	cpu, mem := containerAppCpuMemory(svc.GetCPUs(), svc.GetMemoryMiB())

	// Scale config
	minReplicas := svc.GetReplicas()
	maxReplicas := minReplicas
	if mr := int32(MaxReplicas.Get(ctx)); mr > 0 { //nolint:gosec // config value is bounded
		maxReplicas = mr
	}

	ingress := buildIngress(svc, nil) // TODO: need top-level networks to decide whether 'default' is internal
	if ingress != nil {
		// Preserve any customDomains binding added out-of-band by
		// `defang cert generate` (BYOD) or the delegate-domain cert flow. The
		// provider never sets customDomains itself, so feeding back the live
		// value keeps the CreateOrUpdate below from dropping the binding. See
		// readLiveCustomDomains.
		ingress.CustomDomains = readLiveCustomDomains(ctx, infra, serviceName)
	}
	probes := buildProbes(svc)

	var registries app.RegistryCredentialsArray
	// Collect user-assigned identities the app needs: one for ACR pull (when
	// BuildInfra is present) and one for Key Vault reads (when secrets are
	// referenced by KeyVaultUrl). Every identity referenced by a secret or
	// registry must be declared here, or Container Apps rejects the spec.
	var userIdentities pulumi.StringArray
	if infra.BuildInfra != nil {
		identityID := infra.BuildInfra.ManagedIdentityID()
		registries = app.RegistryCredentialsArray{
			app.RegistryCredentialsArgs{
				Server:   infra.BuildInfra.LoginServer(),
				Identity: identityID,
			},
		}
		userIdentities = append(userIdentities, identityID)
	}
	if len(result.Secrets) > 0 && infra.KeyVaultIdentityID != nil {
		userIdentities = append(userIdentities, infra.KeyVaultIdentityID.ToStringPtrOutput().Elem())
	}
	var identity *commontypesv5.ManagedServiceIdentityArgs
	if len(userIdentities) > 0 {
		identity = &commontypesv5.ManagedServiceIdentityArgs{
			Type:                   pulumi.String("UserAssigned"),
			UserAssignedIdentities: userIdentities,
		}
	}

	template := &app.TemplateArgs{
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
				Env:     result.Envs,
				Probes:  probes,
				Resources: &app.ContainerResourcesArgs{
					Cpu:    pulumi.Float64(cpu),
					Memory: pulumi.String(mem),
				},
			},
		},
	}
	// Embed the etag in the revision name (e.g. "myservice--abc123def456") so
	// ContainerAppConsoleLogs_CL rows are filterable by RevisionName_s without
	// needing a separate join. Suffix max length is 64 chars; etags are 12.
	if infra.Etag != "" {
		template.RevisionSuffix = pulumi.String(infra.Etag)
	}

	containerApp, err := app.NewContainerApp(ctx, serviceName, &app.ContainerAppArgs{
		ResourceGroupName:    infra.ResourceGroup.Name,
		ContainerAppName:     pulumi.StringPtr(serviceName),
		ManagedEnvironmentId: infra.Environment.ID().ToStringOutput(),
		Identity:             identity,
		Tags:                 ServiceTags(serviceName),
		Configuration: &app.ConfigurationArgs{
			Ingress:    ingress,
			Registries: registries,
			Secrets:    result.Secrets,
		},
		Template: template,
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Container App: %w", err)
	}

	return &containerAppResult{App: containerApp}, nil
}

// resolveSubscriptionID resolves the Azure subscription for the out-of-band
// customDomains read. It prefers the azure-native:subscriptionId stack config
// (what the provider itself is configured with), then falls back to the
// ARM_SUBSCRIPTION_ID / AZURE_SUBSCRIPTION_ID env vars the Azure SDK / CLI use.
// The env fallback matters for BYOD: `defang cert generate` takes its
// subscription from the CLI driver, which isn't necessarily mirrored into the
// Pulumi stack config — but the CD task's environment carries it.
func resolveSubscriptionID(ctx *pulumi.Context) string {
	if sub := config.GetSubscriptionId(ctx); sub != "" {
		return sub
	}
	if sub := os.Getenv("ARM_SUBSCRIPTION_ID"); sub != "" {
		return sub
	}
	return os.Getenv("AZURE_SUBSCRIPTION_ID")
}

// readLiveCustomDomains fetches the Container App's current
// configuration.ingress.customDomains directly from ARM and returns them as a
// Pulumi input, so the CreateOrUpdate this provider issues *preserves* any
// binding added out-of-band — by `defang cert generate` (BYOD) or the
// delegate-domain cert flow — instead of wiping it.
//
// This replaces an earlier pulumi.IgnoreChanges("...customDomains") guard,
// which was ineffective: IgnoreChanges reuses the value from Pulumi's last
// checkpoint, and since the provider never sets customDomains that value is
// always empty — so the next deploy's PUT still dropped the binding. Reading
// the live value each deploy makes the desired state match reality.
//
// On the first deploy the app (or its resource group) doesn't exist yet; a 404
// is treated as "nothing to preserve". Any other read error is returned so the
// deploy fails loudly rather than silently clobbering the binding. With no
// subscription configured (e.g. a bare preview) the read is skipped.
func readLiveCustomDomains(ctx *pulumi.Context, infra *SharedInfra, serviceName string) app.CustomDomainArrayOutput {
	subscriptionID := resolveSubscriptionID(ctx)
	return infra.ResourceGroup.Name.ApplyT(func(rgName string) ([]app.CustomDomain, error) {
		if subscriptionID == "" {
			return nil, nil
		}
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("custom-domain preserve: build credential: %w", err)
		}
		client, err := armappcontainers.NewContainerAppsClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("custom-domain preserve: build client: %w", err)
		}
		resp, err := client.Get(ctx.Context(), rgName, serviceName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				return nil, nil // app/resource group not created yet — nothing to preserve
			}
			return nil, fmt.Errorf("custom-domain preserve: reading %s: %w", serviceName, err)
		}
		if resp.Properties == nil || resp.Properties.Configuration == nil ||
			resp.Properties.Configuration.Ingress == nil {
			return nil, nil
		}
		var domains []app.CustomDomain
		for _, cd := range resp.Properties.Configuration.Ingress.CustomDomains {
			if cd == nil || cd.Name == nil {
				continue
			}
			domain := app.CustomDomain{Name: *cd.Name, CertificateId: cd.CertificateID}
			if cd.BindingType != nil {
				bt := string(*cd.BindingType)
				domain.BindingType = &bt
			}
			domains = append(domains, domain)
		}
		return domains, nil
	}).(app.CustomDomainArrayOutput)
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

func buildIngress(svc compose.ServiceConfig, networks compose.Networks) *app.IngressArgs {
	if !svc.HasIngressPorts() {
		return nil
	}
	var ingressPort compose.ServicePortConfig
	for _, p := range svc.Ports {
		if p.IsIngress() {
			ingressPort = p
			break // TODO: support more than one ingress port
		}
	}
	return &app.IngressArgs{
		External:   pulumi.Bool(common.InPublicNetwork(networks, svc)),
		TargetPort: pulumi.Int(ingressPort.Target),
	}
}

// buildProbes returns the liveness probe(s) for a Container App.
//
// When the compose service declares a `healthcheck:`, we emit an HTTP probe
// against the path parsed from the `test:` command (e.g. CMD curl
// http://localhost:PORT/healthz) — falling back to "/" if no URL is found.
// Probe cadence (interval/timeout/retries/start period) is copied from the
// compose healthcheck.
//
// When there's no compose healthcheck, we emit an explicit TCP probe on the
// service's target port instead of returning nil. Returning nil here causes
// Azure Container Apps to inject its DEFAULT liveness probe — HTTP GET on
// `/`, which 404s for many services (Hasura is the canonical case) and puts
// the container into a crash-loop. A TCP probe just checks the listener is
// up, which is correct default behavior regardless of HTTP semantics.
func buildProbes(svc compose.ServiceConfig) app.ContainerAppProbeArray {
	if len(svc.Ports) == 0 {
		return nil
	}
	targetPort := pulumi.Int(svc.Ports[0].Target)

	if svc.HealthCheck == nil || len(svc.HealthCheck.Test) == 0 {
		// Fall back to a TCP probe so Azure's default HTTP-/ probe doesn't
		// activate. Same probe shape as Azure's default (10s/5s/5) but TCP.
		return app.ContainerAppProbeArray{
			app.ContainerAppProbeArgs{
				Type: pulumi.String("Liveness"),
				TcpSocket: &app.ContainerAppProbeTcpSocketArgs{
					Port: targetPort,
				},
			},
		}
	}

	// Use the path AND port parsed from the test command — matches what the
	// user literally wrote (e.g. `curl http://localhost:9000/healthz` probes
	// :9000, not whatever svc.Ports[0].Target happens to be). When no port
	// appears in the test URL, GetHealthCheckPathAndPort returns 80 — same
	// semantics as TS (`cd/aws/healthcheck.ts`: `parsed.port || 80`).
	path, port := compose.GetHealthCheckPathAndPort(svc.HealthCheck)
	probe := app.ContainerAppProbeArgs{
		Type: pulumi.String("Liveness"),
		HttpGet: &app.ContainerAppProbeHttpGetArgs{
			Port: pulumi.Int(port),
			Path: pulumi.String(path),
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
	return app.ContainerAppProbeArray{probe}
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
