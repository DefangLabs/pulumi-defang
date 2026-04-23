package azure

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/internals"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigProvider(t *testing.T) {
	cp := NewConfigProvider("proj", "")
	require.NotNil(t, cp)
	assert.NotNil(t, cp.cache)
	assert.Empty(t, cp.cache)
	assert.False(t, cp.fetched)
	assert.Equal(t, "Defang", cp.prefix)
}

// TestGetConfigValue_ReturnsCachedValue verifies the cache hit path: when a
// value is already in the cache (simulating a prior successful fetch), it's
// returned as a secret-marked StringOutput.
func TestGetConfigValue_ReturnsCachedValue(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		cp := NewConfigProvider("myproject", "")
		// Pre-seed the cache as if a prior fetch succeeded. `fetched=true`
		// keeps GetConfigValue from re-running the (nonexistent) fetch in
		// this in-package unit test.
		cp.cache["MY_KEY"] = pulumi.ToSecret(pulumi.String("secret-value").ToStringOutput()).(pulumi.StringOutput)
		cp.fetched = true

		out := cp.GetConfigValue(ctx, "MY_KEY")

		// Await instead of ApplyT so we can inspect the secret bit,
		// which ApplyT does not expose.
		res, err := internals.UnsafeAwaitOutput(ctx.Context(), out)
		require.NoError(t, err)
		assert.Equal(t, "secret-value", res.Value)
		assert.True(t, res.Secret, "GetConfigValue output must be marked secret")
		return nil
	}, pulumi.WithMocks("myproject", "mystack", azureNoopMocks{}))
	require.NoError(t, err)
}

// TestGetConfigValue_UnknownKeyReturnsEmptyString verifies the Azure contract:
// unknown keys resolve to "" (not an error), matching the previous stack-config
// backed implementation.
func TestGetConfigValue_UnknownKeyReturnsEmptyString(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		// No vault URL → fetch is skipped; cache stays empty.
		cp := NewConfigProvider("myproject", "")

		out := cp.GetConfigValue(ctx, "MISSING")

		res, err := internals.UnsafeAwaitOutput(ctx.Context(), out)
		require.NoError(t, err)
		assert.Empty(t, res.Value)
		assert.True(t, res.Secret)
		return nil
	}, pulumi.WithMocks("myproject", "mystack", azureNoopMocks{}))
	require.NoError(t, err)
}

func TestGetConfigValue_CachesOutput(t *testing.T) {
	// Verify repeated calls for the same key return the identical cached
	// pulumi.StringOutput (same OutputState pointer) instead of re-wrapping.
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		cp := NewConfigProvider("proj", "")
		out1 := cp.GetConfigValue(ctx, "K")
		out2 := cp.GetConfigValue(ctx, "K")
		assert.Equal(t, out1, out2)
		return nil
	}, pulumi.WithMocks("proj", "stack", azureNoopMocks{}))
	require.NoError(t, err)
}

func TestGetSecretRef(t *testing.T) {
	const vaultURL = "https://myvault.vault.azure.net"
	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "replaces underscores with hyphens",
			key:  "DB_PASSWORD",
			want: vaultURL + "/secrets/Defang--myproject--mystack--DB-PASSWORD",
		},
		{
			name: "key without underscores passes through",
			key:  "APIKEY",
			want: vaultURL + "/secrets/Defang--myproject--mystack--APIKEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				cp := NewConfigProvider("myproject", vaultURL)
				ref, err := cp.GetSecretRef(ctx, tt.key)
				require.NoError(t, err)
				assert.Equal(t, tt.want, ref)
				return nil
			}, pulumi.WithMocks("myproject", "mystack", azureNoopMocks{}))
			require.NoError(t, err)
		})
	}
}

// TestGetSecretRef_NoVault verifies that calling GetSecretRef on a ConfigProvider
// constructed without a keyVaultURL returns an explicit error — the caller
// (Container App env builder) must gate on KeyVaultURL before calling this.
func TestGetSecretRef_NoVault(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		cp := NewConfigProvider("myproject", "")
		_, err := cp.GetSecretRef(ctx, "SOMETHING")
		assert.ErrorIs(t, err, ErrNoKeyVaultConfigured)
		return nil
	}, pulumi.WithMocks("myproject", "mystack", azureNoopMocks{}))
	require.NoError(t, err)
}

type azureNoopMocks struct{}

func (azureNoopMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (azureNoopMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}
