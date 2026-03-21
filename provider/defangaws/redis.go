package defangaws

import (
	"fmt"

	provideraws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	awsec2 "github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

// AwsRedis is the controller struct for the defang-aws:index:AwsRedis component.
type AwsRedis struct{}

// AwsRedisInputs defines the inputs for a standalone AWS ElastiCache Redis instance.
type AwsRedisInputs struct {
	ProjectName *string                    `pulumi:"project_name"`
	Redis       *shared.RedisInput         `pulumi:"redis,optional"`
	Image       *string                    `pulumi:"image,optional"`
	Ports       []shared.ServicePortConfig `pulumi:"ports,optional"`
	Deploy      *shared.DeployConfig       `pulumi:"deploy,optional"`
	Environment map[string]*string         `pulumi:"environment,optional"`
	AWS         *shared.AWSConfigInput     `pulumi:"aws,optional"`
}

// AwsRedisOutputs holds the outputs of an AwsRedis component.
type AwsRedisOutputs struct {
	pulumi.ResourceState
	Endpoint   pulumi.StringOutput `pulumi:"endpoint"`
	Dependency pulumi.Resource     // typically CNAME record, for dependees
}

// Construct implements the ComponentResource interface for AwsRedis.
func (*AwsRedis) Construct(ctx *pulumi.Context, name, typ string, inputs AwsRedisInputs, opts pulumi.ResourceOption) (*AwsRedisOutputs, error) {
	comp := &AwsRedisOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	childOpt := pulumi.Parent(comp)

	redis := inputs.Redis
	if redis == nil {
		redis = &shared.RedisInput{}
	}

	svc := shared.ServiceInput{
		Redis:       redis,
		Image:       inputs.Image,
		Ports:       inputs.Ports,
		Deploy:      inputs.Deploy,
		Environment: inputs.Environment,
	}

	configProvider := provideraws.NewConfigProvider(*inputs.ProjectName)
	recipe := provideraws.LoadRecipe(ctx)

	sg, err := awsec2.NewSecurityGroup(ctx, name, &awsec2.SecurityGroupArgs{
		VpcId:       pulumi.String(inputs.AWS.VpcID),
		Description: pulumi.String("Security group for Redis"),
		Egress: awsec2.SecurityGroupEgressArray{
			&awsec2.SecurityGroupEgressArgs{
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
	redisResult, err := provideraws.CreateElasticache(ctx, configProvider, name, svc, pulumi.String(inputs.AWS.VpcID), privateSubnetIDs, sg, nil, "", recipe, nil, childOpt)
	if err != nil {
		return nil, fmt.Errorf("creating ElastiCache: %w", err)
	}

	port := 6379
	if len(svc.Ports) > 0 {
		port = svc.Ports[0].Target
	}
	endpoint := pulumi.StringOutput(pulumix.Apply(redisResult.Address, func(addr string) string {
		return fmt.Sprintf("%s:%d", addr, port)
	}))

	comp.Dependency = redisResult.Record
	comp.Endpoint = endpoint

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"dependency": redisResult.Record,
		"endpoint":   endpoint,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}

type RedisResult struct {
	Endpoint   pulumi.StringOutput
	Dependency pulumi.Resource
}

// newRedisComponent registers a component resource for a managed Redis service,
// creates its ElastiCache children, registers outputs, and returns the host:port endpoint.
func newRedisComponent(
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
) (*RedisResult, error) {
	comp := &serviceComponent{}
	if err := ctx.RegisterComponentResource("defang-aws:index:AwsRedis", serviceName, comp, parentOpt); err != nil {
		return nil, fmt.Errorf("registering redis component %s: %w", serviceName, err)
	}
	opts := []pulumi.ResourceOption{pulumi.Parent(comp)}

	redisResult, err := provideraws.CreateElasticache(ctx, configProvider, serviceName, svc, vpcID, privateSubnetIDs, sg, privateZoneId, privateFqdn, recipe, deps, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Redis for %s: %w", serviceName, err)
	}

	port := 6379
	if len(svc.Ports) > 0 {
		port = svc.Ports[0].Target
	}
	endpoint := pulumi.StringOutput(pulumix.Apply(redisResult.Address, func(addr string) string {
		return fmt.Sprintf("%s:%d", addr, port)
	}))

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{"endpoint": endpoint}); err != nil {
		return nil, fmt.Errorf("registering outputs for %s: %w", serviceName, err)
	}
	return &RedisResult{
		Endpoint:   endpoint,
		Dependency: redisResult.Record,
	}, nil
}
