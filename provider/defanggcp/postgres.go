package defanggcp

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providergcp "github.com/DefangLabs/pulumi-defang/provider/defanggcp/gcp"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/sql"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Postgres is the controller struct for the defang-gcp:index:Postgres component.
type Postgres struct{}

// PostgresInputs defines the inputs for a standalone GCP Cloud SQL Postgres instance.
type PostgresInputs struct {
	ProjectName string                      `pulumi:"project_name"`
	Postgres    *compose.PostgresConfig     `pulumi:"postgres,optional"`
	Image       *string                     `pulumi:"image,optional"`
	Deploy      *compose.DeployConfig       `pulumi:"deploy,optional"`
	Environment map[string]*string          `pulumi:"environment,optional"`
	Ports       []compose.ServicePortConfig `pulumi:"ports,optional"`
}

// PostgresOutputs holds the outputs of a Postgres component.
type PostgresOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
	// Instance is an internal-only handle to the Cloud SQL instance. The project
	// dispatcher reads it to build an LBServiceEntry. Untagged — not part of the
	// SDK schema.
	Instance *sql.DatabaseInstance
}

// PostgresComponentType is the Pulumi resource type token for the Postgres component.
const PostgresComponentType = "defang-gcp:index:Postgres"

// Construct implements the ComponentResource interface for Postgres.
func (*Postgres) Construct(
	ctx *pulumi.Context, name, typ string, inputs PostgresInputs, opts pulumi.ResourceOption,
) (*PostgresOutputs, error) {
	comp := &PostgresOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	svc := compose.ServiceConfig{
		Postgres:    inputs.Postgres,
		Image:       inputs.Image,
		Deploy:      inputs.Deploy,
		Environment: inputs.Environment,
		Ports:       inputs.Ports,
	}

	var configProvider compose.ConfigProvider
	if ctx.DryRun() {
		configProvider = &compose.DryRunConfigProvider{}
	} else {
		configProvider = providergcp.NewConfigProvider(inputs.ProjectName)
	}
	// Standalone Construct runs without a shared GlobalConfig; the project-level
	// dispatcher calls createPostgres with a non-nil infra.
	return comp, createPostgres(ctx, comp, configProvider, name, svc, nil)
}

// createPostgres creates the Cloud SQL instance under an already-registered Postgres
// component, populates its Endpoint/Instance, and registers its outputs. Shared
// between Construct and the project-level dispatcher.
func createPostgres(
	ctx *pulumi.Context,
	comp *PostgresOutputs,
	configProvider compose.ConfigProvider,
	serviceName string,
	svc compose.ServiceConfig,
	infra *providergcp.SharedInfra,
) error {
	childOpt := pulumi.Parent(comp)

	sqlResult, err := providergcp.CreateCloudSQL(ctx, configProvider, serviceName, svc, infra, childOpt)
	if err != nil {
		return fmt.Errorf("creating Cloud SQL for %s: %w", serviceName, err)
	}

	port := firstPort(svc.Ports, defaultPostgresPort)
	comp.Endpoint = pulumi.Sprintf("%s:%d", sqlResult.Instance.PublicIpAddress, port)
	comp.Instance = sqlResult.Instance

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": comp.Endpoint,
	}); err != nil {
		return fmt.Errorf("registering outputs for %s: %w", serviceName, err)
	}
	return nil
}
