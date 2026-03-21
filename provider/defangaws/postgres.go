package defangaws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	awssdk "github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

// AwsPostgres is the controller struct for the defang-aws:index:AwsPostgres component.
type AwsPostgres struct{}

// AwsPostgresInputs defines the inputs for a standalone AWS RDS Postgres instance.
type AwsPostgresInputs struct {
	ProjectName *string                `pulumi:"project_name"`
	Postgres    *shared.PostgresInput  `pulumi:"postgres,optional"`
	Image       *string                `pulumi:"image,optional"`
	Deploy      *shared.DeployConfig   `pulumi:"deploy,optional"`
	Environment map[string]*string     `pulumi:"environment,optional"`
	AWS         *shared.AWSConfigInput `pulumi:"aws,optional"`
}

// AwsPostgresOutputs holds the outputs of an AwsPostgres component.
type AwsPostgresOutputs struct {
	pulumi.ResourceState
	Endpoint   pulumix.Output[string] `pulumi:"endpoint"`
	Dependency pulumi.Resource        // typically CNAME record, for dependees
}

// Construct implements the ComponentResource interface for AwsPostgres.
func (*AwsPostgres) Construct(ctx *pulumi.Context, name, typ string, inputs AwsPostgresInputs, opts pulumi.ResourceOption) (*AwsPostgresOutputs, error) {
	comp := &AwsPostgresOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	childOpt := pulumi.Parent(comp)
	svc := shared.ServiceInput{
		Postgres:    inputs.Postgres,
		Image:       inputs.Image,
		Deploy:      inputs.Deploy,
		Environment: inputs.Environment,
	}

	configProvider := aws.NewConfigProvider(*inputs.ProjectName)
	recipe := aws.LoadRecipe(ctx)

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
	rdsResult, err := provideraws.CreateRDS(ctx, configProvider, name, svc, pulumi.String(inputs.AWS.VpcID), privateSubnetIDs, sg, pulumi.ID(inputs.AWS.PrivateZoneID), "", recipe, nil, childOpt)
	if err != nil {
		return nil, fmt.Errorf("creating RDS: %w", err)
	}

	comp.Dependency = rdsResult.Record
	comp.Endpoint = pulumix.Apply(pulumix.Output[string](rdsResult.Instance.Address), func(addr string) string {
		return fmt.Sprintf("%s:%d", addr, 5432)
	})

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"dependency": rdsResult.Record,
		"endpoint":   comp.Endpoint,
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
	configProvider shared.ConfigProvider,
	serviceName string,
	svc shared.ServiceInput,
	vpcID pulumi.StringInput,
	privateSubnetIDs pulumi.StringArrayInput,
	sg *ec2.SecurityGroup,
	privateZoneId pulumi.IDInput,
	privateFqdn string,
	recipe provideraws.Recipe,
	deps []pulumi.Resource,
	parentOpt pulumi.ResourceOption,
) (*PostgresResult, error) {
	comp := &serviceComponent{}
	if err := ctx.RegisterComponentResource("defang-aws:index:AwsPostgres", serviceName, comp, parentOpt); err != nil {
		return nil, fmt.Errorf("registering postgres component %s: %w", serviceName, err)
	}
	opts := []pulumi.ResourceOption{pulumi.Parent(comp)}

	rdsResult, err := provideraws.CreateRDS(ctx, configProvider, serviceName, svc, vpcID, privateSubnetIDs, sg, privateZoneId, privateFqdn, recipe, deps, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating RDS for %s: %w", serviceName, err)
	}

	endpoint := pulumi.StringOutput(pulumix.Apply(pulumix.Output[string](rdsResult.Instance.Address), func(addr string) string {
		return fmt.Sprintf("%s:%d", addr, 5432)
	}))

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{"endpoint": endpoint}); err != nil {
		return nil, fmt.Errorf("registering outputs for %s: %w", serviceName, err)
	}
	return &PostgresResult{
		Endpoint:   endpoint,
		Dependency: rdsResult.Record,
	}, nil
}
