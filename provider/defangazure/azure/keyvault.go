package azure

import (
	"fmt"
	"strings"

	"github.com/pulumi/pulumi-azure-native-sdk/authorization/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/managedidentity/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Azure built-in role definition ID for "Key Vault Secrets User" — not a credential.
//
//nolint:gosec // built-in role definition ID, not a secret
const keyVaultSecretsUserRoleID = "4633458b-17de-408a-b874-0445c86b69e6"

// ToContainerAppSecretName converts an env var name to a Container App secret
// name (lowercase, hyphens instead of underscores).
func ToContainerAppSecretName(envKey string) string {
	return strings.ToLower(strings.ReplaceAll(envKey, "_", "-"))
}

// KeyVaultSecretURL returns the Key Vault URL for a specific secret.
func KeyVaultSecretURL(vaultURL, project, stack, envKey string) string {
	// Mirror the CLI's ToSecretName convention:
	// "/{prefix}/{project}/{stack}/{KEY}" with / -> -- and _ -> -
	secretName := "Defang--" + project + "--" + stack + "--" + strings.ReplaceAll(envKey, "_", "-")
	return vaultURL + "/secrets/" + secretName
}

// CreateKeyVaultIdentity creates a user-assigned managed identity with
// Key Vault Secrets User role on the vault. Returns the identity resource ID
// (with an implicit dependency on the role assignment completing).
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
	vaultScope := infra.ResourceGroup.ID().ApplyT(func(rgID string) string {
		return rgID + "/providers/Microsoft.KeyVault/vaults/" + vaultName
	}).(pulumi.StringOutput)

	roleAssignment, err := authorization.NewRoleAssignment(ctx, "kv-secrets-user", &authorization.RoleAssignmentArgs{
		Scope:            vaultScope,
		RoleDefinitionId: pulumi.String(roleDefID),
		PrincipalId:      identity.PrincipalId,
		PrincipalType:    pulumi.String("ServicePrincipal"),
	}, append(opts, pulumi.Parent(identity))...)
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
