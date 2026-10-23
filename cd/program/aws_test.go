package program

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	awscompose "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// toAWSServiceArgs copies the compose service into the SDK args field by
// field, so a managed extension the copy forgets is dropped in silence: the
// provider sees no extension and deploys the service as an ordinary ECS task.
// That is how x-defang-s3 reached AWS as a MinIO container on its first
// end-to-end run, with the bucket never created.
func TestToAWSServiceArgsCarriesManagedExtensions(t *testing.T) {
	args := toAWSServiceArgs(compose.ServiceConfig{
		Image:       pulumi.String("minio/minio"),
		ObjectStore: &compose.ObjectStoreConfig{Bucket: "proj-uploads"},
	})

	store, ok := args.ObjectStore.(awscompose.ObjectStoreConfigArgs)
	require.True(t, ok, "x-defang-s3 did not survive the copy into ServiceConfigArgs")
	assert.Equal(t, pulumi.String("proj-uploads"), store.Bucket)
}

func TestToAWSServiceArgsWithoutObjectStore(t *testing.T) {
	args := toAWSServiceArgs(compose.ServiceConfig{Image: pulumi.String("nginx")})
	assert.Nil(t, args.ObjectStore)
}
