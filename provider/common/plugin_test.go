package common

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginIdentityFrom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		opts        []pulumi.ResourceOption
		wantVersion string
		wantURL     string
	}{
		{
			// The regression this guards: an unpinned identity must stay
			// unpinned. Whatever this build's linker-stamped Version happens to
			// be — a release tag, the Dockerfile's 0.0.1 placeholder, a
			// pulumictl pre-release string — it must not reach the checkpoint,
			// because only a version that names a published GitHub release can
			// ever be resolved again by a different CD image.
			name:        "no options pins no version",
			wantVersion: "",
			wantURL:     PluginDownloadURL,
		},
		{
			name:        "caller may pin a version",
			opts:        []pulumi.ResourceOption{pulumi.Version("2.7.1")},
			wantVersion: "2.7.1",
			wantURL:     PluginDownloadURL,
		},
		{
			// pulumi.Version's parser rejects a leading "v".
			name:        "a pinned tag is normalized to semver",
			opts:        []pulumi.ResourceOption{pulumi.Version("v2.8.0-beta.1")},
			wantVersion: "2.8.0-beta.1",
			wantURL:     PluginDownloadURL,
		},
		{
			name:        "caller may override the download URL",
			opts:        []pulumi.ResourceOption{pulumi.PluginDownloadURL("github://api.github.com/acme/fork")},
			wantVersion: "",
			wantURL:     "github://api.github.com/acme/fork",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			identity := PluginIdentityFrom(tt.opts...)
			assert.Equal(t, tt.wantVersion, identity.Version)
			assert.Equal(t, tt.wantURL, identity.DownloadURL)
		})
	}
}

func TestPluginIdentityResourceOptions(t *testing.T) {
	t.Parallel()

	t.Run("unpinned identity emits a URL and no version", func(t *testing.T) {
		t.Parallel()

		snapshot, err := pulumi.NewResourceOptions(PluginIdentityFrom().ResourceOptions()...)
		require.NoError(t, err)
		assert.Equal(t, PluginDownloadURL, snapshot.PluginDownloadURL)
		assert.Empty(t, snapshot.Version,
			"a versioned registration is resolvable only by the exact build that wrote it")
	})

	t.Run("a pinned identity emits both", func(t *testing.T) {
		t.Parallel()

		id := PluginIdentityFrom(pulumi.Version("v2.7.1"))
		snapshot, err := pulumi.NewResourceOptions(id.ResourceOptions()...)
		require.NoError(t, err)
		assert.Equal(t, PluginDownloadURL, snapshot.PluginDownloadURL)
		assert.Equal(t, "2.7.1", snapshot.Version)
	})
}
