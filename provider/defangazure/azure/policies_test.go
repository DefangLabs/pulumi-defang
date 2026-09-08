package azure

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsAzureRoleDefinitionID(t *testing.T) {
	tests := []struct {
		name   string
		policy string
		want   bool
	}{
		{"built-in role name", "Contributor", false},
		{"custom role name", "deployer", false},
		{"full role definition id", "/subscriptions/sub/providers/Microsoft.Authorization/roleDefinitions/guid", true},
		{
			"resource-group-scoped id",
			"/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Authorization/roleDefinitions/guid",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isAzureRoleDefinitionID(tt.policy))
		})
	}
}

// TestCreatePolicyIdentity_NoPolicies verifies the fast path: no policies
// means no identity is created and no error — the common case, since most
// services don't use x-defang-policies at all.
func TestCreatePolicyIdentity_NoPolicies(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		infra := &SharedInfra{}
		identity, err := CreatePolicyIdentity(ctx, "svc", nil, infra)
		require.NoError(t, err)
		assert.Nil(t, identity)
		return nil
	}, pulumi.WithMocks("proj", "stack", azureNoopMocks{}))
	require.NoError(t, err)
}
