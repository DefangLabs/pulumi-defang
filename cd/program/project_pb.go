package program

import (
	"fmt"
	"net/url"
	"os"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const protobufContentType = "application/protobuf"

func parseStateURL(ctx *pulumi.Context) (*url.URL, error) {
	stateURL := common.String("stateUrl", "").Get(ctx)
	if stateURL == "" {
		return nil, nil
	}
	u, err := url.Parse(stateURL)
	if err != nil {
		return nil, fmt.Errorf("invalid DEFANG_STATE_URL %q: %w", stateURL, err)
	}
	return u, nil
}

// projectPbKey returns the object-store key for the ProjectUpdate protobuf:
// `projects/{project}/{stack}/project.pb`. For Azure, the `projects/` prefix
// is stripped by the caller since Azure uses a dedicated `projects` container.
func projectPbKey(ctx *pulumi.Context) string {
	return fmt.Sprintf("projects/%s/%s/project.pb", ctx.Project(), ctx.Stack())
}

// NewTempFileAsset writes data to a temp file and returns a FileAsset referencing it.
// Pulumi's StringAsset rejects non-UTF-8 data (gRPC marshal error), so binary
// protobufs must go through the filesystem. The temp file is left for the OS
// to reap — it's a few KB and Pulumi may still be reading it after this returns.
// Passing "" picks $TMPDIR so the OS reap-on-reboot story actually applies.
func NewTempFileAsset(pattern string, data []byte) (pulumi.Asset, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return nil, fmt.Errorf("creating temp file for project.pb: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return nil, fmt.Errorf("writing temp file for project.pb: %w", err)
	}
	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("closing temp file for project.pb: %w", err)
	}
	return pulumi.NewFileAsset(f.Name()), nil
}
