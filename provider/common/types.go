package common

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const DefangComment = "Managed by Defang"

// BuildResult holds the outputs of a cloud build.
type BuildResult struct {
	Endpoints       pulumi.StringMapOutput
	LoadBalancerDNS pulumi.StringPtrOutput
}

// MergeOptions is like TypeScripts `pulumi.mergeOptions`
func MergeOptions(opts []pulumi.ResourceOption, overrides ...pulumi.ResourceOption) []pulumi.ResourceOption {
	return append(append([]pulumi.ResourceOption{}, opts...), overrides...)
}

// Go has no slice covariance, so an option list that is valid for both
// resources and invokes cannot be handed to either API directly. These two
// widen such a list; every element is kept, nothing is inspected or dropped.
//
// The pair exists because a function that both looks a value up and creates
// resources from it — CreateRDS, CreateCloudSQL, CreatePostgresFlexible —
// takes pulumi.ResourceOrInvokeOption and needs each half separately. The
// invoke half matters: the CD runs with pulumi:disable-default-providers, so
// an invoke that carries neither a parent nor a provider is rejected.

func ResourceOptions(opts []pulumi.ResourceOrInvokeOption) []pulumi.ResourceOption {
	resourceOpts := make([]pulumi.ResourceOption, len(opts))
	for i, opt := range opts {
		resourceOpts[i] = opt
	}
	return resourceOpts
}

func InvokeOptions(opts []pulumi.ResourceOrInvokeOption) []pulumi.InvokeOption {
	invokeOpts := make([]pulumi.InvokeOption, len(opts))
	for i, opt := range opts {
		invokeOpts[i] = opt
	}
	return invokeOpts
}
