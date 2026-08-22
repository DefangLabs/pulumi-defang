package aws

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildSpecCommands generates a buildspec for a realistic .tar.gz build context. Tests that
// specifically exercise the context-extraction step should call buildSpecCommandsWithContext
// instead, so they control the source URL (and therefore the archive filename/extension).
func buildSpecCommands(t *testing.T, build compose.BuildConfig) []string {
	t.Helper()
	return buildSpecCommandsWithContext(t, build, "s3://bucket/context.tar.gz")
}

func buildSpecCommandsWithContext(t *testing.T, build compose.BuildConfig, contextURL string) []string {
	t.Helper()
	spec, err := getBuildSpec(build, "123.dkr.ecr.us-test-2.amazonaws.com/repo:latest", contextURL)
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

func TestNormalizeCodeBuildS3Location(t *testing.T) {
	tests := map[string]string{
		"s3://bucket/uploads/context.tar.gz":                                     "bucket/uploads/context.tar.gz",
		"https://bucket.s3.amazonaws.com/uploads/context.tar.gz?signature=value": "bucket/uploads/context.tar.gz",
		"https://bucket.s3.us-west-2.amazonaws.com/uploads/context.tar.gz":       "bucket/uploads/context.tar.gz",
		"https://s3.us-west-2.amazonaws.com/bucket/uploads/context.tar.gz":       "bucket/uploads/context.tar.gz",
		// Legacy dash form of the regional endpoint, in both styles.
		"https://bucket.s3-us-west-2.amazonaws.com/uploads/context.tar.gz": "bucket/uploads/context.tar.gz",
		"https://s3-us-west-2.amazonaws.com/bucket/uploads/context.tar.gz": "bucket/uploads/context.tar.gz",
		// The legacy global endpoint, path style.
		"https://s3.amazonaws.com/bucket/uploads/context.tar.gz": "bucket/uploads/context.tar.gz",
		// IPv6 dual-stack endpoints, in both virtual-hosted and path style. The
		// SDK emits these when AWS_USE_DUALSTACK_ENDPOINT is set.
		"https://bucket.s3.dualstack.us-west-2.amazonaws.com/uploads/context.tar.gz": "bucket/uploads/context.tar.gz",
		"https://s3.dualstack.us-west-2.amazonaws.com/bucket/uploads/context.tar.gz": "bucket/uploads/context.tar.gz",
	}
	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			actual, err := normalizeCodeBuildS3Location(input)
			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		})
	}
}

func TestNormalizeCodeBuildS3LocationRejectsInvalidURL(t *testing.T) {
	tests := []string{
		"https://example.com/context.tar.gz",
		// Lookalike hosts: a naive substring/prefix check on the hostname
		// (".s3." anywhere, or a "s3." prefix) would accept these as if they
		// were real S3 endpoints.
		"https://bucket.s3.evil.example/context.tar.gz",
		"https://s3.evil.example/bucket/context.tar.gz",
		"https://bucket.s3.amazonaws.com.evil.example/context.tar.gz",
		// The dual-stack label must not become an escape hatch for lookalikes.
		"https://s3.dualstack.evil.example/bucket/context.tar.gz",
		"https://bucket.s3.dualstack.us-west-2.amazonaws.com.evil.example/context.tar.gz",
		// Dual-stack endpoints are always regional -- there is no global
		// "s3.dualstack.amazonaws.com", so "dualstack" must never be read as
		// the region itself, in either addressing style.
		"https://s3.dualstack.amazonaws.com/bucket/context.tar.gz",
		"https://bucket.s3.dualstack.amazonaws.com/context.tar.gz",
		// The legacy dash form needs a region after the dash: a bare "s3-"
		// is not an endpoint, in either addressing style.
		"https://s3-.amazonaws.com/bucket/context.tar.gz",
		"https://bucket.s3-.amazonaws.com/context.tar.gz",
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := normalizeCodeBuildS3Location(input)
			require.Error(t, err)
		})
	}
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

