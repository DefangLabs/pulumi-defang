package defangaws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/DefangLabs/pulumi-defang/provider/shared"
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
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
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

	net, err := provideraws.ResolveNetworking(ctx, common.ToAWSConfig(inputs.AWS), childOpt)
	if err != nil {
		return nil, fmt.Errorf("resolving networking: %w", err)
	}

	sg, err := awsec2.NewSecurityGroup(ctx, name, &awsec2.SecurityGroupArgs{
		VpcId:       pulumi.StringOutput(net.VpcID),
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

	redisResult, err := provideraws.CreateElasticache(ctx, configProvider, name, svc, net.VpcID, net.PrivateSubnetIDs, sg, recipe, childOpt)
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
