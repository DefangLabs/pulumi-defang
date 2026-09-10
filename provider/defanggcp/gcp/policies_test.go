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

	// A bare role ID holds letters, digits, "_" and "." only, so an
	// identifier belonging to another cloud is rejected here rather than
	// resolved into "projects/<proj>/roles/<other cloud's ARN>".
	for _, entry := range []string{
		"arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
		"/subscriptions/sub/providers/Microsoft.Authorization/roleDefinitions/x",
		"Contributor@/subscriptions/sub/resourceGroups/rg",
	} {
		_, err := ParsePolicies([]string{entry})
		require.ErrorIs(t, err, ErrPolicyNotGCP, "entry %q", entry)
		require.ErrorContains(t, err, "${VAR}", "entry %q", entry)
	}

	// The literal check is shared, and reported as the compose-level error.
	_, err = ParsePolicies([]string{"${POLICIES}"})
	require.ErrorIs(t, err, compose.ErrPolicyUnresolvedVariable)
}
