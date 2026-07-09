package aws

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildSpecCommands(t *testing.T, build compose.BuildConfig) []string {
	t.Helper()
	spec, err := getBuildSpec(build, "123.dkr.ecr.us-test-2.amazonaws.com/repo:latest")
	require.NoError(t, err)
	var parsed struct {
		Phases struct {
			PreBuild struct {
				Commands []string `json:"commands"`
			} `json:"pre_build"`
			Build struct {
				Commands []string `json:"commands"`
			} `json:"build"`
		} `json:"phases"`
	}
	require.NoError(t, json.Unmarshal([]byte(spec), &parsed))
	return append(parsed.Phases.PreBuild.Commands, parsed.Phases.Build.Commands...)
}

func TestGetBuildSpecDefault(t *testing.T) {
	cmds := buildSpecCommands(t, compose.BuildConfig{Context: pulumi.String("s3://bucket/ctx")})
	joined := strings.Join(cmds, "\n")
	assert.NotContains(t, joined, "--platform")
	assert.NotContains(t, joined, "--cache-from")
	assert.NotContains(t, joined, "--cache-to")
	assert.Contains(t, joined,
		"docker buildx build -t 123.dkr.ecr.us-test-2.amazonaws.com/repo:latest -f Dockerfile --push $CODEBUILD_SRC_DIR")
}

func TestGetBuildSpecPlatformsAndCache(t *testing.T) {
	cmds := buildSpecCommands(t, compose.BuildConfig{
		Context:   pulumi.String("s3://bucket/ctx"),
		Platforms: []string{"linux/arm64", "linux/amd64"},
		CacheFrom: []string{"type=registry,ref=my/app:cache"},
		CacheTo:   []string{"type=registry,mode=max,ref=my/app:cache"},
	})
	joined := strings.Join(cmds, "\n")
	assert.Contains(t, joined,
		"buildx create --use --driver=docker-container --use --platform linux/arm64,linux/amd64")
	assert.Contains(t, joined,
		"buildx build --platform linux/arm64,linux/amd64"+
			" --cache-from=type=registry,ref=my/app:cache"+
			" --cache-to=type=registry,mode=max,ref=my/app:cache -t ")
}
