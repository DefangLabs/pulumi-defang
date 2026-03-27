package aws

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

type AWSConfig struct {
	ProjectDomain string                `pulumi:"projectDomain,optional" yaml:"projectDomain,omitempty"`
	PublicZoneId  pulumi.StringPtrInput `pulumi:"publicZoneId,optional"  yaml:"publicZoneId,omitempty"`
}
