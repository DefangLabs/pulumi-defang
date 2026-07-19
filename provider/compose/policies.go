package compose

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	// ErrPolicyUnresolvedVariable rejects entries still containing `${…}`:
	// the CLI substitutes `.env` values at compose-load time, and `defang
	// config` is deliberately not supported for policies.
	ErrPolicyUnresolvedVariable = errors.New(
		"policy entries are substituted from `.env` when the compose file is loaded; " +
			"`defang config` is not supported for policies")
	// ErrPolicyForeignCloud rejects an identifier whose syntax belongs to a
	// different cloud: there is no cross-cloud filtering.
	ErrPolicyForeignCloud = errors.New(
		"use a ${VAR} entry with per-stack `.env` values to vary policies per cloud")
)

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
// identifiers — so one `${VAR}` substituted from `.env` can carry a
// variable-length list. Empty entries (a "${VAR:-}" the stack leaves unset)
// are dropped.
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
// the CLI substitutes `${VAR}` from `.env` at compose-load time, and `defang
// config` is deliberately not supported for policies — so an unresolved
// variable is an error, as is an identifier whose syntax belongs to a
// different cloud (there is no cross-cloud filtering; per-cloud values come
// from per-stack `.env` files).
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
	}
	return nil
}
