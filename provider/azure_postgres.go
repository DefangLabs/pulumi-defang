package provider

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	providerazure "github.com/DefangLabs/pulumi-defang/provider/azure"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// AzurePostgres is the controller struct for the defang:index:AzurePostgres component.
type AzurePostgres struct{}

// AzurePostgresInputs defines the inputs for a standalone Azure PostgreSQL Flexible Server.
type AzurePostgresInputs struct {
	Image       *string            `pulumi:"image,optional"`
	Postgres    *PostgresInput     `pulumi:"postgres,optional"`
	Deploy      *DeployConfig      `pulumi:"deploy,optional"`
	Environment map[string]string  `pulumi:"environment,optional"`
	Azure       *AzureConfigInput  `pulumi:"azure,optional"`
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
	svc := common.ServiceConfig{
		Image:       inputs.Image,
		Deploy:      toDeploy(inputs.Deploy),
		Environment: inputs.Environment,
		Postgres:    toPostgres(inputs.Postgres, inputs.Image, inputs.Environment),
	}

	result, err := providerazure.BuildStandalonePostgres(ctx, name, svc, toAzureConfig(inputs.Azure), childOpt)
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
