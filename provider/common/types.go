package common

import (
	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// AWSConfig holds optional AWS infrastructure configuration (not provider auth).
type AWSConfig struct {
	VpcID            string
	SubnetIDs        []string
	PrivateSubnetIDs []string
}

// ToAWSConfig converts Pulumi AWSConfigInput to AWSConfig.
func ToAWSConfig(a *shared.AWSConfigInput) *AWSConfig {
	if a == nil {
		return nil
	}
	return &AWSConfig{
		VpcID:            a.VpcID,
		SubnetIDs:        a.SubnetIDs,
		PrivateSubnetIDs: a.PrivateSubnetIDs,
	}
}

// BuildArgs are the inputs to a cloud provider's Build function.
type BuildArgs struct {
	Services map[string]shared.ServiceInput
}

// BuildResult holds the outputs of a cloud build.
type BuildResult struct {
	Endpoints       pulumi.StringMapOutput
	LoadBalancerDNS pulumi.StringPtrOutput
}
