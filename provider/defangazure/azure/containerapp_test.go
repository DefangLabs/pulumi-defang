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

func TestBuildEnvVarsSecretNamesStableAcrossEdits(t *testing.T) {
	const (
		configEnv   = "Z_PASSWORD"
		composedEnv = "Z_URL"
	)
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		cp := NewConfigProvider("https://example.vault.azure.net")
		cp.cache["PASSWORD"] = pulumi.ToSecret(pulumi.String("password")).(pulumi.StringOutput)
		cp.fetched = true
		infra := &SharedInfra{ConfigProvider: cp}
		svc := compose.ServiceConfig{Environment: compose.Environment{
			configEnv:   pulumi.String("${PASSWORD}"),
			composedEnv: pulumi.String("prefix${PASSWORD}"),
			"Z-URL":     pulumi.String("prefix${PASSWORD}"),
			"z_url":     pulumi.String("prefix${PASSWORD}"),
		}}
		before := envVarsByName(buildEnvVars(ctx, "app", svc, infra, nil, nil, nil))
		// Normalizing case or punctuation must not merge distinct env keys.
		refs := make(map[pulumi.String]bool)
		for _, key := range []string{configEnv, composedEnv, "Z-URL", "z_url"} {
			ref := before[key].SecretRef.(pulumi.String)
			assert.False(t, refs[ref], "duplicate secret reference for %s", key)
			refs[ref] = true
		}
		// An earlier alias must not move the shared config secret; an
		// unrelated new secret must not renumber the composed secret either.
		svc.Environment["A_ALIAS"] = pulumi.String("${PASSWORD}")
		svc.Environment["B_TOKEN"] = pulumi.String("${TOKEN}")
		after := envVarsByName(buildEnvVars(ctx, "app", svc, infra, nil, nil, nil))
		for _, key := range []string{configEnv, composedEnv} {
			assert.Equal(t, before[key].SecretRef, after[key].SecretRef, "%s changed identity after insertion", key)
		}
		assert.Equal(t, after["A_ALIAS"].SecretRef, after[configEnv].SecretRef)
		delete(svc.Environment, configEnv)
		delete(svc.Environment, "B_TOKEN")
		removed := envVarsByName(buildEnvVars(ctx, "app", svc, infra, nil, nil, nil))
		assert.Equal(t, before[configEnv].SecretRef, removed["A_ALIAS"].SecretRef)
		assert.Equal(t, before[composedEnv].SecretRef, removed[composedEnv].SecretRef)
		return nil
	}, pulumi.WithMocks("project", "stack", azureNoopMocks{}))
	require.NoError(t, err)
}

// TestBuildEnvVarsEmitsSecretRefs verifies that env vars matching the bare
// ${VAR} pattern (per compose.GetConfigName) are emitted as Key Vault-backed
// Container App secret references (separate Secret entry +
// EnvironmentVar.SecretRef), NOT as inline plain values (which would leak
// plaintext into state); and that a composite value which embeds the same
// config var (not a bare match) still avoids a plaintext Value by getting a
// Container App-native (value-backed, non-Key-Vault) secret instead — see
// DefangLabs/station#198.
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
				"SECRET":  pulumi.String("${CONFIG}"),             // bare ref → Key Vault secret entry + SecretRef
				"OTHER":   pulumi.String("${CONFIG}"),             // same secret, second env var → shared entry
				"MIXED":   pulumi.String("prefix${CONFIG}suffix"), // not bare, but embeds CONFIG → computed secret + SecretRef
			},
		}

		result := buildEnvVars(ctx, "svc", svc, infra, nil, nil, nil)

		// One Key Vault-backed Secret entry (deduped across SECRET/OTHER) plus
		// one computed (value-backed) entry for MIXED.
		require.Len(t, result.Secrets, 2,
			"expected one Key-Vault-backed entry (deduped) plus one computed entry for MIXED")
		assert.True(t, result.HasKeyVaultSecret, "a Key Vault-backed secret was created")

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

		// MIXED: "prefix${CONFIG}suffix" embeds a config secret — must be a
		// SecretRef, never a plaintext Value, even though it isn't a bare match.
		mixed, ok := envByName["MIXED"]
		require.True(t, ok, "MIXED missing")
		assert.Nil(t, mixed.Value,
			"MIXED embeds a config secret and must not have inline Value")
		require.NotNil(t, mixed.SecretRef, "MIXED must be a SecretRef")

		// The MIXED secret must be value-backed (no KeyVaultUrl): buildEnvVars
		// only has the fully-interpolated string, not a fresh Key Vault entry
		// to point at.
		mixedRef := mixed.SecretRef.(pulumi.String)
		var mixedSecret app.SecretArgs
		var mixedSecretFound bool
		for _, entry := range result.Secrets {
			if s, ok := entry.(app.SecretArgs); ok && s.Name.(pulumi.String) == mixedRef {
				mixedSecret, mixedSecretFound = s, true
				break
			}
		}
		require.True(t, mixedSecretFound, "no Secret entry found for MIXED's SecretRef")
		assert.NotNil(t, mixedSecret.Value, "MIXED's secret should carry the computed value")
		assert.Nil(t, mixedSecret.KeyVaultUrl, "MIXED's secret is computed, not Key-Vault-backed")

		return nil
	}, pulumi.WithMocks("proj", "stack", azureNoopMocks{}))
	require.NoError(t, err)
}

