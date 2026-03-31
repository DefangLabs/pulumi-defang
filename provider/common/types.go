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
