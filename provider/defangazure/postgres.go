package defangazure

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/DefangLabs/pulumi-defang/provider/defangazure/azure"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Postgres is the controller struct for the defang-azure:index:Postgres component.
type Postgres struct{}

// AzurePostgresInputs defines the inputs for a standalone Azure PostgreSQL Flexible Server.
type AzurePostgresInputs struct {
	ProjectName string                  `pulumi:"projectName"`
	Image       *string                 `pulumi:"image,optional"`
	Postgres    *compose.PostgresConfig `pulumi:"postgres,optional"`
	Deploy      *compose.DeployConfig   `pulumi:"deploy,optional"`
	Environment map[string]*string      `pulumi:"environment,optional"`
}

// AzurePostgresOutputs holds the outputs of an Postgres component.
type AzurePostgresOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
}

// Construct implements the ComponentResource interface for Postgres.
func (*Postgres) Construct(
	ctx *pulumi.Context, name, typ string, inputs AzurePostgresInputs, opts pulumi.ResourceOption,
) (*AzurePostgresOutputs, error) {
	comp := &AzurePostgresOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	childOpt := pulumi.Parent(comp)
	svc := compose.ServiceConfig{
		Image:       inputs.Image,
		Postgres:    inputs.Postgres,
		Deploy:      inputs.Deploy,
		Environment: inputs.Environment,
	}

	rg, err := resources.NewResourceGroup(ctx, name, &resources.ResourceGroupArgs{
		// Location: pulumi.String(location),
	}, childOpt)
	if err != nil {
		return nil, fmt.Errorf("creating resource group: %w", err)
	}

	infra := &azure.SharedInfra{ResourceGroup: rg}
	var configProvider compose.ConfigProvider
	if ctx.DryRun() {
		configProvider = &compose.DryRunConfigProvider{}
	} else {
		// Standalone Postgres has no Key Vault wiring, so secret refs are
		// unavailable and the lazy fetch is a no-op; pass an empty URL.
		configProvider = azure.NewConfigProvider("")
	}

	pgResult, err := azure.CreatePostgresFlexible(ctx, configProvider, name, svc, infra, childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build Azure PostgreSQL: %w", err)
	}

	comp.Endpoint = pulumi.Sprintf("%s:5432", pgResult.Server.FullyQualifiedDomainName)

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": comp.Endpoint,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}
