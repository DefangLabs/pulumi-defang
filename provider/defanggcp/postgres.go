package defanggcp

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providergcp "github.com/DefangLabs/pulumi-defang/provider/defanggcp/gcp"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Postgres is the controller struct for the defang-gcp:index:Postgres component.
type Postgres struct{}

// GcpCloudSqlInputs defines the inputs for a standalone GCP Cloud SQL Postgres instance.
type GcpCloudSqlInputs struct {
	ProjectName string                 `pulumi:"project_name"`
	Postgres    *compose.PostgresConfig `pulumi:"postgres,optional"`
	Image       *string                `pulumi:"image,optional"`
	Deploy      *compose.DeployConfig  `pulumi:"deploy,optional"`
	Environment map[string]string      `pulumi:"environment,optional"`
}

// GcpCloudSqlOutputs holds the outputs of a Postgres component.
type GcpCloudSqlOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
}

// Construct implements the ComponentResource interface for Postgres.
func (*Postgres) Construct(
	ctx *pulumi.Context, name, typ string, inputs GcpCloudSqlInputs, opts pulumi.ResourceOption,
) (*GcpCloudSqlOutputs, error) {
	comp := &GcpCloudSqlOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	childOpt := pulumi.Parent(comp)
	svc := compose.ServiceConfig{
		Postgres:    inputs.Postgres,
		Image:       inputs.Image,
		Deploy:      inputs.Deploy,
		Environment: inputs.Environment,
	}

	configProvider := providergcp.NewConfigProvider(inputs.ProjectName)
	sqlResult, err := providergcp.CreateCloudSQL(ctx, configProvider, name, svc, childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build GCP Cloud SQL: %w", err)
	}

	endpoint := pulumi.Sprintf("%s:5432", sqlResult.Instance.PublicIpAddress)
	comp.Endpoint = endpoint

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": endpoint,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}
