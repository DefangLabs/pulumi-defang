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
		"buildx create --use --driver=docker-container --buildkitd-config=/tmp/buildkitd.toml"+
			" --driver-opt network=host --use --platform linux/arm64,linux/amd64")
	assert.Contains(t, joined,
		"buildx build --platform linux/arm64,linux/amd64"+
			" --cache-from=type=registry,ref=my/app:cache"+
			" --cache-to=type=registry,mode=max,ref=my/app:cache -t ")
}

// indexOfCommand returns the index of the first command containing substr, or -1.
func indexOfCommand(cmds []string, substr string) int {
	for i, c := range cmds {
		if strings.Contains(c, substr) {
			return i
		}
	}
	return -1
}

func TestGetBuildSpecDockerhubMirror(t *testing.T) {
	cmds := buildSpecCommands(t, compose.BuildConfig{Context: pulumi.String("s3://bucket/ctx")})
	joined := strings.Join(cmds, "\n")

	// Daemon and BuildKit both point at the local mirror.
	assert.Contains(t, joined, `"registry-mirrors":["http://localhost:5000"]`)
	assert.Contains(t, joined, `> /etc/docker/daemon.json`)
	assert.Contains(t, joined, `[registry."docker.io"]`)
	assert.Contains(t, joined, "> /tmp/buildkitd.toml")
	assert.Contains(t, joined, "kill -HUP $(pidof dockerd)")

	// The mirror itself: token fetch, config render, container start.
	assert.Contains(t, joined, "docker-credential-ecr-login get")
	assert.Contains(t, joined, "> /tmp/nginx.conf")
	assert.Contains(t, joined, "proxy_pass https://public.ecr.aws/v2/docker/")
	assert.Contains(t, joined, "docker rm -f dockerhub-ecr-mirror || true")
	assert.Contains(t, joined, "docker run -d --rm -p 5000:80 --name dockerhub-ecr-mirror")
	assert.Contains(t, joined, "-v /tmp/nginx.conf:/etc/nginx/nginx.conf:ro")

	// The daemon.json fragment (written via a quoted heredoc, not `echo '...'`, so a mirror value
	// containing a single quote can't break out of the shell string) must be valid JSON, and Docker
	// wants the mirror as a full URL there.
	daemonCmd := cmds[indexOfCommand(cmds, "/etc/docker/daemon.json")]
	_, jsonPart, ok := strings.Cut(daemonCmd, "daemon.json <<'EOF'\n")
	require.True(t, ok)
	jsonPart, _, ok = strings.Cut(jsonPart, "\nEOF")
	require.True(t, ok)
	var daemonCfg map[string][]string
	require.NoError(t, json.Unmarshal([]byte(jsonPart), &daemonCfg))
	assert.Equal(t, []string{"http://localhost:5000"}, daemonCfg["registry-mirrors"])

	// BuildKit's buildkitd.toml wants bare host:port (no scheme) plus a separate http=true flag
	// for a plaintext mirror, unlike daemon.json above (https://docs.docker.com/build/buildkit/toml-configuration/).
	buildkitdCmd := cmds[indexOfCommand(cmds, "/tmp/buildkitd.toml")]
	_, tomlPart, ok := strings.Cut(buildkitdCmd, "buildkitd.toml <<'EOF'")
	require.True(t, ok)
	tomlPart, _, ok = strings.Cut(tomlPart, "EOF")
	require.True(t, ok)
	assert.Contains(t, tomlPart, `mirrors = [`+"\n    \"localhost:5000\"\n  ]")
	assert.NotContains(t, tomlPart, "http://localhost:5000")
	assert.Contains(t, tomlPart, "\n  http = true")
}

// The mirror container must never get a restart policy: on a reused CodeBuild host the daemon
// would resurrect it from a stale /tmp/nginx.conf bind-mount spec, and Docker creates a missing
// bind-mount source as a directory, breaking the next build's `awk ... > /tmp/nginx.conf`.
// See https://github.com/DefangLabs/defang-mvp/issues/2869
func TestGetBuildSpecMirrorHasNoRestartPolicy(t *testing.T) {
	cmds := buildSpecCommands(t, compose.BuildConfig{Context: pulumi.String("s3://bucket/ctx")})
	assert.NotContains(t, strings.Join(cmds, "\n"), "--restart")
}

func TestGetBuildSpecMirrorStepOrder(t *testing.T) {
	cmds := buildSpecCommands(t, compose.BuildConfig{Context: pulumi.String("s3://bucket/ctx")})

	publicEcrLogin := indexOfCommand(cmds, "docker login --username AWS --password-stdin public.ecr.aws")
	hup := indexOfCommand(cmds, "kill -HUP $(pidof dockerd)")
	rm := indexOfCommand(cmds, "docker rm -f dockerhub-ecr-mirror")
	run := indexOfCommand(cmds, "docker run -d --rm -p 5000:80")
	buildxCreate := indexOfCommand(cmds, "docker buildx create")
	buildxBuild := indexOfCommand(cmds, "docker buildx build")

	for _, i := range []int{publicEcrLogin, hup, rm, run, buildxCreate, buildxBuild} {
		require.NotEqual(t, -1, i)
	}
	// Log in before pulling the mirror image; reload the daemon before starting the mirror;
	// remove any leftover container before starting ours; have the mirror up before buildx runs.
	assert.Less(t, publicEcrLogin, hup)
	assert.Less(t, hup, rm)
	assert.Less(t, rm, run)
	assert.Less(t, run, buildxCreate)
	assert.Less(t, buildxCreate, buildxBuild)
}

func TestGetSetupMirrorStepsEmpty(t *testing.T) {
	steps, err := getSetupMirrorSteps(nil)
	require.NoError(t, err)
	assert.Empty(t, steps)
}