// TestBuildEnvVarsCompositeSecretDoesNotCollideWithKeyVaultSecretName covers a
// naming collision CodeRabbit flagged on an earlier version of this fix
// (pulumi-defang#637), back when secret names were derived from user-chosen
// text (a config var name, or an env var key) — a bare ${VAR} reference and a
// composite value could derive the same name when a config var's name
// matched another env var's own key:
//
//	A: "${B}"        // bare ref → Key Vault secret named "b"
//	B: "prefix${C}"  // composite → would also derive secret name "b"
//
// Config references and composed values use separate namespaces, even when
// a config key equals an environment key.
func TestBuildEnvVarsCompositeSecretDoesNotCollideWithKeyVaultSecretName(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		const vaultURL = "https://myvault.vault.azure.net"
		cp := NewConfigProvider(vaultURL)
		cp.cache["C"] = pulumi.ToSecret(pulumi.String("c-value").ToStringOutput()).(pulumi.StringOutput)
		cp.fetched = true

		infra := &SharedInfra{ConfigProvider: cp, KeyVaultURL: vaultURL}
		svc := compose.ServiceConfig{
			Environment: compose.Environment{
				"A": pulumi.String("${B}"),
				"B": pulumi.String("prefix${C}"),
			},
		}

		result := buildEnvVars(ctx, "svc", svc, infra, nil, nil, nil)
		envByName := envVarsByName(result)

		a, ok := envByName["A"]
		require.True(t, ok, "A missing")
		require.NotNil(t, a.SecretRef, "A must be a SecretRef")

		b, ok := envByName["B"]
		require.True(t, ok, "B missing")
		require.NotNil(t, b.SecretRef, "B must be a SecretRef")

		aRef := string(a.SecretRef.(pulumi.String))
		bRef := string(b.SecretRef.(pulumi.String))
		assert.NotEqual(t, aRef, bRef,
			"A (Key Vault ref to config var B) and B (composite value) must not share a secret name")

		// B's own secret must carry its interpolated value, not A's Key Vault ref.
		var bSecret app.SecretArgs
		var bSecretFound bool
		for _, entry := range result.Secrets {
			if s, ok := entry.(app.SecretArgs); ok && s.Name.(pulumi.String) == pulumi.String(bRef) {
				bSecret, bSecretFound = s, true
				break
			}
		}
		require.True(t, bSecretFound, "no Secret entry found for B's SecretRef")
		assert.Nil(t, bSecret.KeyVaultUrl, "B's secret is computed, not Key-Vault-backed")
		require.NotNil(t, bSecret.Value)
		bSecret.Value.ToStringPtrOutput().ApplyT(func(v *string) *string {
			require.NotNil(t, v)
			assert.Equal(t, "prefixc-value", *v)
			return v
		})

		return nil
	}, pulumi.WithMocks("proj", "stack", azureNoopMocks{}))
	require.NoError(t, err)
}

