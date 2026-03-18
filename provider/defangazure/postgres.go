package defangazure

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/defangazure/azure"
	providerazure "github.com/DefangLabs/pulumi-defang/provider/defangazure/azure"
	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// AzurePostgres is the controller struct for the defang-azure:index:AzurePostgres component.
type AzurePostgres struct{}

// AzurePostgresInputs defines the inputs for a standalone Azure PostgreSQL Flexible Server.
type AzurePostgresInputs struct {
	ProjectName *string               `pulumi:"project_name"`
	Image       *string               `pulumi:"image,optional"`
	Postgres    *shared.PostgresInput `pulumi:"postgres,optional"`
	Deploy      *shared.DeployConfig  `pulumi:"deploy,optional"`
	Environment map[string]*string    `pulumi:"environment,optional"`
}

// AzurePostgresOutputs holds the outputs of an AzurePostgres component.
type AzurePostgresOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
}

// Construct implements the ComponentResource interface for AzurePostgres.
func (*AzurePostgres) Construct(ctx *pulumi.Context, name, typ string, inputs AzurePostgresInputs, opts pulumi.ResourceOption) (*AzurePostgresOutputs, error) {
	comp := &AzurePostgresOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	childOpt := pulumi.Parent(comp)
	svc := shared.ServiceInput{
		Image:       inputs.Image,
		Postgres:    inputs.Postgres,
		Deploy:      inputs.Deploy,
		Environment: inputs.Environment,
	}

	configProvider := azure.NewConfigProvider(*inputs.ProjectName)
	result, err := providerazure.BuildStandalonePostgres(ctx, configProvider, name, svc, childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build Azure PostgreSQL: %w", err)
	}

	comp.Endpoint = result.Endpoint

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": result.Endpoint,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}
