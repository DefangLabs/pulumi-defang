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
			ConfigProvider:     NewConfigProvider("proj", vaultURL),
			KeyVaultURL:        vaultURL,
			KeyVaultIdentityID: pulumi.String(identityID).ToStringOutput(),
		}
		svc := compose.ServiceConfig{
			Environment: map[string]string{
				"LITERAL": "plain-value",
				"SECRET":  "${CONFIG}",             // bare ref → secret entry + SecretRef
				"OTHER":   "${CONFIG}",             // same secret, second env var → shared entry
				"MIXED":   "prefix${CONFIG}suffix", // not bare → plain Value, no Secret entry
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
