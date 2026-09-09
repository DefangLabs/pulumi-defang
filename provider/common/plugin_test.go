package common

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
)

func TestPluginIdentityFromNormalizesVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		fallbackVersion string
		opts            []pulumi.ResourceOption
		want            string
	}{
		{name: "release tag", fallbackVersion: "v2.7.1", want: "2.7.1"},
		{name: "semver", fallbackVersion: "2.7.1", want: "2.7.1"},
		{name: "empty development build", want: ""},
		{
			name:            "option overrides fallback",
			fallbackVersion: "v2.7.1",
			opts:            []pulumi.ResourceOption{pulumi.Version("v2.8.0-beta.1")},
			want:            "2.8.0-beta.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			identity := PluginIdentityFrom(tt.fallbackVersion, tt.opts...)
			assert.Equal(t, tt.want, identity.Version)
		})
	}
}
