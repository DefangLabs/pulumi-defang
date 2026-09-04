package aws

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parsePolicy(t *testing.T, doc string) lifecyclePolicy {
	t.Helper()
	var parsed lifecyclePolicy
	require.NoError(t, json.Unmarshal([]byte(doc), &parsed))
	return parsed
}

func buildPolicy(t *testing.T, keep int) string {
	t.Helper()
	doc, err := buildLifecyclePolicy(keep)
	require.NoError(t, err)
	return doc
}

func cachePolicy(t *testing.T, keep int) string {
	t.Helper()
	doc, err := cacheLifecyclePolicy(keep)
	require.NoError(t, err)
	return doc
}

func TestBuildLifecyclePolicy(t *testing.T) {
	policy := parsePolicy(t, buildPolicy(t, keepBuildImages))
	require.Len(t, policy.Rules, 2)

	untagged := policy.Rules[0]
	assert.Equal(t, 1, untagged.RulePriority)
	assert.Equal(t, "untagged", untagged.Selection.TagStatus)
	assert.Equal(t, "sinceImagePushed", untagged.Selection.CountType)
	assert.Equal(t, "days", untagged.Selection.CountUnit)
	assert.Equal(t, "expire", untagged.Action.Type)

	// Bounding the count is the whole point: without it the repo grows forever.
	keep := policy.Rules[1]
	assert.Equal(t, 2, keep.RulePriority)
	assert.Equal(t, "any", keep.Selection.TagStatus)
	assert.Equal(t, "imageCountMoreThan", keep.Selection.CountType)
	assert.Equal(t, keepBuildImages, keep.Selection.CountNumber)
	// Age would delete images that live-but-not-recently-rebuilt services need.
	assert.Empty(t, keep.Selection.CountUnit)
}

func TestCacheLifecyclePolicyHasNoUntaggedRule(t *testing.T) {
	policy := parsePolicy(t, cachePolicy(t, keepCacheImages))
	require.Len(t, policy.Rules, 1, "a cache needs exactly one count rule")

	rule := policy.Rules[0]
	assert.Equal(t, "any", rule.Selection.TagStatus)
	assert.Equal(t, keepCacheImages, rule.Selection.CountNumber)

	// Images pulled by digest arrive untagged, so an untagged rule would empty
	// the cache and push every deploy onto the upstream registry's rate limits.
	for _, r := range policy.Rules {
		assert.NotEqual(t, "untagged", r.Selection.TagStatus)
	}
}

func TestLifecyclePoliciesAreValidJSON(t *testing.T) {
	for name, doc := range map[string]string{
		"build": buildPolicy(t, 5),
		"cache": cachePolicy(t, 5),
	} {
		t.Run(name, func(t *testing.T) {
			assert.True(t, json.Valid([]byte(doc)))
			// ECR rejects a policy with an unknown or empty action type.
			for _, r := range parsePolicy(t, doc).Rules {
				assert.Equal(t, "expire", r.Action.Type)
				assert.NotEmpty(t, r.Description)
				assert.Positive(t, r.Selection.CountNumber)
			}
		})
	}
}