// CodeBuild's S3 source only auto-extracts .zip; the CLI uploads the build
// context as .tar.gz for every non-Railpack build on every provider (GCP's
// Cloud Build extracts that natively, CodeBuild does not). A live smoketest
// (defang-mvp#3181) hit this for real: the build phase failed with
// "open Dockerfile: no such file or directory" because $CODEBUILD_SRC_DIR
// held the untouched archive, not its contents.
func TestGetBuildSpecExtractsArchiveBeforeAnythingElse(t *testing.T) {
	cmds := buildSpecCommands(t, compose.BuildConfig{})
	require.NotEmpty(t, cmds)
	assert.Contains(t, cmds[0], "tar xzf",
		"extraction must be the first pre_build command, before anything else needs the source tree")
	assert.Contains(t, cmds[0], "context.tar.gz",
		"must name the archive CodeBuild will actually download, taken from the source URL")
	assert.NotContains(t, cmds[0], "ls ",
		"the archive name is known up front from the source URL; no need to glob-discover it")
}

// The legacy TS CD also accepted ".tgz" (not just ".tar.gz"); keep parity.
func TestGetBuildSpecExtractsTgzArchive(t *testing.T) {
	cmds := buildSpecCommandsWithContext(t, compose.BuildConfig{}, "s3://bucket/uploads/context.tgz")
	require.NotEmpty(t, cmds)
	assert.Contains(t, cmds[0], "tar xzf")
	assert.Contains(t, cmds[0], "context.tgz")
}

// A .zip source is already extracted by CodeBuild itself, so there's nothing named
// *.tar.gz/*.tgz to find -- getBuildSpec must not emit an extraction step at all.
func TestGetBuildSpecSkipsExtractionForZipSource(t *testing.T) {
	cmds := buildSpecCommandsWithContext(t, compose.BuildConfig{}, "s3://bucket/uploads/context.zip")
	assert.NotContains(t, strings.Join(cmds, "\n"), "tar xzf")
}

// Only the archive's basename is used, even when the S3 object key has a path prefix
// (as real CLI uploads do, e.g. "uploads/<hash>.tar.gz") -- CodeBuild downloads the object
// into $CODEBUILD_SRC_DIR under that basename, not the full key.
func TestGetBuildSpecExtractionUsesArchiveBasename(t *testing.T) {
	cmds := buildSpecCommandsWithContext(t, compose.BuildConfig{}, "s3://bucket/uploads/nested/abc123.tar.gz")
	require.NotEmpty(t, cmds)
	assert.Contains(t, cmds[0], "abc123.tar.gz")
	assert.NotContains(t, cmds[0], "uploads/nested")
}

// Runs the actual generated extraction command (not just asserting on the spec text)
// against a real tarball, confirming it extracts the exact archive named in the command
// and removes it afterward.
func TestBuildSpecExtractionCommandExtractsRealArchive(t *testing.T) {
	cmds := buildSpecCommandsWithContext(t, compose.BuildConfig{}, "s3://bucket/uploads/abc123.tar.gz")
	extractCmd := cmds[0]

	dir := t.TempDir()
	archivePath := filepath.Join(dir, "abc123.tar.gz")
	writeTestTarGz(t, archivePath, map[string]string{"Dockerfile": "FROM scratch\n"})

	cmd := exec.CommandContext(t.Context(), "bash", "-c", extractCmd) //nolint:gosec // G204: test-authored command
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "output: %s", output)

	//nolint:gosec // G304: test-authored path in t.TempDir()
	content, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	require.NoError(t, err)
	assert.Equal(t, "FROM scratch\n", string(content))

	// Not removing it risks a naive `COPY . .` Dockerfile picking up its
	// own multi-megabyte source tarball as part of the image.
	_, err = os.Stat(archivePath)
	assert.True(t, os.IsNotExist(err), "archive should be removed after extraction")
}

func writeTestTarGz(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path) //nolint:gosec // G304: test-authored path in t.TempDir()
	require.NoError(t, err)
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Size: int64(len(content)), Mode: 0o644}))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}
	// Close in dependency order: tar writer flushes into gzip, gzip flushes into the file.
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())
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
