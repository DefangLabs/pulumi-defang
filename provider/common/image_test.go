package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseImage(t *testing.T) {
	tests := []struct {
		input            string
		wantRegistry     string
		wantRepo         string
		wantTag          string
		wantRoundTripped string // fullImage() result; defaults to input if empty
	}{
		{
			input:    "nginx",
			wantRepo: "nginx",
		},
		{
			input:    "nginx:latest",
			wantRepo: "nginx",
			wantTag:  "latest",
		},
		{
			input:        "gcr.io/my-project/myapp:v1",
			wantRegistry: "gcr.io",
			wantRepo:     "my-project/myapp",
			wantTag:      "v1",
		},
		{
			input:        "us-central1-docker.pkg.dev/proj/repo/img:tag",
			wantRegistry: "us-central1-docker.pkg.dev",
			wantRepo:     "proj/repo/img",
			wantTag:      "tag",
		},
		{
			input:        "quay.io/prometheus/node-exporter:v1.8.0",
			wantRegistry: "quay.io",
			wantRepo:     "prometheus/node-exporter",
			wantTag:      "v1.8.0",
		},
		{
			input:        "ghcr.io/owner/image:sha-abc123",
			wantRegistry: "ghcr.io",
			wantRepo:     "owner/image",
			wantTag:      "sha-abc123",
		},
		{
			input:        "docker.io/library/nginx:1.25",
			wantRegistry: "docker.io",
			wantRepo:     "library/nginx",
			wantTag:      "1.25",
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseImage(tt.input)
			assert.Equal(t, tt.wantRegistry, got.Registry, "registry")
			assert.Equal(t, tt.wantRepo, got.Repo, "repo")
			assert.Equal(t, tt.wantTag, got.Tag, "tag")
			want := tt.wantRoundTripped
			if want == "" {
				want = tt.input
			}
			assert.Equal(t, want, got.FullImage(), "fullImage round-trip")
		})
	}
}
