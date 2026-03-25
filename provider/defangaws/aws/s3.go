package aws

import (
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

//nolint:unused // ported from TypeScript, will be used when log buckets are enabled
func createBucket(
	ctx *pulumi.Context,
	name string,
	args *s3.BucketArgs,
	opts ...pulumi.ResourceOption,
) (*s3.Bucket, error) {
	bucket, err := s3.NewBucket(
		ctx,
		name,
		args,
		append([]pulumi.ResourceOption{pulumi.RetainOnDelete(RetainBucketOnDelete.Get(ctx))}, opts...)...,
	)
	if err != nil {
		return nil, err
	}

	if args.ServerSideEncryptionConfiguration == nil {
		_, err = s3.NewBucketServerSideEncryptionConfiguration(ctx, name, &s3.BucketServerSideEncryptionConfigurationArgs{
			Bucket: bucket.ID(),
			Rules: s3.BucketServerSideEncryptionConfigurationRuleArray{
				s3.BucketServerSideEncryptionConfigurationRuleArgs{
					BucketKeyEnabled: pulumi.Bool(BucketKeyEnabled.Get(ctx)), // minimize KMS costs in non-prod environments
				},
			},
		}, opts...)
		if err != nil {
			return nil, err
		}
	}

	return bucket, err
}

//nolint:unused // ported from TypeScript, will be used when log buckets are enabled
func createPrivateBucket(
	ctx *pulumi.Context,
	name string,
	args *s3.BucketArgs,
	opts ...pulumi.ResourceOption,
) (*s3.Bucket, error) {
	bucket, err := createBucket(ctx, name, args, opts...)
	if err != nil {
		return nil, err
	}

	_, err = s3.NewBucketPublicAccessBlock(
		ctx,
		name,
		&s3.BucketPublicAccessBlockArgs{
			Bucket:                bucket.ID(),
			BlockPublicAcls:       pulumi.Bool(true),
			BlockPublicPolicy:     pulumi.Bool(true),
			IgnorePublicAcls:      pulumi.Bool(true),
			RestrictPublicBuckets: pulumi.Bool(true),
		},
		opts...,
	)
	if err != nil {
		return nil, err
	}
	return bucket, err
}
