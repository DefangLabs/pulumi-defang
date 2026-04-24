package main

import (
	"strings"

	defangazure "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-azure"
	azcompose "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-azure/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/authorization/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/keyvault/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v3"
	pulumiazurenative "github.com/pulumi/pulumi-azure-native-sdk/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func runAzure(ctx *pulumi.Context) error {
	location := config.New(ctx, "azure-native").Require("location")
	azureProvider, err := pulumiazurenative.NewProvider(ctx, "azure", &pulumiazurenative.ProviderArgs{
		Location: pulumi.String(location),
	})
	if err != nil {
		return err
	}

	// Resolve the caller's tenant + objectId so we can grant the identity
	// running `pulumi up` read/write on the vault we're about to create.
	clientConfig, err := authorization.GetClientConfig(ctx, pulumi.Provider(azureProvider))
	if err != nil {
		return err
	}

	// Give the RG a fixed name so defang-azure:keyVaultResourceGroup (set in
	// stack config) can reference it statically — the provider uses it to
	// build the role-assignment scope for the Container App's managed identity.
	defangAzureCfg := config.New(ctx, "defang-azure")
	rgName := defangAzureCfg.Require("keyVaultResourceGroup")
	rg, err := resources.NewResourceGroup(ctx, "config-example", &resources.ResourceGroupArgs{
		ResourceGroupName: pulumi.String(rgName),
		Location:          pulumi.String(location),
	}, pulumi.Provider(azureProvider))
	if err != nil {
		return err
	}

	// Key Vault names are globally unique and capped at 24 chars — use one from
	// stack config so different stacks don't collide. defang-azure's
	// FetchUserConfig reads this same value (defang-azure:keyVaultName) at
	// Construct time to find where to look for user config.
	vaultName := defangAzureCfg.Require("keyVaultName")

	// RBAC (not access policies), because defang-azure's CreateKeyVaultIdentity
	// grants vault access via a role assignment — access policies would ignore
	// that grant and the Container App's managed identity can't read secrets.
	vault, err := keyvault.NewVault(ctx, "config-example", &keyvault.VaultArgs{
		ResourceGroupName: rg.Name,
		VaultName:         pulumi.String(vaultName),
		Location:          pulumi.String(location),
		Properties: &keyvault.VaultPropertiesArgs{
			TenantId:                  pulumi.String(clientConfig.TenantId),
			EnableRbacAuthorization:   pulumi.Bool(true),
			EnableSoftDelete:          pulumi.Bool(true),
			SoftDeleteRetentionInDays: pulumi.Int(7),
			Sku: &keyvault.SkuArgs{
				Family: pulumi.String("A"),
				Name:   keyvault.SkuNameStandard,
			},
		},
	}, pulumi.Provider(azureProvider))
	if err != nil {
		return err
	}

	// With RBAC, the caller running `pulumi up` needs write perms on the vault
	// to create the secret below. "Key Vault Administrator" covers data plane
	// read+write. Built-in role ID 00482a5a-887f-4fb3-b363-3b7fe8e74483.
	vaultAdminRole, err := authorization.NewRoleAssignment(ctx, "kv-admin", &authorization.RoleAssignmentArgs{
		Scope:            vault.ID(),
		RoleDefinitionId: pulumi.Sprintf("/subscriptions/%s/providers/Microsoft.Authorization/roleDefinitions/00482a5a-887f-4fb3-b363-3b7fe8e74483", config.New(ctx, "azure-native").Require("subscriptionId")),
		PrincipalId:      pulumi.String(clientConfig.ObjectId),
		PrincipalType:    pulumi.String("User"),
	}, pulumi.Provider(azureProvider))
	if err != nil {
		return err
	}

	// Mirror the CLI's ToSecretName convention used by defang-azure's
	// GetSecretRef: "/" becomes "--" and "_" becomes "-".
	stackPath := "/Defang/" + ctx.Project() + "/" + ctx.Stack() + "/" + configKey
	secretName := strings.ReplaceAll(strings.ReplaceAll(stackPath, "/", "--"), "_", "-")
	secretName = strings.TrimPrefix(secretName, "--")

	// FetchUserConfig lists vault secrets and recovers the original key name
	// from the "original-key" tag — the secret name itself is ignored.
	// DependsOn the role assignment so secret creation waits for the caller's
	// write permission to be in place (otherwise ARM can 403 on propagation lag).
	secret, err := keyvault.NewSecret(ctx, "config", &keyvault.SecretArgs{
		ResourceGroupName: rg.Name,
		VaultName:         vault.Name,
		SecretName:        pulumi.String(secretName),
		Properties: &keyvault.SecretPropertiesArgs{
			Value: pulumi.String(configValue),
		},
		Tags: pulumi.StringMap{
			"original-key": pulumi.String(stackPath),
		},
	}, pulumi.Provider(azureProvider), pulumi.DependsOn([]pulumi.Resource{vaultAdminRole}))
	if err != nil {
		return err
	}

	proj, err := defangazure.NewProject(ctx, defangProjectName, &defangazure.ProjectArgs{
		Services: azcompose.ServiceConfigMap{
			"web": azcompose.ServiceConfigArgs{
				Image:   pulumi.StringPtr("busybox"),
				Command: testCommand,
				Ports: azcompose.ServicePortConfigArray{
					azcompose.ServicePortConfigArgs{
						Target: pulumi.Int(8080),
						Mode:   pulumi.StringPtr("ingress"),
					},
				},
				Environment: testEnvironment,
			},
		},
	}, pulumi.Provider(azureProvider), pulumi.DependsOn([]pulumi.Resource{secret}))
	if err != nil {
		return err
	}

	ctx.Export("azure-endpoints", proj.Endpoints)
	return nil
}
