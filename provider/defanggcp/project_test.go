package defanggcp

import (
	"regexp"
	"strings"
	"testing"

	providergcp "github.com/DefangLabs/pulumi-defang/provider/defanggcp/gcp"
	"github.com/stretchr/testify/assert"
)

var validServiceAccountIdRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// --- serviceAccountId ---

func TestServiceAccountIdFitsWithinGCPLimit(t *testing.T) {
	// GCP adds an 8-char suffix; our prefix must be ≤ 22 chars so the total stays ≤ 30.
	id := serviceAccountId("proj", "svc", providergcp.GlobalConfig{Stack: "dev"})
	assert.LessOrEqual(t, len(id), 22)
}

func TestServiceAccountIdLongNameIsTrimmedWithHash(t *testing.T) {
	id := serviceAccountId(
		"averylongprojectname",
		"averylongservicename",
		providergcp.GlobalConfig{Stack: "averylongstackname"},
	)
	assert.Len(t, id, 22, "long name should be trimmed to exactly 22 chars")
}

func TestServiceAccountIdIsLowercase(t *testing.T) {
	id := serviceAccountId("MyProject", "MyService", providergcp.GlobalConfig{Stack: "Dev"})
	assert.Equal(t, strings.ToLower(id), id)
}

func TestServiceAccountIdOnlyContainsValidChars(t *testing.T) {
	id := serviceAccountId("MyProject", "my_svc-123", providergcp.GlobalConfig{Stack: "dev"})
	assert.Regexp(t, validServiceAccountIdRe, id)
}

func TestServiceAccountIdIsStable(t *testing.T) {
	cfg := providergcp.GlobalConfig{Stack: "dev"}
	first := serviceAccountId("proj", "svc", cfg)
	second := serviceAccountId("proj", "svc", cfg)
	assert.Equal(t, first, second)
}

func TestServiceAccountIdWithPrefix(t *testing.T) {
	withPrefix := serviceAccountId("proj", "svc", providergcp.GlobalConfig{Stack: "dev", Prefix: "tenant"})
	withoutPrefix := serviceAccountId("proj", "svc", providergcp.GlobalConfig{Stack: "dev"})
	assert.NotEqual(t, withPrefix, withoutPrefix, "prefix should affect the account ID")
}

// --- replaceNonAlphaNumericOrDash ---

func TestReplaceNonAlphaNumericOrDash(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello-world", "hello-world"},
		{"Hello World", "hello-world"},
		{"my_service", "my-service"},
		{"foo...bar", "foo---bar"},
		{"trailing-", "trailing"},
		{"UPPER", "upper"},
		{"mixed123", "mixed123"},
		{"a!b@c#d", "a-b-c-d"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.want, replaceNonAlphaNumericOrDash(tc.input))
		})
	}
}

// --- hashTrim ---

func TestHashTrimShortNameUnchanged(t *testing.T) {
	assert.Equal(t, "short", hashTrim("short", 10))
}

func TestHashTrimExactLengthUnchanged(t *testing.T) {
	name := "exact12345" // 10 chars
	assert.Equal(t, name, hashTrim(name, 10))
}

func TestHashTrimLongNameTrimmedToMaxLength(t *testing.T) {
	long := "abcdefghijklmnopqrstuvwxyz"
	result := hashTrim(long, 10)
	assert.Len(t, result, 10)
}

func TestHashTrimIsStable(t *testing.T) {
	long := "abcdefghijklmnopqrstuvwxyz"
	first := hashTrim(long, 10)
	second := hashTrim(long, 10)
	assert.Equal(t, first, second)
}

func TestHashTrimDifferentLongNamesProduceDifferentResults(t *testing.T) {
	a := hashTrim("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 10)
	b := hashTrim("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 10)
	assert.NotEqual(t, a, b, "different long inputs should produce different trimmed results")
}

// --- fullDefangResourceName ---

func TestFullDefangResourceNameWithoutPrefix(t *testing.T) {
	cfg := providergcp.GlobalConfig{Stack: "dev"}
	assert.Equal(t, "proj-dev-svc", fullDefangResourceName("proj", cfg, "svc"))
}

func TestFullDefangResourceNameWithPrefix(t *testing.T) {
	cfg := providergcp.GlobalConfig{Stack: "dev", Prefix: "tenant"}
	assert.Equal(t, "tenant-proj-dev-svc", fullDefangResourceName("proj", cfg, "svc"))
}

func TestFullDefangResourceNameMultipleNames(t *testing.T) {
	cfg := providergcp.GlobalConfig{Stack: "prod"}
	assert.Equal(t, "proj-prod-a-b", fullDefangResourceName("proj", cfg, "a", "b"))
}

func TestFullDefangResourceNameNoExtraNames(t *testing.T) {
	cfg := providergcp.GlobalConfig{Stack: "prod"}
	assert.Equal(t, "proj-prod", fullDefangResourceName("proj", cfg))
}
