package defangaws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	awssdk "github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

// Postgres is the controller struct for the defang-aws:index:Postgres component.
type Postgres struct{}

// PostgresInputs defines the inputs for a standalone AWS RDS Postgres instance.
type PostgresInputs struct {
	ProjectName string                  `pulumi:"project_name"`
	Postgres    *compose.PostgresInput  `pulumi:"postgres,optional"`
	Image       *string                 `pulumi:"image,optional"`
	Deploy      *compose.DeployConfig   `pulumi:"deploy,optional"`
	Environment map[string]string       `pulumi:"environment,optional"`
	AWS         *compose.AWSConfigInput `pulumi:"aws,optional"`
}

// PostgresOutputs holds the outputs of an AWS Postgres component.
type PostgresOutputs struct {
	pulumi.ResourceState
	Endpoint pulumix.Output[string] `pulumi:"endpoint"`
}

// Construct implements the ComponentResource interface for Postgres.
func (*Postgres) Construct(
	ctx *pulumi.Context, name, typ string, inputs PostgresInputs, opts pulumi.ResourceOption,
) (*PostgresOutputs, error) {
	comp := &PostgresOutputs{}
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

	configProvider := provideraws.NewConfigProvider(inputs.ProjectName)

	sg, err := awssdk.NewSecurityGroup(ctx, name, &awssdk.SecurityGroupArgs{
		VpcId:       pulumi.String(inputs.AWS.VpcID),
		Description: pulumi.String("Security group for Postgres"),
		Egress: awssdk.SecurityGroupEgressArray{
			&awssdk.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
	}, childOpt)
	if err != nil {
		return nil, fmt.Errorf("creating security group: %w", err)
	}

	privateSubnetIDs := make(pulumi.StringArray, len(inputs.AWS.PrivateSubnetIDs))
	for i, id := range inputs.AWS.PrivateSubnetIDs {
		privateSubnetIDs[i] = pulumi.String(id)
	}
	rdsResult, err := provideraws.CreateRDS(
		ctx, configProvider, name, svc, pulumi.String(inputs.AWS.VpcID), privateSubnetIDs, sg, nil, childOpt,
	)
	if err != nil {
		return nil, fmt.Errorf("creating RDS: %w", err)
	}

	comp.Endpoint = pulumix.Apply(pulumix.Output[string](rdsResult.Instance.Address), func(addr string) string {
		return fmt.Sprintf("%s:%d", addr, 5432)
	})

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": comp.Endpoint,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}

type PostgresResult struct {
	Endpoint   pulumi.StringOutput
	Dependency pulumi.Resource
}

// newPostgresComponent registers a component resource for a managed Postgres service,
// creates its RDS children, registers outputs, and returns the host:port endpoint.
func newPostgresComponent(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	serviceName string,
	svc compose.ServiceConfig,
	infra *provideraws.SharedInfra,
	deps []pulumi.Resource,
	parentOpt pulumi.ResourceOption,
) (*PostgresResult, error) {
	comp := &serviceComponent{}
	if err := ctx.RegisterComponentResource("defang-aws:index:Postgres", serviceName, comp, parentOpt); err != nil {
		return nil, fmt.Errorf("registering postgres component %s: %w", serviceName, err)
	}
	opts := []pulumi.ResourceOption{pulumi.Parent(comp)}

	rdsResult, err := provideraws.CreateRDS(
		ctx, configProvider, serviceName, svc, infra.VpcID, infra.PrivateSubnetIDs, infra.Sg, deps, opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("creating RDS for %s: %w", serviceName, err)
	}

	endpoint := pulumi.StringOutput(pulumix.Apply(
		pulumix.Output[string](rdsResult.Instance.Address), func(addr string) string {
		return fmt.Sprintf("%s:%d", addr, 5432)
	}))

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{"endpoint": endpoint}); err != nil {
		return nil, fmt.Errorf("registering outputs for %s: %w", serviceName, err)
	}
	return &PostgresResult{
		Endpoint:   endpoint,
		Dependency: rdsResult.Instance,
	}, nil
}
