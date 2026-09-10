package gcp

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ErrPolicyNotGCP rejects an x-defang-policies entry that cannot be a GCP
// role name.
var ErrPolicyNotGCP = errors.New(
	"a GCP role is `roles/…`, `projects/…/roles/…`, `organizations/…/roles/…`, " +
		"or the bare ID of a custom role in this project")

// ParsePolicies normalizes x-defang-policies for a GCP deployment and rejects
// what GCP cannot name. A qualified role has one of the three known prefixes;
// anything else is a bare custom-role ID, whose character set (letters,
// digits, "_" and ".") admits neither "/" nor ":" — so an entry carrying
// either is not a GCP identifier at all, most often another cloud's in a
// compose file deployed to several. Whether the role exists is IAM's answer,
// not this function's.
func ParsePolicies(entries []string) ([]string, error) {
	policies, err := compose.NormalizeLiteralPolicies(entries)
	if err != nil {
		return nil, err
	}
	for _, policy := range policies {
		if strings.HasPrefix(policy, "roles/") ||
			strings.HasPrefix(policy, "projects/") ||
			strings.HasPrefix(policy, "organizations/") {
			continue
		}
		if strings.ContainsAny(policy, "/:") {
			return nil, fmt.Errorf("x-defang-policies entry %q: %w%s",
				policy, ErrPolicyNotGCP, compose.PolicyVarHint)
		}
	}
	return policies, nil
}

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

// GrantPolicyRoles grants x-defang-policies roles to a provider-created
// service account at project level. Unlike AddRolesToServiceAccount (whose
// member names derive from the account email), members are named
// <service>-policy-<role> so a policy that repeats one of the platform-granted
// roles (e.g. the Compute Engine logging/monitoring set) cannot collide on the
// URN; the duplicate binding itself is idempotent on GCP. The created members
// are returned so the service's compute resources can DependsOn them — the
// container may need the granted permissions at startup.
func GrantPolicyRoles(
	ctx *pulumi.Context,
	serviceName string,
	sa *ServiceIdentity,
	roles []string,
	gcpConfig *SharedInfra,
	opts ...pulumi.ResourceOption,
) ([]pulumi.Resource, error) {
	members := make([]pulumi.Resource, 0, len(roles))
	for _, role := range roles {
		member, err := projects.NewIAMMember(ctx, serviceName+"-policy-"+role,
			&projects.IAMMemberArgs{
				Project: pulumi.String(gcpConfig.GcpProject),
				Role:    pulumi.String(role),
				Member:  pulumi.Sprintf("serviceAccount:%v", sa.Email),
			},
			append(opts, sa.deleteOpts()...)...,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, nil
}
