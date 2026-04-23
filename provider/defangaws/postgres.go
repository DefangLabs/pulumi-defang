package defangaws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

// Postgres is the controller struct for the defang-aws:index:Postgres component.
type Postgres struct{}

// PostgresInputs defines the inputs for a standalone AWS RDS Postgres instance.
type PostgresInputs struct {
	ProjectName string                   `pulumi:"project_name"`
	Postgres    *compose.PostgresConfig  `pulumi:"postgres,optional"`
	Image       *string                  `pulumi:"image,optional"`
	Deploy      *compose.DeployConfig    `pulumi:"deploy,optional"`
	Environment map[string]*string       `pulumi:"environment,optional"`
	Infra       *provideraws.SharedInfra `pulumi:"aws,optional"`
}

// PostgresOutputs holds the outputs of an AWS Postgres component.
type PostgresOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
	// Dependency is an internal-only handle (CNAME record or RDS instance) used by
	// downstream services for ordering. Untagged — not part of the SDK schema.
	Dependency pulumi.Resource
}

// PostgresComponentType is the Pulumi resource type token for the Postgres component.
const PostgresComponentType = "defang-aws:index:Postgres"

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
	}

	var configProvider compose.ConfigProvider
	if ctx.DryRun() {
		configProvider = &compose.DryRunConfigProvider{}
	} else {
		configProvider = provideraws.NewConfigProvider(inputs.ProjectName)
	}
	if err := createPostgres(ctx, comp, configProvider, name, svc, inputs.Infra, nil); err != nil {
		return nil, err
	}
	return comp, nil
}

// createPostgres creates the RDS instance (plus optional private-zone CNAME) under
// an already-registered Postgres component, sets its Endpoint/Dependency, and
// registers its outputs. Shared between Construct and the project-level dispatcher.
func createPostgres(
	ctx *pulumi.Context,
	comp *PostgresOutputs,
	configProvider compose.ConfigProvider,
	serviceName string,
	svc compose.ServiceConfig,
	infra *provideraws.SharedInfra,
	deps []pulumi.Resource,
) error {
	childOpt := pulumi.Parent(comp)

	rdsResult, err := provideraws.CreateRDS(
		ctx, configProvider, serviceName, svc,
		infra.VpcID, infra.PrivateSubnetIDs, infra.PrivateSgID,
		deps, childOpt,
	)
	if err != nil {
		return fmt.Errorf("creating RDS for %s: %w", serviceName, err)
	}

	var dependency pulumi.Resource = rdsResult.Instance
	if infra.PrivateZoneID != nil {
		privateFqdn := common.SafeLabel(serviceName) //+ "." + infra.PrivateDomain
		record, cnameErr := provideraws.CreateRecord(ctx, privateFqdn, common.RecordTypeCNAME, &route53.RecordArgs{
			ZoneId:  infra.PrivateZoneID.ToStringPtrOutput().Elem(),
			Records: pulumi.StringArray{rdsResult.Instance.Address},
			Ttl:     pulumi.Int(300),
		}, childOpt)
		if cnameErr != nil {
			return fmt.Errorf("creating CNAME for %s: %w", serviceName, cnameErr)
		}
		dependency = record
	}
	comp.Dependency = dependency

	comp.Endpoint = pulumi.StringOutput(pulumix.Apply(
		pulumix.Output[string](rdsResult.Instance.Address), func(addr string) string {
			return fmt.Sprintf("%s:%d", addr, 5432)
		}))

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{"endpoint": comp.Endpoint}); err != nil {
		return fmt.Errorf("registering outputs for %s: %w", serviceName, err)
	}
	return nil
}
