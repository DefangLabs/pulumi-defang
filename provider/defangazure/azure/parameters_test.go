package azure

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/internals"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigProvider_NilValues(t *testing.T) {
	// Nil map must be replaced with an empty map, otherwise GetConfigValue
	// would panic on the p.values[key] lookup. Regression guard for the
	// previous stack-config backed implementation's contract.
	cp := NewConfigProvider("proj", nil)
	require.NotNil(t, cp)
	assert.NotNil(t, cp.values)
	assert.Empty(t, cp.values)
}

func TestGetConfigValue(t *testing.T) {
	tests := []struct {
		name     string
		values   map[string]string
		key      string
		expected string
	}{
		{
			name:     "returns value for existing key",
			values:   map[string]string{"MY_KEY": "secret-value"},
			key:      "MY_KEY",
			expected: "secret-value",
		},
		{
			// Azure contract: unknown keys resolve to "" (not an error),
			// matching the previous stack-config backed implementation.
			name:     "returns empty string for missing key",
			values:   map[string]string{},
			key:      "MISSING",
			expected: "",
		},
		{
			name:     "returns correct value among multiple values",
			values:   map[string]string{"A": "alpha", "B": "beta"},
			key:      "B",
			expected: "beta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				cp := NewConfigProvider("myproject", tt.values)
				out := cp.GetConfigValue(ctx, tt.key)

				// Await instead of ApplyT so we can inspect the secret bit,
				// which ApplyT does not expose.
				res, err := internals.UnsafeAwaitOutput(ctx.Context(), out)
				require.NoError(t, err)
				assert.Equal(t, tt.expected, res.Value)
				assert.True(t, res.Secret, "GetConfigValue output must be marked secret")
				return nil
			}, pulumi.WithMocks("myproject", "mystack", azureNoopMocks{}))
			require.NoError(t, err)
		})
	}
}

func TestGetConfigValue_CachesOutput(t *testing.T) {
	// Verify repeated calls for the same key return the identical cached
	// pulumi.StringOutput (same OutputState pointer) instead of re-wrapping.
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		cp := NewConfigProvider("proj", map[string]string{"K": "v"})
		out1 := cp.GetConfigValue(ctx, "K")
		out2 := cp.GetConfigValue(ctx, "K")
		assert.Equal(t, out1, out2)
		return nil
	}, pulumi.WithMocks("proj", "stack", azureNoopMocks{}))
	require.NoError(t, err)
}

func TestGetSecretRef(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "replaces underscores with hyphens",
			key:  "DB_PASSWORD",
			want: "Defang--myproject--mystack--DB-PASSWORD",
		},
		{
			name: "key without underscores passes through",
			key:  "APIKEY",
			want: "Defang--myproject--mystack--APIKEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				cp := NewConfigProvider("myproject", nil)
				ref, err := cp.GetSecretRef(ctx, tt.key)
				require.NoError(t, err)
				assert.Equal(t, tt.want, ref)
				return nil
			}, pulumi.WithMocks("myproject", "mystack", azureNoopMocks{}))
			require.NoError(t, err)
		})
	}
}

type azureNoopMocks struct{}

func (azureNoopMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (azureNoopMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}
