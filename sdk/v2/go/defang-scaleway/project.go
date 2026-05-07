package defangscaleway

import (
	"errors"
	"reflect"

	"github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-scaleway/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Project struct {
	pulumi.ResourceState

	Endpoints       pulumi.StringMapOutput `pulumi:"endpoints"`
	LoadBalancerDns pulumi.StringPtrOutput `pulumi:"loadBalancerDns"`
}

type ProjectArgs struct {
	Services compose.ServiceConfigMapInput
	Networks compose.NetworkConfigMapInput
	Etag     *string
}

func (ProjectArgs) ElementType() reflect.Type {
	return reflect.TypeOf((*projectArgs)(nil)).Elem()
}

type projectArgs struct {
	Services compose.ServiceConfigMapInput `pulumi:"services"`
	Networks compose.NetworkConfigMapInput `pulumi:"networks"`
	Etag     pulumi.StringPtrInput         `pulumi:"etag"`
}

func (args ProjectArgs) ToProjectArgsOutput() ProjectArgsOutput {
	return pulumi.ToOutput(args).(ProjectArgsOutput)
}

type ProjectArgsOutput struct{ *pulumi.OutputState }

func (ProjectArgsOutput) ElementType() reflect.Type {
	return reflect.TypeOf((*projectArgs)(nil)).Elem()
}

func NewProject(ctx *pulumi.Context, name string, args *ProjectArgs, opts ...pulumi.ResourceOption) (*Project, error) {
	if args == nil {
		return nil, errors.New("missing one or more required arguments")
	}
	if args.Services == nil {
		return nil, errors.New("invalid value for required argument 'Services'")
	}
	var resource Project
	if err := ctx.RegisterRemoteComponentResource("defang-scaleway:index:Project", name, args, &resource, opts...); err != nil {
		return nil, err
	}
	return &resource, nil
}
