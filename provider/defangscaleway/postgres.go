package defangscaleway

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Postgres is the controller struct for the defang-scaleway:index:Postgres component.
type Postgres struct{}

type ScalewayPostgresInputs struct {
	Postgres *compose.PostgresConfig `pulumi:"postgres,optional"`
}

type ScalewayPostgresOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
}

func (*Postgres) Construct(
	ctx *pulumi.Context, name, typ string, inputs ScalewayPostgresInputs, opts pulumi.ResourceOption,
) (*ScalewayPostgresOutputs, error) {
	comp := &ScalewayPostgresOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("defang-scaleway Postgres runtime is not implemented yet")
}
