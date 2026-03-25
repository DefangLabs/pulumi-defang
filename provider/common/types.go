package common

import (
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const DefangComment = "Managed by Defang"

// BuildArgs are the inputs to a cloud provider's Build function.
type BuildArgs struct {
	Services compose.Services
	Domain   string
}

// BuildResult holds the outputs of a cloud build.
type BuildResult struct {
	Endpoints       pulumi.StringMapOutput
	LoadBalancerDNS pulumi.StringPtrOutput
}

// MergeOptions is like TypeScripts `pulumi.mergeOptions`
func MergeOptions(opts []pulumi.ResourceOption, overrides ...pulumi.ResourceOption) []pulumi.ResourceOption {
	return append(append([]pulumi.ResourceOption{}, opts...), overrides...)
}
