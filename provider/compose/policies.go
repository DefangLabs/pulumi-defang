package compose

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	// ErrPolicyUnresolvedVariable rejects entries still containing `${…}`:
	// compose variables are interpolated before the project reaches the
	// provider (a CLI concern, e.g. from the stack's env files), and `defang
	// config` is deliberately not supported for policies.
	ErrPolicyUnresolvedVariable = errors.New(
		"policy variables must be resolved when the compose file is loaded; " +
			"`defang config` is not supported for policies")
	// ErrPolicyForeignCloud rejects an identifier whose syntax belongs to a
	// different cloud: there is no cross-cloud filtering.
	ErrPolicyForeignCloud = errors.New(
		"use a ${VAR} entry whose per-stack value carries the identifier for the targeted cloud")
	// ErrPolicyMalformedScope rejects a scoped entry with an empty half:
	// "Contributor@" names no scope, "@subscription" names no role.
	ErrPolicyMalformedScope = errors.New(
		"a scoped policy is written ROLE@SCOPE, where SCOPE is `subscription` or a full Azure scope resource ID")
)

const (
	// PolicyScopeSeparator separates an entry's role from the scope the role
	// is granted at: "Contributor@subscription". Azure-only syntax, because
	// Azure is the only one of the three clouds where a role is a capability
	// list with no reach of its own — an AWS managed policy applies wherever
	// it is attached, and a GCP role binds at the project.
	PolicyScopeSeparator = "@"
	// PolicyScopeSubscription grants at the whole subscription the deployment
	// runs in, rather than at the project's own resource group. Needed by a
	// service that manages resources outside its project: creating a resource
	// group is a write at subscription scope, which a resource-group-scoped
	// Contributor cannot do however broad the role is.
	PolicyScopeSubscription = "subscription"
)

// SplitPolicyScope splits "ROLE@SCOPE" into the role and the scope. An entry
// with no separator yields an empty scope, which the provider reads as its own
// default (on Azure, the project's resource group). Splits at the LAST
// separator, so a custom role whose own name contains one still parses.
func SplitPolicyScope(entry string) (string, string) {
	if i := strings.LastIndex(entry, PolicyScopeSeparator); i >= 0 {
		return entry[:i], entry[i+len(PolicyScopeSeparator):]
	}
	return entry, ""
}

// PolicyCloud identifies which cloud an x-defang-policies entry targets.
type PolicyCloud string

const (
	PolicyCloudAWS   PolicyCloud = "aws"
	PolicyCloudGCP   PolicyCloud = "gcp"
	PolicyCloudAzure PolicyCloud = "azure"
	// PolicyCloudAny is a bare name: a custom policy/role in the current
	// account/project, applicable on whichever cloud is being deployed.
	PolicyCloudAny PolicyCloud = "any"
)

// PolicyList holds x-defang-policies entries. In YAML it accepts a sequence
// or a single scalar, and entries may hold several comma-separated
// identifiers — so one `${VAR}` interpolated at compose-load time can carry
// a variable-length list. Empty entries (a "${VAR:-}" the stack leaves
// unset) are dropped.
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

// ClassifyPolicy determines the target cloud of an x-defang-policies entry
// from its qualified form: AWS ARNs, GCP role names, and Azure resource IDs
// are self-identifying; anything else is a bare name that resolves on the
// current cloud.
func ClassifyPolicy(entry string) PolicyCloud {
	switch {
	case strings.Contains(entry, PolicyScopeSeparator):
		// Only Azure takes a scope, so the suffix identifies the cloud on its
		// own — including for a bare role name that would otherwise be "any".
		return PolicyCloudAzure
	case strings.HasPrefix(entry, "arn:"):
		return PolicyCloudAWS
	case strings.HasPrefix(entry, "roles/"),
		strings.HasPrefix(entry, "projects/"),
		strings.HasPrefix(entry, "organizations/"):
		return PolicyCloudGCP
	case strings.HasPrefix(entry, "/"):
		// Azure role-definition resource IDs: /subscriptions/… or /providers/…
		return PolicyCloudAzure
	}
	return PolicyCloudAny
}

// ValidatePolicies rejects x-defang-policies entries that cannot apply on the
// given cloud. Entries must be literals by the time they reach the provider:
// compose variables are interpolated at compose-load time — by the CLI, not
// by the CD/providers — and `defang config` is deliberately not supported
// for policies, so an unresolved variable is an error, as is an identifier
// whose syntax belongs to a different cloud (there is no cross-cloud
// filtering; vary the variable's per-stack value instead).
func ValidatePolicies(cloud PolicyCloud, policies []string) error {
	for _, entry := range policies {
		if strings.Contains(entry, "${") {
			return fmt.Errorf("x-defang-policies entry %q has an unresolved variable: %w",
				entry, ErrPolicyUnresolvedVariable)
		}
		if c := ClassifyPolicy(entry); c != cloud && c != PolicyCloudAny {
			return fmt.Errorf("x-defang-policies entry %q is a %s identifier but this deployment targets %s: %w",
				entry, c, cloud, ErrPolicyForeignCloud)
		}
		if strings.Contains(entry, PolicyScopeSeparator) {
			// The scope keywords themselves are the provider's to know; only
			// the shape is checked here. Which scope a role may be granted at
			// is the cloud's own answer, reported when the grant is made.
			if role, scope := SplitPolicyScope(entry); role == "" || scope == "" {
				return fmt.Errorf("x-defang-policies entry %q: %w", entry, ErrPolicyMalformedScope)
			}
		}
	}
	return nil
}
