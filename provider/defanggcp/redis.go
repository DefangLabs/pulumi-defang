package defanggcp

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providergcp "github.com/DefangLabs/pulumi-defang/provider/defanggcp/gcp"
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
}

// Construct implements the ComponentResource interface for Redis.
func (*Redis) Construct(
	ctx *pulumi.Context, name, typ string, inputs RedisInputs, opts pulumi.ResourceOption,
) (*RedisOutputs, error) {
	comp := &RedisOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	childOpt := pulumi.Parent(comp)
	svc := compose.ServiceConfig{
		Redis:  inputs.Redis,
		Image:  inputs.Image,
		Deploy: inputs.Deploy,
		Ports:  inputs.Ports,
	}

	result, err := providergcp.CreateMemoryStore(ctx, name, svc, nil, childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build GCP Memorystore: %w", err)
	}

	endpoint := pulumi.Sprintf("%s:%d", result.Instance.Host, firstPort(inputs.Ports, defaultRedisPort))
	comp.Endpoint = endpoint

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": endpoint,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}
