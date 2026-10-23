package aws

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The exact string that failed a live deploy of the mastra-extended sample to
// project "mastra-extended", stack "e2eaws": the 30-character cut landed on a
// hyphen, and AWS rejected the prefix.
func TestCachePrefixNameDoesNotEndInASeparator(t *testing.T) {
	const seed = "defang-mastra-extended-e2eaws-mastra-extended-ecr-public"

	got := cachePrefixName(seed)

	require.LessOrEqual(t, len(got), maxCacheRepoLength)
	assert.NotEqual(t, "defang-mastra-extended-e2eaws-", got, "the truncation that AWS rejected")
	assert.NotEmpty(t, strings.TrimRight(got, cachePrefixSeparators),
		"a prefix of only separators is not a name")
	assert.Equal(t, strings.TrimRight(got, cachePrefixSeparators), got,
		"ecrRepositoryPrefix may not end in a separator")
}

// The prefix is an account-global namespace, so two projects that agree for
// the first 30 characters must not claim one rule.
func TestCachePrefixNameLongSeedsDoNotCollide(t *testing.T) {
	shared := "defang-" + strings.Repeat("verylongproject", 3)

	first := cachePrefixName(shared + "-alpha-ecr-public")
	second := cachePrefixName(shared + "-bravo-ecr-public")

	require.LessOrEqual(t, len(first), maxCacheRepoLength)
	require.LessOrEqual(t, len(second), maxCacheRepoLength)
	assert.NotEqual(t, first, second)
	assert.Equal(t, first, cachePrefixName(shared+"-alpha-ecr-public"), "must be deterministic")
}

// A seed that already fits is left exactly as it is: existing rules keep their
// prefix, and this is the common case.
func TestCachePrefixNameShortSeedsAreUnchanged(t *testing.T) {
	assert.Equal(t, "defang-app-prod-ecr-public", cachePrefixName("defang-app-prod-ecr-public"))
	assert.Len(t, cachePrefixName(strings.Repeat("a", maxCacheRepoLength)), maxCacheRepoLength)
}
