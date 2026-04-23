package compose

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConfigProvider returns pre-seeded values; missing keys fail.
type mockConfigProvider struct {
	values map[string]string
}

func (m *mockConfigProvider) GetConfigValue(
	_ *pulumi.Context, key string, opts ...pulumi.InvokeOption,
) pulumi.StringOutput {
	if v, ok := m.values[key]; ok {
		return pulumi.String(v).ToStringOutput()
	}
	return ConfigNotFoundOutput(key)
}

func (m *mockConfigProvider) GetSecretRef(
	_ *pulumi.Context, key string, _ ...pulumi.InvokeOption,
) (string, error) {
	return "mock-secret-ref-" + key, nil
}

// testMocks is a no-op Pulumi mock runtime required by pulumi.WithMocks.
type testMocks struct{}

func (testMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (testMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func TestToPulumiStringArray(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		assert.Nil(t, ToPulumiStringArray(nil))
	})
	t.Run("empty slice returns nil", func(t *testing.T) {
		assert.Nil(t, ToPulumiStringArray([]string{}))
	})
	t.Run("non-empty slice returns array of same length", func(t *testing.T) {
		result := ToPulumiStringArray([]string{"a", "b", "c"})
		assert.Len(t, result, 3)
	})
}

func TestGetConfigOrEnvValue(t *testing.T) {
	tests := []struct {
		name         string
		environment  map[string]*string
		key          string
		defaultValue string
		configs      map[string]string
		expected     string
	}{
		{
			name:         "nil environment uses default",
			environment:  nil,
			key:          "MY_KEY",
			defaultValue: "fallback",
			expected:     "fallback",
		},
		{
			name:         "key absent returns default",
			environment:  map[string]*string{},
			key:          "MY_KEY",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:        "empty string value is literal empty",
			environment: map[string]*string{"MY_KEY": ptr("")},
			key:         "MY_KEY",
			expected:    "",
		},
		{
			name:        "plain string value returned as-is",
			environment: map[string]*string{"MY_KEY": ptr("hello")},
			key:         "MY_KEY",
			expected:    "hello",
		},
		{
			name:        "interpolated value resolves variables from config provider",
			environment: map[string]*string{"MY_KEY": ptr("prefix_${SECRET}_suffix")},
			key:         "MY_KEY",
			configs:     map[string]string{"SECRET": "resolved"},
			expected:    "prefix_resolved_suffix",
		},
		{
			// Compose spec: "KEY:" (no value) → resolve from config at runtime.
			name:        "nil value resolves from config provider via ${KEY}",
			environment: map[string]*string{"MY_KEY": nil},
			key:         "MY_KEY",
			configs:     map[string]string{"MY_KEY": "from-config"},
			expected:    "from-config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				svc := ServiceConfig{Environment: tt.environment}
				provider := &mockConfigProvider{values: tt.configs}
				out := GetConfigOrEnvValue(ctx, provider, svc, tt.key, tt.defaultValue)

				out.ApplyT(func(got string) string {
					assert.Equal(t, tt.expected, got)
					return got
				})
				return nil
			}, pulumi.WithMocks("proj", "stack", testMocks{}))
			require.NoError(t, err)
		})
	}
}

func TestGetConfigName(t *testing.T) {
	tests := []struct {
		value   string
		wantVar string
	}{
		{"${MY_KEY}", "MY_KEY"},  // bare braced reference
		{"$OTHER", "OTHER"},      // bare unbraced reference
		{"prefix_${MY_KEY}", ""}, // text before var
		{"${MY_KEY}_suffix", ""}, // text after var
		{"${A}_${B}", ""},        // multiple vars
		{"${VAR:-default}", ""},  // has default modifier
		{"${VAR:+alt}", ""},      // has presence modifier
		{"${VAR:?err}", ""},      // has required modifier; TODO: this could be made to work #159
		{"${VAR+}", ""},          // has empty modifier
		{"literal", ""},          // no interpolation
		{"", ""},                 // empty
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			v := GetConfigName(tt.value)
			assert.Equal(t, tt.wantVar, v)
		})
	}
}

func TestInterpolateEnvironmentVariable(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		configs  map[string]string
		expected string
	}{
		{
			name:     "plain literal",
			value:    "hello",
			expected: "hello",
		},
		{
			name:     "single variable",
			value:    "${MY_VAR}",
			configs:  map[string]string{"MY_VAR": "secret"},
			expected: "secret",
		},
		{
			name:     "variable with prefix and suffix",
			value:    "prefix_${MY_VAR}_suffix",
			configs:  map[string]string{"MY_VAR": "value"},
			expected: "prefix_value_suffix",
		},
		{
			name:     "multiple variables",
			value:    "${VAR1}_${VAR2}",
			configs:  map[string]string{"VAR1": "hello", "VAR2": "world"},
			expected: "hello_world",
		},
		{
			name:     "unbraced variable",
			value:    "$MY_VAR",
			configs:  map[string]string{"MY_VAR": "secret"},
			expected: "secret",
		},
		{
			name:     "escaped dollar produces literal dollar",
			value:    "$${NOT_A_VAR}",
			expected: "${NOT_A_VAR}",
		},
		{
			name:     "missing variable resolves from config",
			value:    "${SECRET}",
			configs:  map[string]string{"SECRET": "found"},
			expected: "found",
		},
		{
			name:     "empty string",
			value:    "",
			expected: "",
		},
		{
			name:     "variable adjacent to text on both sides without separators",
			value:    "arn:aws:iam::${ACCOUNT_ID}:role/my-role",
			configs:  map[string]string{"ACCOUNT_ID": "123456789"},
			expected: "arn:aws:iam::123456789:role/my-role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				provider := &mockConfigProvider{values: tt.configs}
				out := InterpolateEnvironmentVariable(ctx, provider, tt.value)

				out.ApplyT(func(got string) string {
					assert.Equal(t, tt.expected, got)
					return got
				})
				return nil
			}, pulumi.WithMocks("proj", "stack", testMocks{}))
			require.NoError(t, err)
		})
	}

	t.Run("nil config provider returns raw string", func(t *testing.T) {
		err := pulumi.RunErr(func(ctx *pulumi.Context) error {
			out := InterpolateEnvironmentVariable(ctx, nil, "value with ${VAR}")

			out.ApplyT(func(got string) string {
				assert.Equal(t, "value with ${VAR}", got)
				return got
			})
			return nil
		}, pulumi.WithMocks("proj", "stack", testMocks{}))
		require.NoError(t, err)
	})
}
