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

// InvokeOptions returns the subset of opts that are also invoke options.
//
// Provider-package options such as pulumi.Parent and pulumi.Provider are
// declared as pulumi.ResourceOrInvokeOption, so a slice collected as
// []pulumi.ResourceOption still carries everything an invoke needs to resolve
// its provider. Invokes inherit the provider from their parent, and the CD
// runs with pulumi:disable-default-providers, so an invoke issued without one
// of these is rejected outright.
func InvokeOptions(opts []pulumi.ResourceOption) []pulumi.InvokeOption {
	var invokeOpts []pulumi.InvokeOption
	for _, opt := range opts {
		if invokeOpt, ok := opt.(pulumi.InvokeOption); ok {
			invokeOpts = append(invokeOpts, invokeOpt)
		}
	}
	return invokeOpts
}
