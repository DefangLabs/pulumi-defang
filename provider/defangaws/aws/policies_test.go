package aws

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePolicies(t *testing.T) {
	// ARNs and bare policy names pass, normalized and with empty entries (a
	// "${VAR:-}" the stack leaves unset) dropped.
	got, err := ParsePolicies([]string{
		"arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess, arn:aws:iam::123456789012:policy/deployer",
		"",
		"deployer",
		// IAM's PolicyName pattern is [\w+=,.@-]+, so "@" is legal in a name.
		"team@corp",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
		"arn:aws:iam::123456789012:policy/deployer",
		"deployer",
		"team@corp",
	}, got)

	// A name cannot hold "/" or ":", so an identifier belonging to another
	// cloud is rejected here rather than sent to IAM as a name.
	for _, entry := range []string{
		"roles/run.developer",
		"projects/my-proj/roles/deployer",
		"/subscriptions/sub/providers/Microsoft.Authorization/roleDefinitions/x",
		"Contributor@/subscriptions/sub/resourceGroups/rg",
	} {
		_, err := ParsePolicies([]string{entry})
		require.ErrorIs(t, err, ErrPolicyNotAWS, "entry %q", entry)
		require.ErrorContains(t, err, "${VAR}", "entry %q", entry)
	}

	// The literal check is shared, and reported as the compose-level error.
	_, err = ParsePolicies([]string{"${POLICIES}"})
	require.ErrorIs(t, err, compose.ErrPolicyUnresolvedVariable)
}
