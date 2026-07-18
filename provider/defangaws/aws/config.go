package aws

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

type AWSConfig struct {
	ProjectDomain string                `pulumi:"projectDomain,optional" yaml:"projectDomain,omitempty"`
	PublicZoneId  pulumi.StringPtrInput `pulumi:"publicZoneId,optional"  yaml:"publicZoneId,omitempty"`

	// DnsRoleArn is an IAM role to assume for public Route53 operations (zone
	// lookup and record management) when the DNS zone lives in a different
	// account. ACM certificates and listener attachments stay in the service
	// account.
	DnsRoleArn pulumi.StringInput `pulumi:"dnsRoleArn,optional" yaml:"dnsRoleArn,omitempty"`

	// AlarmTopicArn is a pre-existing SNS topic attached as the alarm/OK
	// action of every provider-created database alarm (created when the
	// alarms recipe is enabled). Unset means alarms are still created
	// (console-visible) but don't notify.
	AlarmTopicArn pulumi.StringInput `pulumi:"alarmTopicArn,optional" yaml:"alarmTopicArn,omitempty"`

	// AlbCertificateArn is the default certificate for the ALB's HTTPS
	// listener when the project doesn't manage its own domain (i.e. no
	// projectDomain/publicZoneId). Services can still bring their own domain
	// via domainname, which attaches additional certificates to the listener.
	AlbCertificateArn pulumi.StringInput `pulumi:"albCertificateArn,optional" yaml:"albCertificateArn,omitempty"`

	// VpcID adopts an existing VPC instead of creating one. PublicSubnetIDs is
	// required when set. PrivateSubnetIDs must have outbound connectivity (NAT
	// or equivalent) and are used for services without public ingress; when
	// omitted, all services run in the public subnets with public IPs. The
	// project still creates its own private hosted zone, attached to the
	// adopted VPC; the VPC itself (DHCP options, routing) is left untouched.
	VpcID            pulumi.StringInput      `pulumi:"vpcID,optional"            yaml:"vpcID,omitempty"`
	PublicSubnetIDs  pulumi.StringArrayInput `pulumi:"publicSubnetIDs,optional"  yaml:"publicSubnetIDs,omitempty"`
	PrivateSubnetIDs pulumi.StringArrayInput `pulumi:"privateSubnetIDs,optional" yaml:"privateSubnetIDs,omitempty"`
}
