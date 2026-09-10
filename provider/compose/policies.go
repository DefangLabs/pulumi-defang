package compose

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ErrPolicyUnresolvedVariable rejects entries still containing `${…}`:
// compose variables are interpolated before the project reaches the
// provider (a CLI concern, e.g. from the stack's env files), and `defang
// config` is deliberately not supported for policies.
var ErrPolicyUnresolvedVariable = errors.New(
	"policy variables must be resolved when the compose file is loaded; " +
		"`defang config` is not supported for policies")

// PolicyList holds x-defang-policies entries. In YAML it accepts a sequence
// or a single scalar, and entries may hold several comma-separated
// identifiers — so one `${VAR}` interpolated at compose-load time can carry
// a variable-length list. Empty entries (a "${VAR:-}" the stack leaves
// unset) are dropped.
//
// What an entry means is the target cloud's business: each provider parses
// x-defang-policies in its own package (aws.ParsePolicies,
// gcp.ParsePolicies, azure.ParsePolicies), because a policy identifier is
// cloud-specific and nothing here can say which cloud a bare name belongs
// to. This file holds only what is true of every cloud: the YAML shape, and
// that an entry must be a literal by the time a provider sees it.
type PolicyList []string

// UnmarshalYAML accepts `x-defang-policies: ${POLICIES}` (scalar) in addition
// to the list form, normalizing either into individual entries.
func (p *PolicyList) UnmarshalYAML(value *yaml.Node) error {
	var entries []string
	if value.Kind == yaml.ScalarNode {
		var s string
		if err := value.Decode(&s); err != nil {
			return err
		}
		entries = []string{s}
	} else if err := value.Decode(&entries); err != nil {
		return err
	}
	*p = NormalizePolicies(entries)
	return nil
}

// NormalizePolicies flattens x-defang-policies entries: comma-separated
// identifiers within an entry are split out, whitespace is trimmed, and empty
// entries are dropped. Idempotent; call sites normalize again because inputs
// can also arrive as plain lists from Pulumi programs, bypassing the YAML
// path.
func NormalizePolicies(entries []string) []string {
	var out []string
	for _, entry := range entries {
		for _, part := range strings.Split(entry, ",") {
			if part = strings.TrimSpace(part); part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

// NormalizeLiteralPolicies normalizes entries and rejects any that is not a
// literal. Every provider's own ParsePolicies starts here, because compose
// interpolation is a property of the compose file rather than of any cloud:
// variables are substituted at compose-load time by the CLI, not by the
// CD/providers, so a `${…}` still present means the stack had no value for
// it and the deployment would otherwise attach a policy named after the
// variable.
func NormalizeLiteralPolicies(entries []string) ([]string, error) {
	policies := NormalizePolicies(entries)
	for _, entry := range policies {
		if strings.Contains(entry, "${") {
			return nil, fmt.Errorf("x-defang-policies entry %q has an unresolved variable: %w",
				entry, ErrPolicyUnresolvedVariable)
		}
	}
	return policies, nil
}

// PolicyVarHint is the tail of a provider's "this is not one of mine" error.
// A single compose file can target several clouds, and the entries that do
// not apply to all of them belong in a variable whose per-stack value carries
// the right identifier — there is no cross-cloud filtering of a literal list.
const PolicyVarHint = "; a compose file deployed to more than one cloud should carry " +
	"cloud-specific policies in a ${VAR} whose per-stack value is the identifier for that cloud"
