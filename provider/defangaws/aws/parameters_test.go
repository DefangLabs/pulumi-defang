package aws

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/internals"
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

func TestGetConfigValue(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]string
		key      string
		wantErr  bool // true when the key is missing and GetConfigValue should fail
		expected string
	}{
		{
			name:     "returns value for existing key",
			params:   map[string]string{"MY_KEY": "secret-value"},
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
				out := cp.GetConfigValue(ctx, tt.key)

				if !tt.wantErr {
					// Await instead of ApplyT so we can inspect the secret bit,
					// which ApplyT does not expose.
					res, err := internals.UnsafeAwaitOutput(ctx.Context(), out)
					require.NoError(t, err)
					assert.Equal(t, tt.expected, res.Value)
					assert.True(t, res.Secret, "GetConfigValue output must be marked secret")
				}
				return nil
			}, pulumi.WithMocks("myproject", "mystack", ssmMocks{params: tt.params}))
			require.NoError(t, err)
		})
	}
}

func TestGetConfigValue_CachesAfterFirstFetch(t *testing.T) {
	callCount := 0
	mocks := &countingMocks{params: map[string]string{"K": "v"}, calls: &callCount}

	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		cp := NewConfigProvider("proj")
		cp.GetConfigValue(ctx, "K")
		cp.GetConfigValue(ctx, "K")
		out := cp.GetConfigValue(ctx, "K")

		// Assert inside ApplyT so it runs after the async fetch completes.
		out.ApplyT(func(got string) string {
			assert.Equal(t, 1, callCount, "SSM should only be called once")
			return got
		})
		return nil
	}, pulumi.WithMocks("proj", "stack", mocks))
	require.NoError(t, err)
}

func TestGetSecretRef(t *testing.T) {
	mocks := identityMocks{
		region:    "us-west-2",
		accountId: "123456789012",
	}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		cp := NewConfigProvider("myproject")
		ref, err := cp.GetSecretRef(ctx, "DB_PASSWORD")
		require.NoError(t, err)
		assert.Equal(t, "arn:aws:ssm:us-west-2:123456789012:parameter/Defang/myproject/mystack/DB_PASSWORD", ref)
		return nil
	}, pulumi.WithMocks("myproject", "mystack", mocks))
	require.NoError(t, err)
}

// identityMocks returns region and account ID for GetRegion/GetCallerIdentity invokes.
type identityMocks struct {
	region    string
	accountId string
}

func (m identityMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (m identityMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	switch args.Token {
	case "aws:index/getRegion:getRegion":
		return resource.PropertyMap{
			"region": resource.NewStringProperty(m.region),
		}, nil
	case "aws:index/getCallerIdentity:getCallerIdentity":
		return resource.PropertyMap{
			"accountId": resource.NewStringProperty(m.accountId),
		}, nil
	}
	return args.Args, nil
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
