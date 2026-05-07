package defangscaleway

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Project is the controller struct for the defang-scaleway:index:Project component.
type Project struct{}

// ProjectInputs defines the top-level inputs for the Scaleway Project component.
type ProjectInputs struct {
	Services compose.Services `pulumi:"services"          yaml:"services"`
	Networks compose.Networks `pulumi:"networks,optional" yaml:"networks,omitempty"`
	Etag     string           `pulumi:"etag,optional"     yaml:"etag,omitempty"`
}

// ProjectOutputs holds the outputs of the Project component.
type ProjectOutputs struct {
	pulumi.ResourceState

	Endpoints       pulumi.StringMapOutput `pulumi:"endpoints"`
	LoadBalancerDNS pulumi.StringPtrOutput `pulumi:"loadBalancerDns,optional"`
}

// Construct implements the ComponentResource interface for Project.
func (*Project) Construct(
	ctx *pulumi.Context, name, typ string, inputs ProjectInputs, opts pulumi.ResourceOption,
) (*ProjectOutputs, error) {
	comp := &ProjectOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("defang-scaleway Project runtime is not implemented yet")
}
