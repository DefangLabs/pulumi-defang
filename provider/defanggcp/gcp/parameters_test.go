package gcp

import (
	"errors"
	"fmt"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const getSecretVersionToken = "gcp:secretmanager/getSecretVersion:getSecretVersion"

var errSecretNotFound = errors.New("secret not found")

func TestGetSecretID(t *testing.T) {
	assert.Equal(t, "Defang_myproject_prod_DB_PASSWORD", getSecretID("myproject", "prod", "DB_PASSWORD"))
}

func TestGetSecretRef(t *testing.T) {
	mocks := noopMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		cp := NewConfigProvider("myproject")
		ref, err := cp.GetSecretRef(ctx, "DB_PASSWORD")
		require.NoError(t, err)
		assert.Equal(t, "Defang_myproject_mystack_DB_PASSWORD", ref)
		return nil
	}, pulumi.WithMocks("myproject", "mystack", mocks))
	require.NoError(t, err)
}

func TestGetConfigValue(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]string // keyed by full secret ID
		key      string
		wantErr  bool // true when the key is missing and the output errors
		expected string
	}{
		{
			name:     "returns value for existing key",
			params:   map[string]string{"Defang_myproject_mystack_MY_KEY": "secret-value"},
			key:      "MY_KEY",
			expected: "secret-value",
		},
		{
			name:    "returns error output for missing key",
			params:  map[string]string{},
			key:     "MISSING",
			wantErr: true,
		},
		{
			name: "returns correct value among multiple params",
			params: map[string]string{
				"Defang_myproject_mystack_A": "alpha",
				"Defang_myproject_mystack_B": "beta",
			},
			key:      "B",
			expected: "beta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				cp := NewConfigProvider("myproject")
				out := cp.GetConfigValue(ctx, tt.key)

				if !tt.wantErr {
					out.ApplyT(func(got string) string {
						assert.Equal(t, tt.expected, got)
						return got
					})
				}
				return nil
			}, pulumi.WithMocks("myproject", "mystack", secretManagerMocks{params: tt.params}))
			require.NoError(t, err)
		})
	}
}

func TestGetConfigValue_CachesPerKey(t *testing.T) {
	// LookupSecretVersion is per-key, so the cache must dedupe repeated lookups
	// of the same key. Distinct keys still incur one call each.
	callCount := 0
	mocks := &countingSecretMocks{
		params: map[string]string{
			"Defang_proj_stack_K1": "v1",
			"Defang_proj_stack_K2": "v2",
		},
		calls: &callCount,
	}

	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		cp := NewConfigProvider("proj")
		out1a := cp.GetConfigValue(ctx, "K1")
		cp.GetConfigValue(ctx, "K1")
		out1b := cp.GetConfigValue(ctx, "K1")
		cp.GetConfigValue(ctx, "K2")

		// Cached output must be the identical instance, not a re-wrapped copy.
		assert.Equal(t, out1a, out1b)

		out1b.ApplyT(func(got string) string {
			assert.Equal(t, 2, callCount, "LookupSecretVersion should only be called once per distinct key")
			return got
		})
		return nil
	}, pulumi.WithMocks("proj", "stack", mocks))
	require.NoError(t, err)
}

type noopMocks struct{}

func (noopMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (noopMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

// secretManagerMocks answers LookupSecretVersion invokes: returns secretData
// for known secret IDs, errors for unknown ones (triggering ConfigNotFoundOutput).
type secretManagerMocks struct {
	params map[string]string
}

func (m secretManagerMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (m secretManagerMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	if args.Token != getSecretVersionToken {
		return args.Args, nil
	}
	secret := args.Args["secret"].StringValue()
	val, ok := m.params[secret]
	if !ok {
		return nil, fmt.Errorf("%w: %q", errSecretNotFound, secret)
	}
	return resource.PropertyMap{
		"secret":     resource.NewStringProperty(secret),
		"secretData": resource.NewStringProperty(val),
	}, nil
}

type countingSecretMocks struct {
	params map[string]string
	calls  *int
}

func (m *countingSecretMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (m *countingSecretMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	if args.Token != getSecretVersionToken {
		return args.Args, nil
	}
	*m.calls++
	secret := args.Args["secret"].StringValue()
	val, ok := m.params[secret]
	if !ok {
		return nil, fmt.Errorf("%w: %q", errSecretNotFound, secret)
	}
	return resource.PropertyMap{
		"secret":     resource.NewStringProperty(secret),
		"secretData": resource.NewStringProperty(val),
	}, nil
}
