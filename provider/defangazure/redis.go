package defangazure

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/DefangLabs/pulumi-defang/provider/defangazure/azure"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Redis is the controller struct for the defang-azure:index:Redis component.
type Redis struct{}

// AzureRedisInputs defines the inputs for a standalone Azure Cache for Redis instance.
type AzureRedisInputs struct {
	ProjectName string                `pulumi:"project_name"`
	Image       *string               `pulumi:"image,optional"`
	Redis       *compose.RedisConfig  `pulumi:"redis,optional"`
	Deploy      *compose.DeployConfig `pulumi:"deploy,optional"`
	Environment map[string]string     `pulumi:"environment,optional"`
}

// AzureRedisOutputs holds the outputs of an Azure Redis component.
type AzureRedisOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
}

// Construct implements the ComponentResource interface for Redis.
func (*Redis) Construct(
	ctx *pulumi.Context, name, typ string, inputs AzureRedisInputs, opts pulumi.ResourceOption,
) (*AzureRedisOutputs, error) {
	comp := &AzureRedisOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	childOpt := pulumi.Parent(comp)

	redis := inputs.Redis
	if redis == nil {
		redis = &compose.RedisConfig{}
	}

	svc := compose.ServiceConfig{
		Image:       inputs.Image,
		Redis:       redis,
		Deploy:      inputs.Deploy,
		Environment: inputs.Environment,
	}

	location := azure.Location(ctx)

	rg, err := resources.NewResourceGroup(ctx, name, &resources.ResourceGroupArgs{
		Location: pulumi.String(location),
	}, childOpt)
	if err != nil {
		return nil, fmt.Errorf("creating resource group: %w", err)
	}

	infra := &azure.SharedInfra{ResourceGroup: rg}

	redisResult, err := azure.CreateRedisEnterprise(ctx, name, svc, infra, childOpt)
	if err != nil {
		return nil, fmt.Errorf("creating Azure Managed Redis: %w", err)
	}

	comp.Endpoint = pulumi.Sprintf("%s:10000", redisResult.Cluster.HostName)

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": comp.Endpoint,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}
