package aws

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ssmMocks struct {
	// params maps base name -> value, e.g. {"MY_KEY": "my-value"}
	params map[string]string
}

func (m ssmMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (m ssmMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	if args.Token != "aws:ssm/getParametersByPath:getParametersByPath" {
		return args.Args, nil
	}

	path := args.Args["path"].StringValue()

	names := make([]resource.PropertyValue, 0, len(m.params))
	values := make([]resource.PropertyValue, 0, len(m.params))
	for k, v := range m.params {
		names = append(names, resource.NewStringProperty(path+k))
		values = append(values, resource.NewStringProperty(v))
	}

	return resource.PropertyMap{
		"names":  resource.NewArrayProperty(names),
		"values": resource.NewArrayProperty(values),
	}, nil
}

func TestGetConfig(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]string
		key         string
		wantUnknown bool // true when a zero StringOutput is expected (key absent)
		expected    string
	}{
		{
			name:     "returns value for existing key",
			params:   map[string]string{"MY_KEY": "secret-value"},
			key:      "MY_KEY",
			expected: "secret-value",
		},
		{
			name:        "returns zero output for missing key",
			params:      map[string]string{},
			key:         "MISSING",
			wantUnknown: true,
		},
		{
			name:     "returns correct value among multiple params",
			params:   map[string]string{"A": "alpha", "B": "beta"},
			key:      "B",
			expected: "beta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				cp := NewConfigProvider("myproject")
				out := cp.GetConfig(ctx, tt.key)

				if !tt.wantUnknown {
					out.ApplyT(func(got string) string {
						assert.Equal(t, tt.expected, got)
						return got
					})
				}
				return nil
			}, pulumi.WithMocks("myproject", "mystack", ssmMocks{params: tt.params}))
			require.NoError(t, err)
		})
	}
}

func TestGetConfig_CachesAfterFirstFetch(t *testing.T) {
	callCount := 0
	mocks := &countingMocks{params: map[string]string{"K": "v"}, calls: &callCount}

	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		cp := NewConfigProvider("proj")
		cp.GetConfig(ctx, "K")
		cp.GetConfig(ctx, "K")
		out := cp.GetConfig(ctx, "K")

		// Assert inside ApplyT so it runs after the async fetch completes.
		out.ApplyT(func(got string) string {
			assert.Equal(t, 1, callCount, "SSM should only be called once")
			return got
		})
		return nil
	}, pulumi.WithMocks("proj", "stack", mocks))
	require.NoError(t, err)
}

type countingMocks struct {
	params map[string]string
	calls  *int
}

func (m *countingMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (m *countingMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	if args.Token != "aws:ssm/getParametersByPath:getParametersByPath" {
		return args.Args, nil
	}

	*m.calls++
	path := args.Args["path"].StringValue()

	names := make([]resource.PropertyValue, 0, len(m.params))
	values := make([]resource.PropertyValue, 0, len(m.params))
	for k, v := range m.params {
		names = append(names, resource.NewStringProperty(path+k))
		values = append(values, resource.NewStringProperty(v))
	}

	return resource.PropertyMap{
		"names":  resource.NewArrayProperty(names),
		"values": resource.NewArrayProperty(values),
	}, nil
}
