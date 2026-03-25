// Ported from https://github.com/DefangLabs/defang-mvp/blob/main/pulumi/shared/aws/lb.ts
package aws

import (
	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/aws/smithy-go/ptr"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/lb"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type LoadBalancerType string

const (
	ApplicationLoadBalancer LoadBalancerType = "application"
	NetworkLoadBalancer     LoadBalancerType = "network"
)

//nolint:unused // ported from TypeScript, will be used when access logging is enabled
func createLogsBucket(
	ctx *pulumi.Context,
	name string,
	typ LoadBalancerType,
	opts ...pulumi.InvokeOption,
) (*s3.Bucket, error) {
	lbAccountId, err := getCallerAccountId(ctx, opts...)
	if err != nil {
		return nil, err
	}

	var lbPrincipal any
	if typ == NetworkLoadBalancer {
		lbPrincipal = getNlbPrincipal()
	} else {
		lbRegion, err := getCallerRegion(ctx, opts...)
		if err != nil {
			return nil, err
		}
		lbPrincipal = getElbPrincipal(lbRegion)
	}

	bucket, err := createPrivateBucket(ctx, name, &s3.BucketArgs{
		ServerSideEncryptionConfiguration: s3.BucketServerSideEncryptionConfigurationTypeArgs{
			Rule: s3.BucketServerSideEncryptionConfigurationRuleArgs{
				//   applyServerSideEncryptionByDefault: { sseAlgorithm: "AES256" }, this is the default
				BucketKeyEnabled: pulumi.Bool(true), // frequently accessed objects will use bucket keys
			},
		},
	},
	// opts...,
	)
	if err != nil {
		return nil, err
	}

	// Expire logs after 30 days (from recipe)
	_, err = s3.NewBucketLifecycleConfiguration(ctx, name, &s3.BucketLifecycleConfigurationArgs{
		Bucket: bucket.ID(),
		Rules: s3.BucketLifecycleConfigurationRuleArray{
			s3.BucketLifecycleConfigurationRuleArgs{
				Status: pulumi.String("Enabled"),
				Expiration: &s3.BucketLifecycleConfigurationRuleExpirationArgs{
					Days: pulumi.Int(common.LogRetentionDays.Get(ctx)),
				},
			},
		},
	},
	// opts...,
	)
	if err != nil {
		return nil, err
	}

	// From AWS docs on access logging bucket requirements for NLB and ALB:
	// https://docs.aws.amazon.com/elasticloadbalancing/latest/network/load-balancer-access-logs.html
	// https://docs.aws.amazon.com/elasticloadbalancing/latest/application/load-balancer-access-logs.html
	policyJson := bucket.ID().ApplyT(func(bucketId string) pulumi.StringOutput {
		policy := iam.PolicyDocument{
			Version: "2012-10-17",
			Statement: []iam.PolicyStatement{
				{
					Sid:       ptr.String("AWSLogDeliveryWrite"),
					Effect:    "Allow",
					Principal: lbPrincipal,
					Action:    "s3:PutObject",
					// Value derived from an output https://www.pulumi.com/docs/intro/concepts/inputs-outputs/
					Resource: `arn:aws:s3:::` + bucketId + `/AWSLogs/` + lbAccountId + `/*`,
				},
				{
					Sid:       ptr.String("AWSLogDeliveryAclCheck"),
					Effect:    "Allow",
					Principal: lbPrincipal,
					Action:    "s3:GetBucketAcl",
					Resource:  `arn:aws:s3:::` + bucketId,
				},
			},
		}
		return pulumi.JSONMarshal(policy)
	})

	_, err = s3.NewBucketPolicy(
		ctx,
		name,
		&s3.BucketPolicyArgs{
			Bucket: bucket.ID(),
			Policy: policyJson,
		},
		// opts...,
	)
	if err != nil {
		return nil, err
	}

	return bucket, nil
}

// Ported from https://github.com/DefangLabs/defang-mvp/blob/main/pulumi/shared/aws/common.ts
//
//nolint:unused // ported from TypeScript, will be used when custom listeners are needed
func createListener(
	ctx *pulumi.Context,
	name string,
	args *lb.ListenerArgs,
	opts ...pulumi.ResourceOption,
) (*lb.Listener, error) {
	if args == nil {
		args = &lb.ListenerArgs{}
	}
	if args.Port == nil {
		if args.CertificateArn != nil {
			args.Port = pulumi.Int(443)
		} else {
			args.Port = pulumi.Int(80)
		}
	}
	if args.DefaultActions == nil {
		args.DefaultActions = lb.ListenerDefaultActionArray{
			lb.ListenerDefaultActionArgs{
				Type: pulumi.String("fixed-response"),
				FixedResponse: &lb.ListenerDefaultActionFixedResponseArgs{
					ContentType: pulumi.String("text/html"),
					MessageBody: pulumi.String(common.GetErrorHtml(
						"404 Not Found",
						"Resource Not Found",
						"The service you are looking for doesn't exist or is pending deployment. Check its status or domain name.",
					)),
					StatusCode: pulumi.String("404"),
				},
			},
		}
	}
	if args.Protocol == nil {
		if args.CertificateArn != nil {
			args.Protocol = pulumi.String("HTTPS")
		} else {
			args.Protocol = pulumi.String("HTTP")
		}
	}
	return lb.NewListener(ctx,
		name,
		args,
		common.MergeOptions(opts,
			// can't create a listener with the same port as an existing one
			pulumi.DeleteBeforeReplace(true),
			// certs remain in use even after update, so replace the listener
			pulumi.ReplaceOnChanges([]string{"certificateArn"}),
		)...,
	)
}
