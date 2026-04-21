package azure

import (
	"fmt"

	"github.com/pulumi/pulumi-azure-native-sdk/authorization/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/managedidentity/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Azure built-in role definition ID for "Key Vault Secrets User" — not a credential.
//
//nolint:gosec // built-in role definition ID, not a secret
const keyVaultSecretsUserRoleID = "4633458b-17de-408a-b874-0445c86b69e6"

// CreateKeyVaultIdentity creates a user-assigned managed identity with
// Key Vault Secrets User role on the vault. Returns the identity resource ID
// (with an implicit dependency on the role assignment completing).
//
// The vault may live in a different resource group than the project (set via
// defang-azure:keyVaultResourceGroup); if unset, it's assumed to share the
// project RG.
func CreateKeyVaultIdentity(
	ctx *pulumi.Context,
	vaultName string,
	infra *SharedInfra,
	location string,
	opts ...pulumi.ResourceOption,
) (pulumi.StringOutput, error) {
	identity, err := managedidentity.NewUserAssignedIdentity(
		ctx, "kv", &managedidentity.UserAssignedIdentityArgs{
			ResourceGroupName: infra.ResourceGroup.Name,
			Location:          pulumi.String(location),
		}, opts...)
	if err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("creating KV managed identity: %w", err)
	}

	subID := SubscriptionID(ctx)
	roleDefID := fmt.Sprintf(
		"/subscriptions/%s/providers/Microsoft.Authorization/roleDefinitions/%s",
		subID, keyVaultSecretsUserRoleID,
	)

	var vaultScope pulumi.StringOutput
	if kvRG := KeyVaultResourceGroup(ctx); kvRG != "" {
		vaultScope = pulumi.String(fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.KeyVault/vaults/%s",
			subID, kvRG, vaultName,
		)).ToStringOutput()
	} else {
		vaultScope = infra.ResourceGroup.ID().ApplyT(func(rgID string) string {
			return rgID + "/providers/Microsoft.KeyVault/vaults/" + vaultName
		}).(pulumi.StringOutput)
	}

	// Parent defaults to the surrounding component (via opts). Pulumi's SDK
	// discourages using custom resources as parents — destruction order here is
	// already enforced by the implicit data dependency on identity.PrincipalId.
	roleAssignment, err := authorization.NewRoleAssignment(ctx, "kv-secrets-user", &authorization.RoleAssignmentArgs{
		Scope:            vaultScope,
		RoleDefinitionId: pulumi.String(roleDefID),
		PrincipalId:      identity.PrincipalId,
		PrincipalType:    pulumi.String("ServicePrincipal"),
	}, opts...)
	if err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("creating KV role assignment: %w", err)
	}

	// Return identity ID with implicit dependency on the role assignment.
	identityID := pulumi.All(identity.ID(), roleAssignment.ID()).ApplyT(
		func(args []interface{}) string {
			return string(args[0].(pulumi.ID))
		},
	).(pulumi.StringOutput)

	return identityID, nil
}