// TestBuildEnvVarsDisambiguatesConfigVarNamedLikeEnvNamespace covers a deeper
// version of the same collision class: a fixed namespace prefix on the
// composite side wouldn't have been provably disjoint from the Key Vault
// side, because the Key Vault side's name came from a user-chosen config var
// name too. Both branches now use separate namespaces over a digest of
// the original key, so user-chosen prefixes cannot cross those namespaces.
func TestBuildEnvVarsDisambiguatesConfigVarNamedLikeEnvNamespace(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		const vaultURL = "https://myvault.vault.azure.net"
		cp := NewConfigProvider(vaultURL)
		cp.cache["OTHER"] = pulumi.ToSecret(pulumi.String("other-value").ToStringOutput()).(pulumi.StringOutput)
		cp.fetched = true

		infra := &SharedInfra{ConfigProvider: cp, KeyVaultURL: vaultURL}
		svc := compose.ServiceConfig{
			Environment: compose.Environment{
				"USES_ENV_DB": pulumi.String("${ENV_DB}"),      // bare ref → KV secret named "env-db"
				"DB":          pulumi.String("prefix${OTHER}"), // composite → would also derive "env-db"-shaped name
			},
		}

		result := buildEnvVars(ctx, "svc", svc, infra, nil, nil, nil)
		envByName := envVarsByName(result)

		usesEnvDB, ok := envByName["USES_ENV_DB"]
		require.True(t, ok, "USES_ENV_DB missing")
		require.NotNil(t, usesEnvDB.SecretRef)

		db, ok := envByName["DB"]
		require.True(t, ok, "DB missing")
		require.NotNil(t, db.SecretRef)

		usesEnvDBRef := string(usesEnvDB.SecretRef.(pulumi.String))
		dbRef := string(db.SecretRef.(pulumi.String))
		assert.NotEqual(t, usesEnvDBRef, dbRef,
			"a Key Vault secret for config var ENV_DB and a composite secret for env var DB must not share a name")

		// Every Secret entry's Name must be unique — the real invariant Azure needs.
		seen := map[string]bool{}
		for _, entry := range result.Secrets {
			s, ok := entry.(app.SecretArgs)
			require.True(t, ok)
			name := string(s.Name.(pulumi.String))
			assert.False(t, seen[name], "duplicate Container App secret name %q", name)
			seen[name] = true
		}

		return nil
	}, pulumi.WithMocks("proj", "stack", azureNoopMocks{}))
	require.NoError(t, err)
}

// TestPostgresURLEndpointFlagsEmbeddedSecret verifies that postgresURLEndpoint
// reports hasSecret=true when the DSN's credentials interpolate a
// config-provided value (e.g. ${POSTGRES_PASSWORD}) — the exact shape of
// station's compose.yaml DATABASE_URL — so callers keep the resolved password
// out of the Container App's plaintext env. See DefangLabs/station#198.
func TestPostgresURLEndpointFlagsEmbeddedSecret(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		cp := NewConfigProvider("https://myvault.vault.azure.net")
		// Pre-seed the cache (as TestGetConfigValue_ReturnsCachedValue does) so
		// GetConfigValue doesn't attempt a real Key Vault fetch in this test.
		cp.cache["POSTGRES_PASSWORD"] = pulumi.ToSecret(pulumi.String("hunter2").ToStringOutput()).(pulumi.StringOutput)
		cp.fetched = true

		serviceEndpoints := map[string]pulumi.StringOutput{
			"db": pulumi.String("mystation.postgres.database.azure.com:5432").ToStringOutput(),
		}

		dsn := "postgresql://postgres:${POSTGRES_PASSWORD}@db:5432/station?sslmode=require"
		out, matched, hasSecret := postgresURLEndpoint(ctx, dsn, serviceEndpoints, cp)
		require.True(t, matched, "DSN host matches a managed service")
		assert.True(t, hasSecret, "DSN embeds ${POSTGRES_PASSWORD}; must be flagged as secret-bearing")

		out.ApplyT(func(got string) string {
			assert.Equal(t,
				"postgresql://postgres:hunter2@mystation.postgres.database.azure.com:5432/station?sslmode=require",
				got)
			return got
		})

		// A DSN with no ${VAR} at all matches but interpolates nothing, so it
		// isn't secret-bearing even though the branch runs.
		//nolint:gosec // test fixture, not a credential
		plainDSN := "postgresql://postgres:plaintext@db:5432/station?sslmode=require"
		_, matched2, hasSecret2 := postgresURLEndpoint(ctx, plainDSN, serviceEndpoints, cp)
		require.True(t, matched2)
		assert.False(t, hasSecret2, "no ${VAR} in the DSN; nothing was substituted")

		return nil
	}, pulumi.WithMocks("proj", "stack", azureNoopMocks{}))
	require.NoError(t, err)
}

