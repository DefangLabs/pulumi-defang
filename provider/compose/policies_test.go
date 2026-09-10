package compose

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

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

func TestNormalizeLiteralPolicies(t *testing.T) {
	// Normalizes like NormalizePolicies, and says nothing about what an entry
	// means — that is each cloud's ParsePolicies.
	got, err := NormalizeLiteralPolicies([]string{"roles/run.developer, deployer", ""})
	require.NoError(t, err)
	assert.Equal(t, []string{"roles/run.developer", "deployer"}, got)

	// An unresolved variable means compose-load interpolation had no value
	// for it; defang config is deliberately unsupported for policies.
	_, err = NormalizeLiteralPolicies([]string{"${POLICIES}"})
	require.ErrorContains(t, err, "unresolved variable")
	require.ErrorContains(t, err, "defang config")
	require.ErrorIs(t, err, ErrPolicyUnresolvedVariable)
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
