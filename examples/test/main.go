package main

import (
	"reflect"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ProjectArgs are the inputs for the defang:index:Project component.
type ProjectArgs struct {
	Provider string                  `pulumi:"provider"`
	Services map[string]ServiceInput `pulumi:"services"`
}

type ServiceInput struct {
	Image       *string           `pulumi:"image,optional"`
	Ports       []PortConfig      `pulumi:"ports,optional"`
	Deploy      *DeployConfig     `pulumi:"deploy,optional"`
	Environment map[string]string `pulumi:"environment,optional"`
}

type PortConfig struct {
	Target      int    `pulumi:"target"`
	Mode        string `pulumi:"mode,optional"`
	Protocol    string `pulumi:"protocol,optional"`
	AppProtocol string `pulumi:"appProtocol,optional"`
}

type DeployConfig struct {
	Replicas  *int             `pulumi:"replicas,optional"`
	Resources *ResourcesConfig `pulumi:"resources,optional"`
}

type ResourcesConfig struct {
	Reservations *ResourceConfig `pulumi:"reservations,optional"`
	Limits       *ResourceConfig `pulumi:"limits,optional"`
}

type ResourceConfig struct {
	CPUs   *float64 `pulumi:"cpus,optional"`
	Memory *string  `pulumi:"memory,optional"`
}

type projectArgs ProjectArgs

func (a ProjectArgs) ElementType() reflect.Type {
	return reflect.TypeOf((*projectArgs)(nil)).Elem()
}

// Project is the output resource.
type Project struct {
	pulumi.ResourceState

	Endpoints       pulumi.StringMapOutput `pulumi:"endpoints"`
	LoadBalancerDns pulumi.StringOutput    `pulumi:"loadBalancerDns"`
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		image := "nginx:latest"
		replicas := 1
		cpus := 0.25
		memory := "512m"

		proj := &Project{}
		err := ctx.RegisterRemoteComponentResource("defang:index:Project", "myProject", &ProjectArgs{
			Provider: "aws",
			Services: map[string]ServiceInput{
				"web": {
					Image: &image,
					Ports: []PortConfig{
						{Target: 80, Mode: "ingress", AppProtocol: "http"},
					},
					Deploy: &DeployConfig{
						Replicas:  &replicas,
						Resources: &ResourcesConfig{Reservations: &ResourceConfig{CPUs: &cpus, Memory: &memory}},
					},
					Environment: map[string]string{
						"NODE_ENV": "production",
					},
				},
			},
		}, proj)
		if err != nil {
			return err
		}

		ctx.Export("endpoints", proj.Endpoints)
		ctx.Export("loadBalancerDns", proj.LoadBalancerDns)
		return nil
	})
}
