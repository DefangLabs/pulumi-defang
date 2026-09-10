package gcp

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePolicies(t *testing.T) {
	// The three qualified forms and a bare custom-role ID pass, normalized
	// and with empty entries (a "${VAR:-}" the stack leaves unset) dropped.
	got, err := ParsePolicies([]string{
		"roles/run.developer, projects/my-proj/roles/deployer",
		"",
		"organizations/123/roles/auditor",
		"deployer",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"roles/run.developer",
		"projects/my-proj/roles/deployer",
		"organizations/123/roles/auditor",
		"deployer",
	}, got)

	// A role ID holds letters, digits, "_" and "." only, and a qualified
	// name is one of exactly three shapes — so anything else is rejected
	// here rather than resolved into "projects/<proj>/roles/<not a role>".
	for _, entry := range []string{
		"arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
		"/subscriptions/sub/providers/Microsoft.Authorization/roleDefinitions/x",
		"Contributor@/subscriptions/sub/resourceGroups/rg",
		// An Azure scope suffix holds no "/" or ":", so the shape of the ID
		// itself is what rules it out rather than a stray character.
		"Contributor@subscription",
		// Prefix-only values that are not role paths: a bare prefix match
		// would have let these through to be bound as roles.
		"projects/my-proj",
		"roles/",
		"organizations/123/roles/",
		"projects/my-proj/deployer",
	} {
		_, err := ParsePolicies([]string{entry})
		require.ErrorIs(t, err, ErrPolicyNotGCP, "entry %q", entry)
		require.ErrorContains(t, err, "${VAR}", "entry %q", entry)
	}

	// The literal check is shared, and reported as the compose-level error.
	_, err = ParsePolicies([]string{"${POLICIES}"})
	require.ErrorIs(t, err, compose.ErrPolicyUnresolvedVariable)
}
