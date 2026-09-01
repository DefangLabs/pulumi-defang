package defangaws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ObjectStore is the controller struct for the defang-aws:index:ObjectStore component.
type ObjectStore struct{}

// ObjectStoreInputs defines the inputs for a standalone AWS S3 bucket.
type ObjectStoreInputs struct {
	ProjectName string                     `pulumi:"projectName"`
	ObjectStore *compose.ObjectStoreConfig `pulumi:"objectStore"`
	Image       *string                    `pulumi:"image,optional"`
	Deploy      *compose.DeployConfig      `pulumi:"deploy,optional"`
	Environment compose.Environment        `pulumi:"environment,optional"`
	Infra       *provideraws.SharedInfra   `pulumi:"aws,optional"`
}

// ObjectStoreOutputs holds the outputs of an AWS ObjectStore component.
type ObjectStoreOutputs struct {
	pulumi.ResourceState
	// Endpoint is the bucket's virtual-hosted-style regional endpoint, e.g.
	// https://<bucket>.s3.<region>.amazonaws.com.
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
	// Bucket is the bucket name.
	Bucket pulumi.StringOutput `pulumi:"bucket"`
	Region pulumi.StringOutput `pulumi:"region"`
	// Arn is the bucket ARN — not consumed yet, but needed by a later
	// IAM-wiring PR that attaches a policy granting access to it.
	Arn pulumi.StringOutput `pulumi:"arn"`
	// Dependency is an internal-only handle (the bucket resource) used by
	// downstream services for ordering. Untagged — not part of the SDK schema.
	Dependency pulumi.Resource
}

// ObjectStoreComponentType is the Pulumi resource type token for the ObjectStore component.
const ObjectStoreComponentType = "defang-aws:index:ObjectStore"

// Construct implements the ComponentResource interface for ObjectStore.
func (*ObjectStore) Construct(
	ctx *pulumi.Context, name, typ string, inputs ObjectStoreInputs, opts pulumi.ResourceOption,
) (*ObjectStoreOutputs, error) {
	comp := &ObjectStoreOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	svc := compose.ServiceConfig{
		ObjectStore: inputs.ObjectStore,
		Image:       compose.ImageFromPtr(inputs.Image),
		Deploy:      inputs.Deploy,
		Environment: inputs.Environment,
	}

	if err := createObjectStore(ctx, comp, name, svc, inputs.Infra, nil); err != nil {
		return nil, err
	}
	return comp, nil
}

// createObjectStore creates a private S3 bucket under an already-registered
// ObjectStore component, sets its Endpoint/Bucket/Region/Arn/Dependency, and
// registers its outputs. Shared between Construct and the project-level dispatcher.
//
//nolint:unparam // infra is unused today (S3 buckets aren't VPC-scoped) but is
// kept for signature parity with createPostgres/createRedis: the project
// dispatcher calls all three workers uniformly, and a later IAM-wiring PR is
// expected to need infra (e.g. task role attachment).
func createObjectStore(
	ctx *pulumi.Context,
	comp *ObjectStoreOutputs,
	serviceName string,
	svc compose.ServiceConfig,
	infra *provideraws.SharedInfra,
	deps []pulumi.Resource,
) error {
	childOpt := pulumi.Parent(comp)

	opts := []pulumi.ResourceOption{childOpt}
	if len(deps) > 0 {
		opts = append(opts, pulumi.DependsOn(deps))
	}

	bucket, err := provideraws.CreatePrivateBucket(ctx, serviceName, &s3.BucketArgs{
		Bucket: pulumi.String(svc.ObjectStore.Bucket),
	}, nil, opts...)
	if err != nil {
		return fmt.Errorf("creating bucket for %s: %w", serviceName, err)
	}

	comp.Dependency = bucket
	comp.Bucket = bucket.Bucket
	comp.Region = bucket.Region
	comp.Arn = bucket.Arn
	comp.Endpoint = pulumi.Sprintf("https://%s", bucket.BucketRegionalDomainName)

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": comp.Endpoint,
		"bucket":   comp.Bucket,
		"region":   comp.Region,
		"arn":      comp.Arn,
	}); err != nil {
		return fmt.Errorf("registering outputs for %s: %w", serviceName, err)
	}
	return nil
}
