package defanggcp

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providergcp "github.com/DefangLabs/pulumi-defang/provider/defanggcp/gcp"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/redis"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Redis is the controller struct for the defang-gcp:index:Redis component.
type Redis struct{}

// RedisInputs defines the inputs for a standalone GCP Memorystore Redis instance.
type RedisInputs struct {
	Redis  *compose.RedisConfig        `pulumi:"redis,optional"`
	Image  *string                     `pulumi:"image,optional"`
	Deploy *compose.DeployConfig       `pulumi:"deploy,optional"`
	Ports  []compose.ServicePortConfig `pulumi:"ports,optional"`
}

// RedisOutputs holds the outputs of a Redis component.
type RedisOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
	// Instance is an internal-only handle to the Memorystore instance. The project
	// dispatcher reads it to build an LBServiceEntry. Untagged — not part of the
	// SDK schema.
	Instance *redis.Instance
}

// RedisComponentType is the Pulumi resource type token for the Redis component.
const RedisComponentType = "defang-gcp:index:Redis"

// Construct implements the ComponentResource interface for Redis.
func (*Redis) Construct(
	ctx *pulumi.Context, name, typ string, inputs RedisInputs, opts pulumi.ResourceOption,
) (*RedisOutputs, error) {
	comp := &RedisOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	svc := compose.ServiceConfig{
		Redis:  inputs.Redis,
		Image:  inputs.Image,
		Deploy: inputs.Deploy,
		Ports:  inputs.Ports,
	}

	// Standalone Construct runs without a shared GlobalConfig; the project-level
	// dispatcher calls createRedis with a non-nil infra.
	return comp, createRedis(ctx, comp, name, svc, nil)
}

// createRedis creates the Memorystore instance under an already-registered Redis
// component, populates its Endpoint/Instance, and registers its outputs. Shared
// between Construct and the project-level dispatcher.
func createRedis(
	ctx *pulumi.Context,
	comp *RedisOutputs,
	serviceName string,
	svc compose.ServiceConfig,
	infra *providergcp.SharedInfra,
) error {
	childOpt := pulumi.Parent(comp)

	result, err := providergcp.CreateMemoryStore(ctx, serviceName, svc, infra, childOpt)
	if err != nil {
		return fmt.Errorf("creating Memorystore for %s: %w", serviceName, err)
	}

	comp.Endpoint = pulumi.Sprintf("%s:%d", result.Instance.Host, firstPort(svc.Ports, defaultRedisPort))
	comp.Instance = result.Instance

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": comp.Endpoint,
	}); err != nil {
		return fmt.Errorf("registering outputs for %s: %w", serviceName, err)
	}
	return nil
}
