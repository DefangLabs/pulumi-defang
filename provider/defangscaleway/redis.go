package defangscaleway

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providerscaleway "github.com/DefangLabs/pulumi-defang/provider/defangscaleway/scaleway"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/redis"
)

// Redis is the controller struct for the defang-scaleway:index:Redis component.
type Redis struct{}

// RedisInputs defines the inputs for a standalone Scaleway Redis cluster.
type ScalewayRedisInputs struct {
	Redis       *compose.RedisConfig          `pulumi:"redis,optional"`
	Image       *string                       `pulumi:"image,optional"`
	Ports       []compose.ServicePortConfig   `pulumi:"ports,optional"`
	Deploy      *compose.DeployConfig         `pulumi:"deploy,optional"`
	Environment map[string]*string            `pulumi:"environment,optional"`
	Scaleway    *providerscaleway.SharedInfra `pulumi:"scaleway,optional"`
}

// RedisOutputs holds the outputs of a Scaleway Redis component.
type ScalewayRedisOutputs struct {
	pulumi.ResourceState
	Endpoint      pulumi.StringOutput `pulumi:"endpoint"`
	ConnectionURL pulumi.StringOutput `pulumi:"connectionUrl"`
	Cluster       *redis.Cluster
	Dependency    pulumi.Resource
}

type RedisInputs = ScalewayRedisInputs
type RedisOutputs = ScalewayRedisOutputs

// RedisComponentType is the Pulumi resource type token for the Redis component.
const RedisComponentType = "defang-scaleway:index:Redis"

// Construct implements the ComponentResource interface for Redis.
func (*Redis) Construct(
	ctx *pulumi.Context, name, typ string, inputs ScalewayRedisInputs, opts pulumi.ResourceOption,
) (*ScalewayRedisOutputs, error) {
	comp := &ScalewayRedisOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	redisConfig := inputs.Redis
	if redisConfig == nil {
		redisConfig = &compose.RedisConfig{}
	}
	svc := compose.ServiceConfig{
		Redis:       redisConfig,
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

	if err := createRedis(ctx, comp, configProvider, name, svc, inputs.Scaleway); err != nil {
		return nil, err
	}
	return comp, nil
}

func createRedis(
	ctx *pulumi.Context,
	comp *ScalewayRedisOutputs,
	configProvider compose.ConfigProvider,
	serviceName string,
	svc compose.ServiceConfig,
	infra *providerscaleway.SharedInfra,
) error {
	childOpt := pulumi.Parent(comp)

	result, err := providerscaleway.CreateRedis(ctx, configProvider, serviceName, svc, infra, childOpt)
	if err != nil {
		return fmt.Errorf("creating Scaleway Redis for %s: %w", serviceName, err)
	}

	comp.Endpoint = result.Cluster.ConnectionString
	comp.ConnectionURL = result.ConnectionURL
	comp.Cluster = result.Cluster
	comp.Dependency = result.Cluster

	// Store the managed Redis host and connection URL for container env rewriting.
	if infra != nil && infra.ManagedHosts != nil {
		infra.ManagedHosts[serviceName] = result.Cluster.ConnectionString.ApplyT(
			providerscaleway.RedisAddressFromConnectionString,
		).(pulumi.StringOutput)
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
