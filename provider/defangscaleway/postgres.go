package defangscaleway

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providerscaleway "github.com/DefangLabs/pulumi-defang/provider/defangscaleway/scaleway"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/databases"
)

// Postgres is the controller struct for the defang-scaleway:index:Postgres component.
type Postgres struct{}

// PostgresInputs defines the inputs for a standalone Scaleway PostgreSQL instance.
type ScalewayPostgresInputs struct {
	ProjectName string                        `pulumi:"projectName"`
	Postgres    *compose.PostgresConfig       `pulumi:"postgres,optional"`
	Image       *string                       `pulumi:"image,optional"`
	Ports       []compose.ServicePortConfig   `pulumi:"ports,optional"`
	Deploy      *compose.DeployConfig         `pulumi:"deploy,optional"`
	Environment map[string]*string            `pulumi:"environment,optional"`
	Scaleway    *providerscaleway.SharedInfra `pulumi:"scaleway,optional"`
}

// PostgresOutputs holds the outputs of a Scaleway Postgres component.
type ScalewayPostgresOutputs struct {
	pulumi.ResourceState
	Endpoint      pulumi.StringOutput `pulumi:"endpoint"`
	ConnectionURL pulumi.StringOutput `pulumi:"connectionUrl"`
	Instance      *databases.Instance
	Dependency    pulumi.Resource
}

type PostgresInputs = ScalewayPostgresInputs
type PostgresOutputs = ScalewayPostgresOutputs

// PostgresComponentType is the Pulumi resource type token for the Postgres component.
const PostgresComponentType = "defang-scaleway:index:Postgres"

// Construct implements the ComponentResource interface for Postgres.
func (*Postgres) Construct(
	ctx *pulumi.Context, name, typ string, inputs ScalewayPostgresInputs, opts pulumi.ResourceOption,
) (*ScalewayPostgresOutputs, error) {
	comp := &ScalewayPostgresOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	svc := compose.ServiceConfig{
		Postgres:    inputs.Postgres,
		Image:       inputs.Image,
		Ports:       inputs.Ports,
		Deploy:      inputs.Deploy,
		Environment: inputs.Environment,
	}

	configProvider := compose.ConfigProvider(&compose.PulumiConfigProvider{})
	if ctx.DryRun() {
		configProvider = &compose.DryRunConfigProvider{PlaceholderFormat: "DryRun1!-%s"}
	}
	if inputs.Scaleway != nil && inputs.Scaleway.ConfigProvider != nil {
		configProvider = inputs.Scaleway.ConfigProvider
	}

	if err := createPostgres(ctx, comp, configProvider, name, svc, inputs.Scaleway); err != nil {
		return nil, err
	}
	return comp, nil
}

func createPostgres(
	ctx *pulumi.Context,
	comp *ScalewayPostgresOutputs,
	configProvider compose.ConfigProvider,
	serviceName string,
	svc compose.ServiceConfig,
	infra *providerscaleway.SharedInfra,
) error {
	childOpt := pulumi.Parent(comp)

	result, err := providerscaleway.CreatePostgres(ctx, configProvider, serviceName, svc, infra, childOpt)
	if err != nil {
		return fmt.Errorf("creating Scaleway PostgreSQL for %s: %w", serviceName, err)
	}

	comp.Endpoint = pulumi.Sprintf("%s:%d", result.Host, result.Port)
	comp.ConnectionURL = result.ConnectionURL
	comp.Instance = result.Instance
	comp.Dependency = result.Privilege

	// Store the managed Postgres host so container services can rewrite
	// env values like POSTGRES_HOST=database to the actual hostname.
	if infra != nil && infra.ManagedHosts != nil {
		infra.ManagedHosts[serviceName] = result.Host
	}
	if infra != nil && infra.ManagedConnectionURLs != nil {
		infra.ManagedConnectionURLs[serviceName] = result.ConnectionURL
	}

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint":      comp.Endpoint,
		"connectionUrl": comp.ConnectionURL,
	}); err != nil {
		return fmt.Errorf("registering outputs for %s: %w", serviceName, err)
	}
	return nil
}
