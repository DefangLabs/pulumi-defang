package azure

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/authorization/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/managedidentity/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// PolicyIdentity is the per-service user-assigned managed identity created
// for a service's x-defang-policies role assignments (see CreatePolicyIdentity).
type PolicyIdentity struct {
	// ID is the identity's ARM resource ID, for the Container App's
	// ManagedServiceIdentity.UserAssignedIdentities map.
	ID pulumi.StringOutput
	// ClientID is the identity's AAD application (client) ID. With more than
	// one user-assigned identity attached to a Container App,
	// DefaultAzureCredential cannot pick one on its own — the defang CLI
	// (which is what a self-redeploying service runs) needs AZURE_CLIENT_ID
	// in its environment to authenticate as this identity specifically.
	ClientID pulumi.StringOutput
}

// ErrPolicyNotAzure rejects an x-defang-policies entry that cannot be an
// Azure role identifier.
var ErrPolicyNotAzure = errors.New(
	"an Azure policy is a role name, or a full role-definition resource ID (/subscriptions/… or " +
		"/providers/…), each optionally suffixed with @SCOPE where SCOPE is `subscription` or a " +
		"full Azure scope resource ID")

// policyScopeSubscription grants at the whole subscription the deployment
// runs in, rather than at the project's own resource group. Needed by a
// service that manages resources outside its project: creating a resource
// group is a write at subscription scope, which a resource-group-scoped
// Contributor cannot do however broad the role is.
const policyScopeSubscription = "subscription"

// PolicyGrant is one parsed x-defang-policies entry: the role to grant, and
// the scope to grant it at. An empty Scope means the entry named none, which
// is the project's own resource group — see resolvePolicyScope.
type PolicyGrant struct {
	Role  string
	Scope string
}

// String reconstructs the compose entry this grant was parsed from, so an
// error names the value its author wrote rather than the struct.
func (p PolicyGrant) String() string {
	if p.Scope == "" {
		return p.Role
	}
	return p.Role + "@" + p.Scope
}

// ParsePolicies normalizes x-defang-policies for an Azure deployment and
// splits each entry into its role and scope halves.
//
// Azure is the only one of the clouds where a role carries no reach of its
// own — it is a capability list, and the scope decides what it applies to —
// so an entry may name one: "Contributor@subscription". The separator is cut
// at its first occurrence, so an Azure role whose display name contains "@"
// has to be named by its role-definition ID instead.
//
// A role that is not a resource ID is a name to look up, and a role name can
// hold most characters, so almost the only shape ruled out is one that looks
// like a path without being one — which is what another cloud's identifier
// looks like here.
func ParsePolicies(entries []string) ([]PolicyGrant, error) {
	policies, err := compose.NormalizeLiteralPolicies(entries)
	if err != nil {
		return nil, err
	}
	grants := make([]PolicyGrant, 0, len(policies))
	for _, policy := range policies {
		role, scope, scoped := strings.Cut(policy, "@")
		if role == "" || (scoped && scope == "") ||
			(strings.Contains(role, "/") && !isAzureRoleDefinitionID(role)) {
			return nil, fmt.Errorf("x-defang-policies entry %q: %w%s",
				policy, ErrPolicyNotAzure, compose.PolicyVarHint)
		}
		grants = append(grants, PolicyGrant{Role: role, Scope: scope})
	}
	return grants, nil
}

// isAzureRoleDefinitionID reports whether policy is already a fully-qualified
// role-definition resource ID (as opposed to a bare role name to resolve):
// "/subscriptions/…" or "/providers/…".
func isAzureRoleDefinitionID(policy string) bool {
	return strings.HasPrefix(policy, "/")
}

// roleNameFilter builds the ARM $filter value for looking up a role
// definition by name. OData escapes a single quote by doubling it; without
// this, a role name containing "'" either breaks the filter or (crafted as
// `foo' or roleName eq 'Owner`) widens the match to an unintended role.
func roleNameFilter(policy string) string {
	return fmt.Sprintf("roleName eq '%s'", strings.ReplaceAll(policy, "'", "''"))
}

// subscriptionScope reduces a resource group's own ARM ID
// (/subscriptions/<sub>/resourceGroups/<rg>) to the subscription scope above
// it. Reading the subscription off the group this deployment already owns
// makes it right by construction, where azure-native's stack config has to
// be set to be right (see readLiveCustomDomains, which has to cope with it
// being unset).
func subscriptionScope(resourceGroupID string) (string, error) {
	// A leading "/" makes parts[0] empty: "", "subscriptions", "<sub>", …
	parts := strings.Split(resourceGroupID, "/")
	if len(parts) < 3 || !strings.EqualFold(parts[1], "subscriptions") || parts[2] == "" {
		//nolint:err113 // reports the malformed ID itself, not a case a caller branches on
		return "", fmt.Errorf("no subscription in resource group ID %q", resourceGroupID)
	}
	return "/subscriptions/" + parts[2], nil
}

// resolvePolicyScope turns the scope half of an x-defang-policies entry into
// the ARM scope its role assignment is created at.
//
// An empty scope — an entry written without one, which is every entry
// predating this — stays the project's resource group, where all of a
// project's own Defang-managed resources live. `subscription` widens it to
// the whole subscription, for a service that manages resources outside its
// project: creating a resource group is a write at subscription scope, and no
// role granted on the project's group can authorize it. Anything else must
// already be a full ARM scope resource ID, which is how a specific group,
// resource or management group is named.
func resolvePolicyScope(resourceGroupID pulumi.StringOutput, scope string) (pulumi.StringOutput, error) {
	switch {
	case scope == "":
		return resourceGroupID, nil
	case scope == policyScopeSubscription:
		return resourceGroupID.ApplyT(subscriptionScope).(pulumi.StringOutput), nil
	case strings.HasPrefix(scope, "/"):
		return pulumi.String(scope).ToStringOutput(), nil
	}
	//nolint:err113 // the scope is caller-supplied compose data, not a fixed sentinel case
	return pulumi.StringOutput{}, fmt.Errorf(
		"unknown policy scope %q: use %q or a full Azure scope resource ID starting with %q",
		scope, policyScopeSubscription, "/")
}

