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

	vault, err := keyvault.NewVault(ctx, "config-example", &keyvault.VaultArgs{
		ResourceGroupName: rg.Name,
		VaultName:         pulumi.String(vaultName),
		Location:          pulumi.String(location),
		Properties: &keyvault.VaultPropertiesArgs{
			TenantId: pulumi.String(clientConfig.TenantId),
			Sku: &keyvault.SkuArgs{
				Family: pulumi.String("A"),
				Name:   keyvault.SkuNameStandard,
			},
			// Access policies are simpler than RBAC here: no role assignment
			// required, just list the object IDs and the operations allowed.
			AccessPolicies: keyvault.AccessPolicyEntryArray{
				&keyvault.AccessPolicyEntryArgs{
					TenantId: pulumi.String(clientConfig.TenantId),
					ObjectId: pulumi.String(clientConfig.ObjectId),
					Permissions: &keyvault.PermissionsArgs{
						Secrets: pulumi.StringArray{
							pulumi.String("get"),
							pulumi.String("list"),
							pulumi.String("set"),
							pulumi.String("delete"),
						},
					},
				},
			},
		},
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
	}, pulumi.Provider(azureProvider))
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
