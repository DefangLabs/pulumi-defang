package gcp

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCdSourceBucket checks that the shared CD bucket is parsed out of the
// defang:stateUrl config the CD program sets from DEFANG_STATE_URL, and that a
// non-GCS (or absent) value yields no bucket instead of a bogus name.
func TestCdSourceBucket(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		expected string
	}{
		{name: "unset", config: `{}`, expected: ""},
		{name: "gcs url", config: `{"defang:stateUrl": "gs://defang-cd-abc123"}`, expected: "defang-cd-abc123"},
		{
			name:     "gcs url with path",
			config:   `{"defang:stateUrl": "gs://defang-cd-abc123/pulumi"}`,
			expected: "defang-cd-abc123",
		},
		{name: "s3 url", config: `{"defang:stateUrl": "s3://defang-cd-abc123"}`, expected: ""},
		{name: "not a url", config: `{"defang:stateUrl": ":://"}`, expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PULUMI_CONFIG", tt.config)
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				assert.Equal(t, tt.expected, cdSourceBucket(ctx))
				return nil
			}, pulumi.WithMocks("proj", "stack", testMocks{}))
			require.NoError(t, err)
		})
	}
}
