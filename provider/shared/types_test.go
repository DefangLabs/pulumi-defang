package shared

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConfigProvider returns pre-seeded values; missing keys resolve to "".
type mockConfigProvider struct {
	values map[string]string
}

func (m *mockConfigProvider) GetConfig(_ *pulumi.Context, key string) pulumi.StringOutput {
	if v, ok := m.values[key]; ok {
		return pulumi.String(v).ToStringOutput()
	}
	return pulumi.String("").ToStringOutput()
}

// testMocks is a no-op Pulumi mock runtime required by pulumi.WithMocks.
type testMocks struct{}

func (testMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (testMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
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
			name:     "escaped variable treated as literal",
			value:    "$${NOT_A_VAR}",
			expected: "$${NOT_A_VAR}",
		},
		{
			name:     "missing variable resolves to empty string",
			value:    "${MISSING}",
			configs:  map[string]string{},
			expected: "",
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
}
