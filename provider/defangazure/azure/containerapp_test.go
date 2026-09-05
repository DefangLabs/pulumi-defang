package azure

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/app/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// azureNoopMocks is the mocks type already defined in parameters_test.go —
// referenced here; no redefinition needed.

// envVarsByName converts the output of buildEnvVars into a name→args map,
// dropping any entry whose concrete type isn't app.EnvironmentVarArgs (the
// code always appends that exact type, so this is a shape guard).
func envVarsByName(result envResult) map[string]app.EnvironmentVarArgs {
	byName := map[string]app.EnvironmentVarArgs{}
	for _, e := range result.Envs {
		args, ok := e.(app.EnvironmentVarArgs)
		if !ok {
			continue
		}
		name := args.Name.(pulumi.String)
		byName[string(name)] = args
	}
	return byName
}

// TestBuildEnvVarsEmitsSecretRefs verifies that env vars matching the bare
// ${VAR} pattern (per compose.GetConfigName) are emitted as Container App
// secret references (separate Secret entry + EnvironmentVar.SecretRef),
// NOT as inline plain values (which would leak plaintext into state).
//
// Lives at the provider package level (vs. tests/azure/) because buildEnvVars
// is package-private and this lets us supply a fully-populated SharedInfra
// without booting a Project Construct + Key Vault role-assignment chain.
func TestBuildEnvVarsEmitsSecretRefs(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		const (
			vaultURL   = "https://myvault.vault.azure.net"
			identityID = "/subscriptions/s/resourceGroups/rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/kv"
		)
		infra := &SharedInfra{
			ConfigProvider:     NewConfigProvider(vaultURL),
			KeyVaultURL:        vaultURL,
			KeyVaultIdentityID: pulumi.String(identityID).ToStringPtrOutput(),
		}
		svc := compose.ServiceConfig{
			Environment: compose.Environment{
				"LITERAL": pulumi.String("plain-value"),
				"SECRET":  pulumi.String("${CONFIG}"),             // bare ref → secret entry + SecretRef
				"OTHER":   pulumi.String("${CONFIG}"),             // same secret, second env var → shared entry
				"MIXED":   pulumi.String("prefix${CONFIG}suffix"), // not bare → plain Value, no Secret entry
			},
		}

		result := buildEnvVars(ctx, "svc", svc, infra, nil, nil)

		// Exactly one Secret entry — deduped even though two env vars reference CONFIG.
		require.Len(t, result.Secrets, 1,
			"expected exactly one Secret entry per unique referenced ConfigProvider key")

		envByName := envVarsByName(result)

		// LITERAL: has Value, no SecretRef
		literal, ok := envByName["LITERAL"]
		require.True(t, ok, "LITERAL missing")
		assert.NotNil(t, literal.Value, "LITERAL should have a Value")
		assert.Nil(t, literal.SecretRef, "LITERAL should not be a SecretRef")

		// SECRET: has SecretRef, no inline Value
		sec, ok := envByName["SECRET"]
		require.True(t, ok, "SECRET missing")
		assert.Nil(t, sec.Value,
			"secret env var must not have inline Value (would leak plaintext into state)")
		assert.NotNil(t, sec.SecretRef, "SECRET must be a SecretRef")

		// OTHER: same secret, same shape (SecretRef set, no Value)
		other, ok := envByName["OTHER"]
		require.True(t, ok, "OTHER missing")
		assert.Nil(t, other.Value)
		assert.NotNil(t, other.SecretRef)
		// Both should point at the same app-scoped secret name
		secRef := sec.SecretRef.(pulumi.String)
		otherRef := other.SecretRef.(pulumi.String)
		assert.Equal(t, string(secRef), string(otherRef),
			"two env vars pointing at the same secret should share a SecretRef")

		// MIXED: "prefix${CONFIG}suffix" is not a bare ref → plain Value (interpolated)
		mixed, ok := envByName["MIXED"]
		require.True(t, ok, "MIXED missing")
		assert.NotNil(t, mixed.Value, "MIXED should have interpolated Value")
		assert.Nil(t, mixed.SecretRef, "MIXED is not a bare ref; no SecretRef")

		return nil
	}, pulumi.WithMocks("proj", "stack", azureNoopMocks{}))
	require.NoError(t, err)
}

// TestBuildEnvVarsInjectsDefangServiceEnv verifies that the Container App's
// env array always contains DEFANG_SERVICE set to the service name — runtime
// code (health checks, log filters, telemetry) relies on it.
func TestBuildEnvVarsInjectsDefangServiceEnv(t *testing.T) {
	const serviceName = "my-service"
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		result := buildEnvVars(ctx, serviceName, compose.ServiceConfig{}, &SharedInfra{}, nil, nil)

		defang, ok := envVarsByName(result)["DEFANG_SERVICE"]
		require.True(t, ok, "DEFANG_SERVICE env var not found on Container App")
		value, ok := defang.Value.(pulumi.String)
		require.True(t, ok, "DEFANG_SERVICE should have a concrete string value")
		assert.Equal(t, serviceName, string(value),
			"DEFANG_SERVICE value should match the service name")
		return nil
	}, pulumi.WithMocks("proj", "stack", azureNoopMocks{}))
	require.NoError(t, err)
}

// TestBuildProbesClampsInitialDelay covers the Azure ceiling on a probe's
// InitialDelaySeconds. A compose file that is valid on ECS (start_period well over a
// minute) used to fail the whole deploy with HTTP 400
// ContainerAppProbeInitialDelaySecondsOutOfRange; it is now clamped instead.
func TestBuildProbesClampsInitialDelay(t *testing.T) {
	for _, tt := range []struct {
		name        string
		startPeriod int32
		want        int
	}{
		{"over the ceiling is clamped", 240, maxProbeInitialDelaySeconds},
		{"exactly at the ceiling is kept", 60, 60},
		{"under the ceiling is kept", 15, 15},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				svc := compose.ServiceConfig{
					Ports: []compose.ServicePortConfig{{Target: 5050}},
					HealthCheck: &compose.HealthCheckConfig{
						Test:               []string{"CMD", "curl", "-f", "http://localhost:5050/"},
						StartPeriodSeconds: tt.startPeriod,
					},
				}
				probes := buildProbes(ctx, "app", svc)
				require.Len(t, probes, 1)
				args, ok := probes[0].(app.ContainerAppProbeArgs)
				require.True(t, ok)
				assert.Equal(t, pulumi.Int(tt.want), args.InitialDelaySeconds)
				return nil
			}, pulumi.WithMocks("project", "stack", azureNoopMocks{}))
			require.NoError(t, err)
		})
	}
}
