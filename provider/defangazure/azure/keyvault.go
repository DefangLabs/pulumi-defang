package azure

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pulumi/pulumi-azure-native-sdk/authorization/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/keyvault/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/managedidentity/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/v3/config"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Azure built-in role definition ID for "Key Vault Secrets User" — not a credential.
//
//nolint:gosec // built-in role definition ID, not a secret
const keyVaultSecretsUserRoleID = "4633458b-17de-408a-b874-0445c86b69e6"

var ErrNoKeyVault = errors.New("no Key Vault found")

// EnsureKeyVault adopts the CLI-created project Key Vault into Pulumi state
// with RetainOnDelete=true, so:
//
//   - the project tags are applied via the cascading transformation
//     (DefaultTagsTransformation in azure.go);
//   - `pulumi destroy` (defang compose down) does NOT issue a delete API call
//     against the vault, preserving the user's secrets across deploys.
//
// The vault itself is provisioned by the defang CLI (see
// defang/src/pkg/clouds/azure/keyvault.SetUp) before the CD task runs, so
// args here mirror the CLI's BeginCreateOrUpdate parameters exactly to avoid
// proposing a replacement on import.
//
// Returns (nil, ErrNoKeyVault) when the vault doesn't exist yet — the CLI may
// not have run SetUp (e.g. the user never ran `defang config set`). Callers
// should treat that as "no vault, no identity binding".
func EnsureKeyVault(
	ctx *pulumi.Context,
	composeProject string,
	infra *SharedInfra,
	parentOpt pulumi.ResourceOrInvokeOption,
) (*keyvault.Vault, error) {
	vaultName := KeyVaultName(ctx, composeProject)
	rgName := ProjectResourceGroupName(ctx, composeProject)

	existing, err := keyvault.LookupVault(ctx, &keyvault.LookupVaultArgs{
		ResourceGroupName: rgName,
		VaultName:         vaultName,
	}, parentOpt)
	if err != nil {
		if strings.Contains(err.Error(), `Code="ResourceNotFound"`) {
			return nil, ErrNoKeyVault
		}
		return nil, fmt.Errorf("looking up existing Key Vault: %w", err)
	}

	args := &keyvault.VaultArgs{
		Location:          pulumi.StringPtrFromPtr(existing.Location),
		ResourceGroupName: infra.ResourceGroup.Name,
		VaultName:         pulumi.String(vaultName),
		Properties: &keyvault.VaultPropertiesArgs{
			TenantId:                  pulumi.String(existing.Properties.TenantId),
			EnableRbacAuthorization:   pulumi.BoolPtr(true),
			EnableSoftDelete:          pulumi.BoolPtr(true),
			SoftDeleteRetentionInDays: pulumi.IntPtr(7),
			Sku: &keyvault.SkuArgs{
				Family: pulumi.String("A"),
				Name:   keyvault.SkuNameStandard,
			},
		},
	}

	vault, err := keyvault.NewVault(ctx, "config-vault", args,
		parentOpt,
		pulumi.Import(pulumi.ID(existing.Id)),
		pulumi.RetainOnDelete(true),
		// Don't fight the CLI / Azure on properties we don't actively manage.
		// Tags are intentionally NOT ignored — that's the whole point.
		pulumi.IgnoreChanges([]string{
			"properties.networkAcls",
			"properties.accessPolicies",
			"properties.publicNetworkAccess",
			"properties.enabledForDeployment",
			"properties.enabledForDiskEncryption",
			"properties.enabledForTemplateDeployment",
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("importing Key Vault: %w", err)
	}
	return vault, nil
}

// CreateKeyVaultIdentity creates a user-assigned managed identity with
// Key Vault Secrets User role on the vault. Returns the identity resource ID
// (with an implicit dependency on the role assignment completing).
//
// vault is the imported Pulumi Vault resource (from EnsureKeyVault); its ID
// output is the canonical Azure scope, which avoids any case mismatch between
// the locally-derived RG/vault name and what's stored in Azure.
func CreateKeyVaultIdentity(
	ctx *pulumi.Context,
	vault *keyvault.Vault,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (pulumi.StringOutput, error) {
	identity, err := managedidentity.NewUserAssignedIdentity(
		ctx, "kv", &managedidentity.UserAssignedIdentityArgs{
			ResourceGroupName: infra.ResourceGroup.Name,
			// Location:          pulumi.String(location),
		}, opts...)
	if err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("creating KV managed identity: %w", err)
	}

	subID := config.GetSubscriptionId(ctx)
	roleDefID := fmt.Sprintf(
		"/subscriptions/%s/providers/Microsoft.Authorization/roleDefinitions/%s",
		subID, keyVaultSecretsUserRoleID,
	)

	// Parent defaults to the surrounding component (via opts). Pulumi's SDK
	// discourages using custom resources as parents — destruction order here is
	// already enforced by the implicit data dependency on identity.PrincipalId.
	roleAssignment, err := authorization.NewRoleAssignment(ctx, "kv-secrets-user", &authorization.RoleAssignmentArgs{
		Scope:            vault.ID().ToStringOutput(),
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
