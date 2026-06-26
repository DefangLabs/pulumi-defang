package gcp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeLabel(t *testing.T) {
	tests := []struct {
		name        string
		inKey, inVal string
		wantKey, wantVal string
	}{
		{"already valid", "team", "core", "team", "core"},
		{"dotted key, mixed-case value", "com.acme.team", "Core", "com_acme_team", "core"},
		{"uppercase and spaces", "My Label", "Hello World", "my_label", "hello_world"},
		{"value keeps dashes and underscores", "k", "a-b_c", "k", "a-b_c"},
		{"digit-leading key gets letter prefix", "1st", "v", "k_1st", "v"},
		{"empty key falls back", "...", "v", "k____", "v"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, v := SanitizeLabel(tt.inKey, tt.inVal)
			assert.Equal(t, tt.wantKey, k)
			assert.Equal(t, tt.wantVal, v)
		})
	}
}

func TestSanitizeLabel_Truncates(t *testing.T) {
	long := strings.Repeat("a", 100)
	k, v := SanitizeLabel(long, long)
	assert.Len(t, k, 63)
	assert.Len(t, v, 63)
}

func TestSanitizeLabel_KeyStartsWithLetter(t *testing.T) {
	// A key that sanitizes to all-invalid leading chars must still start with a letter.
	k, _ := SanitizeLabel("9-9", "v")
	assert.Regexp(t, `^[a-z]`, k)
}
