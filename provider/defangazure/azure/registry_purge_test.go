package azure

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPurgeTaskYAML(t *testing.T) {
	doc, err := purgeTaskYAML(SharedBuildRepo)
	require.NoError(t, err)

	var spec acrPurgeTaskSpec
	require.NoError(t, yaml.Unmarshal([]byte(doc), &spec))
	require.Len(t, spec.Steps, 2)

	untagged := spec.Steps[0]
	assert.Contains(t, untagged.Cmd, `--filter "builds:.*"`)
	assert.Contains(t, untagged.Cmd, "--untagged")
	assert.Contains(t, untagged.Cmd, "--ago 1d")
	assert.NotContains(t, untagged.Cmd, "--keep")

	// Bounding the count is the whole point: without it the repo grows forever.
	keepNewest := spec.Steps[1]
	assert.NotContains(t, keepNewest.Cmd, "--untagged")
	assert.Contains(t, keepNewest.Cmd, "--keep 20")

	for _, step := range spec.Steps {
		assert.Contains(t, step.Cmd, acrCliImage)
		assert.True(t, step.DisableWorkingDirectoryOverride)
		assert.Positive(t, step.Timeout)
	}
	assert.Equal(t, common.ExpireUntaggedDays, 1)
	assert.Equal(t, common.KeepBuildImages, 20)
}
