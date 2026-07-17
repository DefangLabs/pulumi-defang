package gcp

import "strings"

// ResolvePolicyRole turns an x-defang-policies entry into an IAM role name for
// a project-level binding. Qualified names (`roles/…`, `projects/…/roles/…`,
// `organizations/…/roles/…`) pass through; a bare name resolves to a custom
// role in the given project (the GCP analogue of AWS resolvePolicyArn, which
// resolves bare names to a customer-managed policy in the caller's account).
func ResolvePolicyRole(gcpProject, policy string) string {
	if strings.Contains(policy, "/") {
		return policy
	}
	return "projects/" + gcpProject + "/roles/" + policy
}
