package provider

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	providergcp "github.com/DefangLabs/pulumi-defang/provider/gcp"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// GcpCloudSql is the controller struct for the defang:index:GcpCloudSql component.
type GcpCloudSql struct{}

// GcpCloudSqlInputs defines the inputs for a standalone GCP Cloud SQL Postgres instance.
type GcpCloudSqlInputs struct {
	Postgres    *PostgresInput    `pulumi:"postgres,optional"`
	Image       *string           `pulumi:"image,optional"`
	Deploy      *DeployConfig     `pulumi:"deploy,optional"`
	Environment map[string]string `pulumi:"environment,optional"`
	GCP         *GCPConfigInput   `pulumi:"gcp,optional"`
}

// GcpCloudSqlOutputs holds the outputs of a GcpCloudSql component.
type GcpCloudSqlOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
}

// Construct implements the ComponentResource interface for GcpCloudSql.
func (*GcpCloudSql) Construct(ctx *pulumi.Context, name, typ string, inputs GcpCloudSqlInputs, opts pulumi.ResourceOption) (*GcpCloudSqlOutputs, error) {
	comp := &GcpCloudSqlOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	childOpt := pulumi.Parent(comp)
	svc := common.ServiceConfig{
		Postgres:    toPostgres(inputs.Postgres, inputs.Image, inputs.Environment),
		Deploy:      toDeploy(inputs.Deploy),
		Environment: inputs.Environment,
	}

	result, err := providergcp.BuildStandaloneCloudSQL(ctx, name, svc, toGCPConfig(inputs.GCP), childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build GCP Cloud SQL: %w", err)
	}

	comp.Endpoint = result.Endpoint

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": result.Endpoint,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}
