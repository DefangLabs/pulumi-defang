package azure

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
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

// isAzureRoleDefinitionID reports whether policy is already a fully-qualified
// role-definition resource ID (as opposed to a bare role name to resolve).
// Mirrors compose.ClassifyPolicy's own Azure test ("/subscriptions/…" or
// "/providers/…"), so a value that reaches here already passed
// compose.ValidatePolicies against PolicyCloudAzure.
func isAzureRoleDefinitionID(policy string) bool {
	return strings.HasPrefix(policy, "/")
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

	filter := fmt.Sprintf("roleName eq '%s'", policy)
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
// lists only on Azure — the scope decides what they apply to — so the grant
// is made at the project's resource group, where every Defang-managed
// resource for the project lives (see DefangLabs/pulumi-defang#326).
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
	policies []string,
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

	scopeID := infra.ResourceGroup.ID().ToStringOutput()
	var lastAssignmentID pulumi.IDOutput
	for i, policy := range policies {
		roleDefID := scopeID.ApplyT(func(scope string) (string, error) {
			return resolveRoleDefinitionID(ctx.Context(), scope, policy)
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