// TestBuildEnvVarsKeepsPostgresDSNPasswordOutOfPlaintext reproduces
// DefangLabs/station#198 end-to-end through buildEnvVars: a DATABASE_URL
// shaped exactly like station's compose.yaml (a static string that embeds
// ${POSTGRES_PASSWORD}, so GetConfigName2 doesn't treat it as a bare secret
// ref) must still come out of buildEnvVars as a SecretRef, never as an
// EnvironmentVarArgs.Value carrying the password in clear.
func TestBuildEnvVarsKeepsPostgresDSNPasswordOutOfPlaintext(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		const vaultURL = "https://myvault.vault.azure.net"
		cp := NewConfigProvider(vaultURL)
		cp.cache["POSTGRES_PASSWORD"] = pulumi.ToSecret(pulumi.String("hunter2").ToStringOutput()).(pulumi.StringOutput)
		cp.fetched = true

		infra := &SharedInfra{ConfigProvider: cp, KeyVaultURL: vaultURL}
		serviceEndpoints := map[string]pulumi.StringOutput{
			"db": pulumi.String("mystation.postgres.database.azure.com:5432").ToStringOutput(),
		}
		svc := compose.ServiceConfig{
			Environment: compose.Environment{
				// Same shape as station's compose.yaml: a static DSN with a bare
				// ${POSTGRES_PASSWORD} reference, not itself a bare secret ref.
				"DATABASE_URL":      pulumi.String("postgresql://postgres:${POSTGRES_PASSWORD}@db:5432/station?sslmode=require"),
				"POSTGRES_PASSWORD": nil, // bare key ("POSTGRES_PASSWORD:" in compose): its own Key Vault secret ref
			},
		}

		result := buildEnvVars(ctx, "api", svc, infra, serviceEndpoints, nil, nil)
		envByName := envVarsByName(result)

		dbURL, ok := envByName["DATABASE_URL"]
		require.True(t, ok, "DATABASE_URL missing")
		assert.Nil(t, dbURL.Value,
			"DATABASE_URL must not carry the resolved password as a plaintext env Value")
		require.NotNil(t, dbURL.SecretRef, "DATABASE_URL must be a SecretRef")

		// Confirm no plaintext env value anywhere in the result contains the password.
		for name, args := range envByName {
			if args.Value == nil {
				continue
			}
			args.Value.ToStringPtrOutput().ApplyT(func(v *string) *string {
				if v != nil {
					assert.NotContains(t, *v, "hunter2",
						"env var %q leaks the resolved password in plaintext", name)
				}
				return v
			})
		}

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
		result := buildEnvVars(ctx, serviceName, compose.ServiceConfig{}, &SharedInfra{}, nil, nil, nil)

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

// TestBuildEnvVarsInjectsPolicyIdentityClientID verifies that AZURE_CLIENT_ID
// is set to the x-defang-policies identity's client ID when one is present —
// with more than one user-assigned identity on the Container App,
// DefaultAzureCredential can't otherwise tell which one a self-redeploying
// service should authenticate as (DefangLabs/pulumi-defang#326) — and is
// absent for the (common) case of a service with no policies.
func TestBuildEnvVarsInjectsPolicyIdentityClientID(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		result := buildEnvVars(ctx, "svc", compose.ServiceConfig{}, &SharedInfra{}, nil, nil, nil)
		_, ok := envVarsByName(result)["AZURE_CLIENT_ID"]
		assert.False(t, ok, "AZURE_CLIENT_ID should be absent without a policy identity")

		policyIdentity := &PolicyIdentity{
			ClientID: pulumi.String("11111111-1111-1111-1111-111111111111").ToStringOutput(),
		}
		result = buildEnvVars(ctx, "svc", compose.ServiceConfig{}, &SharedInfra{}, nil, nil, policyIdentity)
		clientID, ok := envVarsByName(result)["AZURE_CLIENT_ID"]
		require.True(t, ok, "AZURE_CLIENT_ID env var not found with a policy identity present")
		assert.Equal(t, policyIdentity.ClientID, clientID.Value,
			"AZURE_CLIENT_ID should carry the policy identity's client ID")

		// A compose-declared AZURE_CLIENT_ID must win: two env entries with the
		// same name make Container Apps reject the spec (or pick one
		// nondeterministically).
		svcWithOwnClientID := compose.ServiceConfig{
			Environment: compose.Environment{"AZURE_CLIENT_ID": pulumi.String("user-supplied")},
		}
		result = buildEnvVars(ctx, "svc", svcWithOwnClientID, &SharedInfra{}, nil, nil, policyIdentity)
		require.Equal(t, 1, countEnvVarsByName(result, "AZURE_CLIENT_ID"),
			"exactly one AZURE_CLIENT_ID entry, not one from the compose file plus ours")
		// The compose-loop path wraps even a literal value in a StringOutput
		// (via compose.InterpolateEnvironmentVariable) — assert inside ApplyT
		// so it runs after that (mock-synchronous) resolution completes.
		userValue, ok := envVarsByName(result)["AZURE_CLIENT_ID"].Value.(pulumi.StringOutput)
		require.True(t, ok)
		userValue.ApplyT(func(v string) string {
			assert.Equal(t, "user-supplied", v,
				"the compose file's own AZURE_CLIENT_ID must not be overridden by the policy identity's")
			return v
		})
		return nil
	}, pulumi.WithMocks("proj", "stack", azureNoopMocks{}))
	require.NoError(t, err)
}