// resolveRoleDefinitionID turns an x-defang-policies entry into a full
// role-definition resource ID at scope. A full resource ID passes through
// unchanged; a bare name (a built-in like "Contributor", or a custom role
// already visible at scope) is looked up by roleName via the ARM list API —
// the one mechanism that covers both role kinds, since built-ins have no
// per-subscription GUID a caller could reasonably hardcode. Mirrors
// resolvePolicyArn (AWS) / ResolvePolicyRole (GCP), except Azure has no
// invoke-style lookup for this (see LookupRoleDefinition, which needs the
// GUID already known) so this calls the raw ARM SDK directly — the same
// approach readLiveCustomDomains uses for a live, out-of-band Azure read.
func resolveRoleDefinitionID(ctx context.Context, scope, policy string) (string, error) {
	if isAzureRoleDefinitionID(policy) {
		return policy, nil
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return "", fmt.Errorf("building credential: %w", err)
	}
	client, err := armauthorization.NewRoleDefinitionsClient(cred, nil)
	if err != nil {
		return "", fmt.Errorf("building role definitions client: %w", err)
	}

	filter := roleNameFilter(policy)
	pager := client.NewListPager(scope, &armauthorization.RoleDefinitionsClientListOptions{Filter: &filter})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return "", fmt.Errorf("listing role definitions at %s: %w", scope, err)
		}
		for _, rd := range page.Value {
			if rd != nil && rd.ID != nil {
				return *rd.ID, nil
			}
		}
	}
	//nolint:err113 // the role name and scope are caller-supplied compose data, not a fixed sentinel case
	return "", fmt.Errorf("no role definition named %q found at %s", policy, scope)
}

// CreatePolicyIdentity creates a per-service user-assigned managed identity
// and grants it the x-defang-policies roles (already normalized and
// validated against PolicyCloudAzure by the caller). Roles are capability
// lists only on Azure — the scope decides what they apply to — so each entry
// may name its own scope (`Contributor@subscription`, see
// resolvePolicyScope), and one that doesn't is granted at the project's
// resource group, where every Defang-managed resource for the project lives
// (see DefangLabs/pulumi-defang#326).
//
// Returns (nil, nil) when policies is empty: no identity, no role
// assignments, nothing for the caller to attach — the common case, since
// most services don't use x-defang-policies.
//
// Azure gotcha (documented, not worked around here): a fresh role assignment
// can take up to ~5 minutes to propagate, so the very first deploy of a
// self-redeploying service may need a retry before its own redeploy trigger
// succeeds.
func CreatePolicyIdentity(
	ctx *pulumi.Context,
	serviceName string,
	policies []PolicyGrant,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*PolicyIdentity, error) {
	if len(policies) == 0 {
		return nil, nil //nolint:nilnil // no policies; the caller treats nil as "nothing to attach"
	}

	identity, err := managedidentity.NewUserAssignedIdentity(
		ctx, serviceName+"-policy", &managedidentity.UserAssignedIdentityArgs{
			ResourceGroupName: infra.ResourceGroup.Name,
		}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating policy managed identity: %w", err)
	}

	resourceGroupID := infra.ResourceGroup.ID().ToStringOutput()
	var lastAssignmentID pulumi.IDOutput
	for i, policy := range policies {
		scopeID, err := resolvePolicyScope(resourceGroupID, policy.Scope)
		if err != nil {
			return nil, fmt.Errorf("granting policy %q: %w", policy, err)
		}
		// The role is looked up at the scope it is granted at: a list there
		// returns both the built-ins (which every scope inherits) and any
		// custom role defined at or above it.
		roleDefID := scopeID.ApplyT(func(scope string) (string, error) {
			return resolveRoleDefinitionID(ctx.Context(), scope, policy.Role)
		}).(pulumi.StringOutput)

		assignment, err := authorization.NewRoleAssignment(
			ctx, fmt.Sprintf("%s-policy-%d", serviceName, i), &authorization.RoleAssignmentArgs{
				Scope:            scopeID,
				RoleDefinitionId: roleDefID,
				PrincipalId:      identity.PrincipalId,
				PrincipalType:    pulumi.String("ServicePrincipal"),
			}, opts...)
		if err != nil {
			return nil, fmt.Errorf("granting policy %q: %w", policy, err)
		}
		lastAssignmentID = assignment.ID()
	}

	// Depend both outputs on the last role assignment so a caller using the
	// identity (attaching it to a Container App, reading its client ID into
	// an env var) implicitly waits for every grant to be created first.
	id := pulumi.All(identity.ID(), lastAssignmentID).ApplyT(
		func(args []interface{}) string { return string(args[0].(pulumi.ID)) },
	).(pulumi.StringOutput)
	clientID := pulumi.All(identity.ClientId, lastAssignmentID).ApplyT(
		func(args []interface{}) string { return args[0].(string) },
	).(pulumi.StringOutput)

	return &PolicyIdentity{ID: id, ClientID: clientID}, nil
}
