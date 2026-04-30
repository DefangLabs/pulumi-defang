package common

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"maps"
	"strings"
	"time"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

// DefaultBuildMaxWaitTime is the default max wait time (in seconds) for a build
// to complete across providers. Long enough to cover cold-cache builds of
// reasonably large images.
const DefaultBuildMaxWaitTime = 3600

// SleepWithContext sleeps for d, returning early with ctx.Err() if ctx is canceled.
// Use in poll loops so cancellation/timeout from Pulumi is honored without
// waiting for the next poll iteration.
func SleepWithContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func isEphemeralBuildArg(key string) bool {
	return strings.HasSuffix(key, "_TOKEN")
}

// removeEphemeralBuildArgs hides ephemeral build args (eg. GITHUB_TOKEN) so we get the same imageTag each CI run
func removeEphemeralBuildArgs(args map[string]string) map[string]string {
	args = maps.Clone(args) // shallow clone
	for key := range args {
		if isEphemeralBuildArg(key) {
			args[key] = "Removed ephemeral token"
		}
	}
	return args
}

func sha256hash(inputs ...[]byte) string {
	h := sha256.New() // sha1 was good enough but triggers linter warnings
	for _, c := range inputs {
		h.Write(c)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// BuildTriggerHash computes a hash of build inputs to trigger replacements when they change.
func BuildTriggerHash(build *compose.BuildConfig) pulumi.StringOutput {
	// Must also hash buildArgs, in case tarball is the same; stably serialize to a string
	argsStr, err := json.Marshal(removeEphemeralBuildArgs(build.Args))
	if err != nil {
		return pulumi.StringOutput{}
	}
	var dockerfile, target string
	if build.Dockerfile != nil {
		dockerfile = *build.Dockerfile
	}
	if build.Target != nil {
		target = *build.Target
	}
	return pulumi.StringOutput(pulumix.Apply(
		pulumix.Output[string](build.Context.ToStringOutput()), func(ctx string) string {
			contextEtag, _, _ := strings.Cut(ctx, "?") // remove sig query param; FIXME: get actual etag from URL, not path
			return sha256hash([]byte(contextEtag), argsStr, []byte(dockerfile), []byte(target))[0:8]
		}))
}
