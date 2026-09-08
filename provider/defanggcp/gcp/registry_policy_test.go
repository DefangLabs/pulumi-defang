package gcp

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/artifactregistry"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCleanupPolicies(t *testing.T) {
	policies := buildCleanupPolicies()
	require.Len(t, policies, 3)

	untagged, ok := policies[0].(*artifactregistry.RepositoryCleanupPolicyArgs)
	require.True(t, ok)
	assert.Equal(t, "expire-untagged", pulumiString(t, untagged.Id))
	assert.Equal(t, "DELETE", pulumiString(t, untagged.Action))
	cond, ok := untagged.Condition.(*artifactregistry.RepositoryCleanupPolicyConditionArgs)
	require.True(t, ok)
	assert.Equal(t, "UNTAGGED", pulumiString(t, cond.TagState))
	assert.Equal(t, "1d", pulumiString(t, cond.OlderThan))

	// Bounding the count is the whole point: without it the repo grows forever.
	keepNewest, ok := policies[1].(*artifactregistry.RepositoryCleanupPolicyArgs)
	require.True(t, ok)
	assert.Equal(t, "KEEP", pulumiString(t, keepNewest.Action))
	mostRecent, ok := keepNewest.MostRecentVersions.(*artifactregistry.RepositoryCleanupPolicyMostRecentVersionsArgs)
	require.True(t, ok)
	keepCount, ok := mostRecent.KeepCount.(pulumi.Int)
	require.True(t, ok)
	assert.Equal(t, common.KeepBuildImages, int(keepCount))

	expireRest, ok := policies[2].(*artifactregistry.RepositoryCleanupPolicyArgs)
	require.True(t, ok)
	assert.Equal(t, "DELETE", pulumiString(t, expireRest.Action))
	restCond, ok := expireRest.Condition.(*artifactregistry.RepositoryCleanupPolicyConditionArgs)
	require.True(t, ok)
	assert.Equal(t, "ANY", pulumiString(t, restCond.TagState))
}

func pulumiString(t *testing.T, v any) string {
	t.Helper()
	s, ok := v.(pulumi.String)
	require.True(t, ok, "expected pulumi.String, got %T", v)
	return string(s)
}
