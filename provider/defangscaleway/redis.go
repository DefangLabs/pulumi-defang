package defangscaleway

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Redis is the controller struct for the defang-scaleway:index:Redis component.
type Redis struct{}

type ScalewayRedisInputs struct {
	Redis *compose.RedisConfig `pulumi:"redis,optional"`
}

type ScalewayRedisOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
}

func (*Redis) Construct(
	ctx *pulumi.Context, name, typ string, inputs ScalewayRedisInputs, opts pulumi.ResourceOption,
) (*ScalewayRedisOutputs, error) {
	comp := &ScalewayRedisOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("defang-scaleway Redis runtime is not implemented yet")
}
