package defangaws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Project is the controller struct for the defang-aws:index:Project component.
type Project struct{}

// ProjectInputs defines the top-level inputs for the AWS Project component.
type ProjectInputs struct {
	// Services map: name -> service config
	Services map[string]shared.ServiceInput `pulumi:"services" yaml:"services"`

	// AWS-specific infrastructure configuration (VPC, subnets)
	AWS *shared.AWSConfigInput `pulumi:"aws,optional"`
}

// ProjectOutputs holds the outputs of the Project component.
type ProjectOutputs struct {
	pulumi.ResourceState

	// Per-service endpoint URLs (service name -> URL)
	Endpoints pulumi.StringMapOutput `pulumi:"endpoints"`

	// Load balancer DNS name (AWS ALB)
	LoadBalancerDNS pulumi.StringPtrOutput `pulumi:"loadBalancerDns,optional"`
}

// Construct implements the ComponentResource interface for Project.
func (*Project) Construct(ctx *pulumi.Context, name, typ string, inputs ProjectInputs, opts pulumi.ResourceOption) (*ProjectOutputs, error) {
	comp := &ProjectOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	childOpt := pulumi.Parent(comp)
	args := common.BuildArgs{
		Services: inputs.Services,
	}

	result, err := provideraws.Build(ctx, name, args, common.ToAWSConfig(inputs.AWS), childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build AWS resources: %w", err)
	}

	comp.Endpoints = result.Endpoints
	comp.LoadBalancerDNS = result.LoadBalancerDNS

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoints":       result.Endpoints,
		"loadBalancerDns": result.LoadBalancerDNS,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}
