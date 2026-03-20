package defangaws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// AwsRedis is the controller struct for the defang-aws:index:AwsRedis component.
type AwsRedis struct{}

// AwsRedisInputs defines the inputs for a standalone AWS ElastiCache Redis instance.
type AwsRedisInputs struct {
	ProjectName *string               `pulumi:"project_name"`
	Redis       *shared.RedisInput    `pulumi:"redis,optional"`
	Image       *string               `pulumi:"image,optional"`
	Ports       []shared.PortConfig   `pulumi:"ports,optional"`
	Deploy      *shared.DeployConfig  `pulumi:"deploy,optional"`
	Environment map[string]*string    `pulumi:"environment,optional"`
	AWS         *shared.AWSConfigInput `pulumi:"aws,optional"`
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

	configProvider := aws.NewConfigProvider(*inputs.ProjectName)
	result, err := provideraws.BuildStandaloneRedis(ctx, configProvider, name, svc, common.ToAWSConfig(inputs.AWS), childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build AWS Redis: %w", err)
	}

	comp.Endpoint = result.Endpoint

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": result.Endpoint,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}
