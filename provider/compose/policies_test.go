package compose

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestClassifyPolicy(t *testing.T) {
	tests := []struct {
		entry string
		want  PolicyCloud
	}{
		{"arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess", PolicyCloudAWS},
		{"arn:aws:iam::123456789012:policy/deployer", PolicyCloudAWS},
		{"roles/run.developer", PolicyCloudGCP},
		{"projects/my-proj/roles/deployer", PolicyCloudGCP},
		{"organizations/123/roles/auditor", PolicyCloudGCP},
		{"/subscriptions/sub-id/providers/Microsoft.Authorization/roleDefinitions/x", PolicyCloudAzure},
		{"/providers/Microsoft.Authorization/roleDefinitions/x", PolicyCloudAzure},
		{"deployer", PolicyCloudAny},
		{"AmazonS3ReadOnlyAccess", PolicyCloudAny},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, ClassifyPolicy(tt.entry), "entry %q", tt.entry)
	}
}

func TestNormalizePolicies(t *testing.T) {
	// Comma-separated entries split (one interpolated ${VAR} can carry a list),
	// whitespace trims, empties drop (a "${VAR:-}" the stack leaves unset).
	got := NormalizePolicies([]string{
		"arn:aws:iam::aws:policy/A, arn:aws:iam::aws:policy/B",
		"",
		"  deployer  ",
		",",
	})
	assert.Equal(t, []string{
		"arn:aws:iam::aws:policy/A",
		"arn:aws:iam::aws:policy/B",
		"deployer",
	}, got)

	assert.Nil(t, NormalizePolicies(nil))
	assert.Nil(t, NormalizePolicies([]string{"", " ", ","}))
}

func TestValidatePolicies(t *testing.T) {
	// Current-cloud and bare entries pass.
	require.NoError(t, ValidatePolicies(PolicyCloudAWS, []string{
		"arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess", "deployer",
	}))
	require.NoError(t, ValidatePolicies(PolicyCloudGCP, []string{
		"roles/run.developer", "deployer",
	}))

	// A foreign identifier is a hard error (no cross-cloud filtering).
	err := ValidatePolicies(PolicyCloudAWS, []string{"roles/run.developer"})
	require.ErrorContains(t, err, "gcp identifier")
	require.ErrorContains(t, err, "targets aws")
	require.ErrorContains(t, err, "${VAR}")

	err = ValidatePolicies(PolicyCloudGCP, []string{"arn:aws:iam::aws:policy/A"})
	require.ErrorContains(t, err, "aws identifier")

	// An unresolved variable means compose-load interpolation had no value
	// for it; defang config is deliberately unsupported for policies.
	err = ValidatePolicies(PolicyCloudAWS, []string{"${POLICIES}"})
	require.ErrorContains(t, err, "unresolved variable")
	require.ErrorContains(t, err, "defang config")
}

func TestPolicyListUnmarshalYAML(t *testing.T) {
	// List form: entries normalized (split, trimmed, empties dropped).
	var list PolicyList
	require.NoError(t, yaml.Unmarshal([]byte(
		"- arn:aws:iam::aws:policy/A\n- \"\"\n- roles/run.developer, deployer\n"), &list))
	assert.Equal(t, PolicyList{
		"arn:aws:iam::aws:policy/A", "roles/run.developer", "deployer",
	}, list)

	// Scalar form: `x-defang-policies: ${POLICIES}` post-substitution is a
	// single comma-separated string.
	var scalar PolicyList
	require.NoError(t, yaml.Unmarshal([]byte(
		`"arn:aws:iam::aws:policy/A,arn:aws:iam::aws:policy/B"`), &scalar))
	assert.Equal(t, PolicyList{
		"arn:aws:iam::aws:policy/A", "arn:aws:iam::aws:policy/B",
	}, scalar)
}

func TestServiceConfigUnmarshalYAMLPoliciesScalar(t *testing.T) {
	// The scalar form must work through the full service decode, which uses
	// a methodless alias of ServiceConfig (PolicyList's own UnmarshalYAML
	// still applies to the field).
	input := `
image: myapp:latest
x-defang-policies: "roles/run.developer, deployer"
`
	var svc ServiceConfig
	require.NoError(t, yaml.Unmarshal([]byte(input), &svc))
	assert.Equal(t, PolicyList{"roles/run.developer", "deployer"}, svc.Policies)
}
