package azure

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
)

func TestAzurePostgresMajorVersion(t *testing.T) {
	tests := []struct {
		name  string
		input *string
		want  *string
	}{
		{"nil", nil, nil},
		{"minor version from image tag", pulumi.StringRef("18.6"), pulumi.StringRef("18")},
		{"bookworm-style suffix", pulumi.StringRef("16.3-bookworm"), pulumi.StringRef("16")},
		{"bare major version", pulumi.StringRef("17"), pulumi.StringRef("17")},
		{"unparseable falls back unchanged", pulumi.StringRef("latest"), pulumi.StringRef("latest")},
		{"empty falls back unchanged", pulumi.StringRef(""), pulumi.StringRef("")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := azurePostgresMajorVersion(tt.input)
			if tt.want == nil {
				assert.Nil(t, got)
				return
			}
			if assert.NotNil(t, got) {
				assert.Equal(t, *tt.want, *got)
			}
		})
	}
}
