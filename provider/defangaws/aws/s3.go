package aws

import (
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func createBucket(
	ctx *pulumi.Context,
	name string,
	args *s3.BucketArgs,
	sseRules s3.BucketServerSideEncryptionConfigurationRuleArrayInput,
	opts ...pulumi.ResourceOption,
) (*s3.Bucket, error) {
	// TS `IS_PROD` sites were split into semantic recipes: force-destroy-bucket,
	// bucket-key-enabled, retain-bucket-on-delete — each toggled independently.
	if args.ForceDestroy == nil {
		args.ForceDestroy = pulumi.Bool(ForceDestroyBucket.Get(ctx))
	}
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
		if sseRules == nil {
			sseRules = s3.BucketServerSideEncryptionConfigurationRuleArray{
				s3.BucketServerSideEncryptionConfigurationRuleArgs{
					ApplyServerSideEncryptionByDefault: &s3.
						BucketServerSideEncryptionConfigurationRuleApplyServerSideEncryptionByDefaultArgs{
						SseAlgorithm: pulumi.String("AES256"),
					},
					BucketKeyEnabled: pulumi.Bool(BucketKeyEnabled.Get(ctx)), // minimize KMS costs in non-prod environments
				},
			}
		}
		_, err = s3.NewBucketServerSideEncryptionConfiguration(ctx, name, &s3.BucketServerSideEncryptionConfigurationArgs{
			Bucket: bucket.ID(),
			Rules:  sseRules,
		}, opts...)
		if err != nil {
			return nil, err
		}
	}

	return bucket, err
}

func createPrivateBucket(
	ctx *pulumi.Context,
	name string,
	args *s3.BucketArgs,
	sseRules s3.BucketServerSideEncryptionConfigurationRuleArrayInput,
	opts ...pulumi.ResourceOption,
) (*s3.Bucket, error) {
	bucket, err := createBucket(ctx, name, args, sseRules, opts...)
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
