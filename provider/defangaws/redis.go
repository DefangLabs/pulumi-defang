package defangaws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	awsec2 "github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

// Redis is the controller struct for the defang-aws:index:Redis component.
type Redis struct{}

// RedisInputs defines the inputs for a standalone AWS ElastiCache Redis instance.
type RedisInputs struct {
	ProjectName string                      `pulumi:"project_name"`
	Redis       *compose.RedisInput         `pulumi:"redis,optional"`
	Image       *string                     `pulumi:"image,optional"`
	Ports       []compose.ServicePortConfig `pulumi:"ports,optional"`
	Deploy      *compose.DeployConfig       `pulumi:"deploy,optional"`
	Environment map[string]string           `pulumi:"environment,optional"`
	AWS         *compose.AWSConfigInput     `pulumi:"aws,optional"`
}

// RedisOutputs holds the outputs of an AWS Redis component.
type RedisOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
}

// Construct implements the ComponentResource interface for Redis.
func (*Redis) Construct(ctx *pulumi.Context, name, typ string, inputs RedisInputs, opts pulumi.ResourceOption) (*RedisOutputs, error) {
	comp := &RedisOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	childOpt := pulumi.Parent(comp)

	redis := inputs.Redis
	if redis == nil {
		redis = &compose.RedisInput{}
	}

	svc := compose.ServiceConfig{
		Redis:       redis,
		Image:       inputs.Image,
		Ports:       inputs.Ports,
		Deploy:      inputs.Deploy,
		Environment: inputs.Environment,
	}

	configProvider := provideraws.NewConfigProvider(inputs.ProjectName)

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
	redisResult, err := provideraws.CreateElasticache(ctx, configProvider, name, svc, pulumi.String(inputs.AWS.VpcID), privateSubnetIDs, sg, nil, childOpt)
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

	comp.Endpoint = endpoint

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": endpoint,
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
	configProvider compose.ConfigProvider,
	serviceName string,
	svc compose.ServiceConfig,
	infra *provideraws.SharedInfra,
	deps []pulumi.Resource,
	parentOpt pulumi.ResourceOption,
) (*RedisResult, error) {
	comp := &serviceComponent{}
	if err := ctx.RegisterComponentResource("defang-aws:index:Redis", serviceName, comp, parentOpt); err != nil {
		return nil, fmt.Errorf("registering redis component %s: %w", serviceName, err)
	}
	opts := []pulumi.ResourceOption{pulumi.Parent(comp)}

	redisResult, err := provideraws.CreateElasticache(ctx, configProvider, serviceName, svc, infra.VpcID, infra.PrivateSubnetIDs, infra.Sg, deps, opts...)
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
		Endpoint: endpoint,
	}, nil
}
