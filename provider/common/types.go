package common

import (
	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const DefangComment = "Managed by Defang"

// AWSConfig holds optional AWS infrastructure configuration (not provider auth).
type AWSConfig = shared.AWSConfigInput

// BuildArgs are the inputs to a cloud provider's Build function.
type BuildArgs struct {
	Services map[string]shared.ServiceInput
}

// BuildResult holds the outputs of a cloud build.
type BuildResult struct {
	Endpoints       pulumi.StringMapOutput
	LoadBalancerDNS pulumi.StringPtrOutput
}
