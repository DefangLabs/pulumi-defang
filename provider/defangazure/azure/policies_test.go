package azure

import (
	"testing"
	"time"

	"github.com/DefangLabs/pulumi-defang/provider/compose"

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

// TestRoleNameFilterEscapesQuotes verifies OData single-quote escaping in the
// role-definition list filter. Without it, a role name containing "'" either
// breaks the filter or (crafted as `foo' or roleName eq 'Owner`) widens the
// match to a role the compose file never asked for.
func TestRoleNameFilterEscapesQuotes(t *testing.T) {
	tests := []struct {
		name   string
		policy string
		want   string
	}{
		{"plain name", "deployer", "roleName eq 'deployer'"},
		{"single quote", "O'Brien", "roleName eq 'O''Brien'"},
		{"injection attempt", "foo' or roleName eq 'Owner", "roleName eq 'foo'' or roleName eq ''Owner'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, roleNameFilter(tt.policy))
		})
	}
}

func TestParsePolicies(t *testing.T) {
	// Role names, role-definition IDs, and either with a scope; normalized,
	// with empty entries (a "${VAR:-}" the stack leaves unset) dropped.
	const roleDefID = "/subscriptions/sub/providers/Microsoft.Authorization/roleDefinitions/guid"
	got, err := ParsePolicies([]string{
		"Contributor@subscription, Storage Blob Data Contributor",
		"",
		roleDefID,
		roleDefID + "@subscription",
		"Reader@/subscriptions/sub/resourceGroups/other",
	})
	require.NoError(t, err)
	assert.Equal(t, []PolicyGrant{
		{Role: "Contributor", Scope: "subscription"},
		{Role: "Storage Blob Data Contributor"},
		{Role: roleDefID},
		{Role: roleDefID, Scope: "subscription"},
		{Role: "Reader", Scope: "/subscriptions/sub/resourceGroups/other"},
	}, got)

	// A role name is free-form, so the shapes ruled out are the ones that
	// look like a path without being a role-definition ID — which is what
	// another cloud's identifier looks like here — plus a half-written scope.
	for _, entry := range []string{
		"arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
		"roles/run.developer",
		"projects/my-proj/roles/deployer",
		"Contributor@",
		"@subscription",
	} {
		_, err := ParsePolicies([]string{entry})
		require.ErrorIs(t, err, ErrPolicyNotAzure, "entry %q", entry)
		require.ErrorContains(t, err, "@SCOPE", "entry %q", entry)
	}

	// The literal check is shared, and reported as the compose-level error.
	_, err = ParsePolicies([]string{"${POLICIES}"})
	require.ErrorIs(t, err, compose.ErrPolicyUnresolvedVariable)
}

// TestPolicyGrantString checks that a grant reconstructs the compose entry it
// was parsed from: the errors raised while granting name it with %q, and the
// value the author wrote is more useful there than the struct.
func TestPolicyGrantString(t *testing.T) {
	assert.Equal(t, "Contributor", PolicyGrant{Role: "Contributor"}.String())
	assert.Equal(t, "Contributor@subscription",
		PolicyGrant{Role: "Contributor", Scope: "subscription"}.String())
	assert.Equal(t, "Reader@/subscriptions/sub/resourceGroups/rg",
		PolicyGrant{Role: "Reader", Scope: "/subscriptions/sub/resourceGroups/rg"}.String())

	// Round-trips through the parser, which is what makes the reconstruction
	// worth trusting in an error message.
	grants, err := ParsePolicies([]string{"Contributor@subscription"})
	require.NoError(t, err)
	assert.Equal(t, "Contributor@subscription", grants[0].String())
}

// TestSubscriptionScope checks the reduction of a resource group's own ARM ID
// to the subscription scope above it — the source of the `subscription`
// keyword's scope, chosen so no stack config has to be set for it to be right.
func TestSubscriptionScope(t *testing.T) {
	got, err := subscriptionScope("/subscriptions/0000-1111/resourceGroups/rg")
	require.NoError(t, err)
	assert.Equal(t, "/subscriptions/0000-1111", got)

	// ARM is inconsistent about the casing of this segment ("resourcegroups"
	// in the CLI's own URLs), so the match is case-insensitive.
	got, err = subscriptionScope("/Subscriptions/0000-1111/resourcegroups/rg")
	require.NoError(t, err)
	assert.Equal(t, "/subscriptions/0000-1111", got)

	for _, bad := range []string{"", "/", "/subscriptions", "/subscriptions/", "/resourceGroups/rg"} {
		_, err := subscriptionScope(bad)
		require.ErrorContains(t, err, "no subscription", "input %q", bad)
	}
}

// TestResolvePolicyScope covers the three spellings a scope half can take,
// plus the rejection of a keyword that is neither.
func TestResolvePolicyScope(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		rgID := pulumi.String("/subscriptions/0000-1111/resourceGroups/rg").ToStringOutput()

		// No scope: the project's resource group, unchanged from before
		// scopes existed.
		scope, err := resolvePolicyScope(rgID, "")
		require.NoError(t, err)
		assert.Equal(t, "/subscriptions/0000-1111/resourceGroups/rg", awaitString(t, scope))

		scope, err = resolvePolicyScope(rgID, "subscription")
		require.NoError(t, err)
		assert.Equal(t, "/subscriptions/0000-1111", awaitString(t, scope))

		scope, err = resolvePolicyScope(rgID, "/subscriptions/0000-1111/resourceGroups/other")
		require.NoError(t, err)
		assert.Equal(t, "/subscriptions/0000-1111/resourceGroups/other", awaitString(t, scope))

		_, err = resolvePolicyScope(rgID, "subscriptions")
		require.ErrorContains(t, err, "unknown policy scope")
		return nil
	}, pulumi.WithMocks("proj", "stack", azureNoopMocks{}))
	require.NoError(t, err)
}

// awaitString resolves a StringOutput inside a pulumi.RunErr program, where
// an ApplyT runs on the engine's own goroutine.
func awaitString(t *testing.T, out pulumi.StringOutput) string {
	t.Helper()
	done := make(chan string, 1)
	out.ApplyT(func(s string) string {
		done <- s
		return s
	})
	select {
	case s := <-done:
		return s
	case <-time.After(10 * time.Second):
		t.Fatal("output never resolved")
		return ""
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
