package compose

import "strings"

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

// PoliciesFor filters x-defang-policies entries to those that apply on the
// given cloud: entries qualified for another cloud are dropped, so a compose
// file targeting several clouds can list entries for each side by side; bare
// names apply everywhere. Returns the applicable entries followed by the
// skipped ones (for logging).
func PoliciesFor(cloud PolicyCloud, policies []string) ([]string, []string) {
	var applicable, skipped []string
	for _, p := range policies {
		if c := ClassifyPolicy(p); c == cloud || c == PolicyCloudAny {
			applicable = append(applicable, p)
		} else {
			skipped = append(skipped, p)
		}
	}
	return applicable, skipped
}

// SkippedPoliciesMessage formats the info-log line for x-defang-policies
// entries that don't apply on the current cloud.
func SkippedPoliciesMessage(serviceName string, skipped []string) string {
	return "service " + serviceName + ": ignoring x-defang-policies entries for other clouds: " +
		strings.Join(skipped, ", ")
}
