package gcp

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSecretID(t *testing.T) {
	assert.Equal(t, "Defang/myproject/prod/DB_PASSWORD", getSecretID("myproject", "prod", "DB_PASSWORD"))
}

func TestGetSecretRef(t *testing.T) {
	mocks := noopMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		cp := NewConfigProvider("myproject")
		ref, err := cp.GetSecretRef(ctx, "DB_PASSWORD")
		require.NoError(t, err)
		assert.Equal(t, "Defang/myproject/mystack/DB_PASSWORD", ref)
		return nil
	}, pulumi.WithMocks("myproject", "mystack", mocks))
	require.NoError(t, err)
}

type noopMocks struct{}

func (noopMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (noopMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}
