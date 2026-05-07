package defangscaleway

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Service is the controller struct for the defang-scaleway:index:Service component.
type Service struct{}

// ScalewayServiceInputs defines the inputs for a standalone Scaleway Serverless Container.
type ScalewayServiceInputs struct {
	Image       string                      `pulumi:"image"`
	Platform    *string                     `pulumi:"platform,optional"`
	Ports       []compose.ServicePortConfig `pulumi:"ports,optional"`
	Deploy      *compose.DeployConfig       `pulumi:"deploy,optional"`
	Environment map[string]*string          `pulumi:"environment,optional"`
	Command     []string                    `pulumi:"command,optional"`
	Entrypoint  []string                    `pulumi:"entrypoint,optional"`
	HealthCheck *compose.HealthCheckConfig  `pulumi:"healthCheck,optional"`
	DomainName  string                      `pulumi:"domainName,optional"`
}

// ScalewayServiceOutputs holds the outputs of a Scaleway Service component.
type ScalewayServiceOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
}

// Construct implements the ComponentResource interface for Service.
func (*Service) Construct(
	ctx *pulumi.Context, name, typ string, inputs ScalewayServiceInputs, opts pulumi.ResourceOption,
) (*ScalewayServiceOutputs, error) {
	comp := &ScalewayServiceOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}
	if inputs.Image == "" {
		return nil, fmt.Errorf("service %s: %w", name, common.ErrStandaloneServiceRequiresImage)
	}
	return nil, fmt.Errorf("defang-scaleway Service runtime is not implemented yet")
}
