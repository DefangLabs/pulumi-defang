package common

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

// AWSConfig holds optional AWS-specific configuration overrides.
type AWSConfig struct {
	VpcID            string
	SubnetIDs        []string
	PrivateSubnetIDs []string
	Region           string
}

// GCPConfig holds optional GCP-specific configuration overrides.
type GCPConfig struct {
	Project string
	Region  string
}

// BuildArgs are the inputs to a cloud provider's Build function.
type BuildArgs struct {
	Services map[string]ServiceConfig
	AWS      *AWSConfig
	GCP      *GCPConfig
}

// BuildResult holds the outputs of a cloud build.
type BuildResult struct {
	Endpoints       pulumi.StringMapOutput
	LoadBalancerDNS pulumi.StringPtrOutput
}
