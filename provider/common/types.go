package common

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

// AWSConfig holds optional AWS infrastructure configuration (not provider auth).
type AWSConfig struct {
	VpcID            string
	SubnetIDs        []string
	PrivateSubnetIDs []string
}

// BuildArgs are the inputs to a cloud provider's Build function.
type BuildArgs struct {
	Services map[string]ServiceConfig
}

// BuildResult holds the outputs of a cloud build.
type BuildResult struct {
	Endpoints       pulumi.StringMapOutput
	LoadBalancerDNS pulumi.StringPtrOutput
}