// countEnvVarsByName counts raw env entries in result.Envs carrying name —
// envVarsByName collapses duplicates into a map, hiding exactly the
// double-entry bug this guards against, so this walks the slice directly.
func countEnvVarsByName(result envResult, name string) int {
	count := 0
	for _, e := range result.Envs {
		args, ok := e.(app.EnvironmentVarArgs)
		if !ok {
			continue
		}
		if n, ok := args.Name.(pulumi.String); ok && string(n) == name {
			count++
		}
	}
	return count
}

// TestBuildProbesClampsInitialDelay covers the Azure ceiling on a probe's
// InitialDelaySeconds. A compose file that is valid on ECS (start_period well over a
// minute) used to fail the whole deploy with HTTP 400
// ContainerAppProbeInitialDelaySecondsOutOfRange. It is now clamped, and the seconds
// removed from the delay are moved into the retry window so the container keeps the
// startup budget its author asked for.
func TestBuildProbesClampsInitialDelay(t *testing.T) {
	for _, tt := range []struct {
		name                           string
		startPeriod, interval, retries int32
		wantDelay, wantThreshold       int
		wantThresholdSet               bool
	}{
		{
			name:        "under the ceiling is left alone",
			startPeriod: 15, interval: 30, retries: 5,
			wantDelay: 15, wantThreshold: 5, wantThresholdSet: true,
		},
		{
			name:        "exactly at the ceiling is left alone",
			startPeriod: 60, interval: 30, retries: 5,
			wantDelay: 60, wantThreshold: 5, wantThresholdSet: true,
		},
		{
			// 240 + 5*30 = 390s requested; 60 + 11*30 would be needed, so the
			// threshold rises by ceil(180/30) = 6, from 5 to 11 -> capped at 10.
			name:        "over the ceiling moves the lost grace into retries",
			startPeriod: 240, interval: 30, retries: 5,
			wantDelay: 60, wantThreshold: 10, wantThresholdSet: true,
		},
		{
			// 120 + 2*60 = 240s requested; losing 60s needs ceil(60/60) = 1 more retry.
			name:        "raises the threshold just enough",
			startPeriod: 120, interval: 60, retries: 2,
			wantDelay: 60, wantThreshold: 3, wantThresholdSet: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				svc := compose.ServiceConfig{
					Ports: []compose.ServicePortConfig{{Target: 5050}},
					HealthCheck: &compose.HealthCheckConfig{
						Test:               []string{"CMD", "curl", "-f", "http://localhost:5050/"},
						StartPeriodSeconds: tt.startPeriod,
						IntervalSeconds:    tt.interval,
						Retries:            tt.retries,
					},
				}
				probes := buildProbes(ctx, "app", svc)
				require.Len(t, probes, 1)
				args, ok := probes[0].(app.ContainerAppProbeArgs)
				require.True(t, ok)
				assert.Equal(t, pulumi.Int(tt.wantDelay), args.InitialDelaySeconds)
				assert.LessOrEqual(t, tt.wantDelay, maxProbeInitialDelaySeconds)
				if tt.wantThresholdSet {
					assert.Equal(t, pulumi.Int(tt.wantThreshold), args.FailureThreshold)
				}
				return nil
			}, pulumi.WithMocks("project", "stack", azureNoopMocks{}))
			require.NoError(t, err)
		})
	}
}
