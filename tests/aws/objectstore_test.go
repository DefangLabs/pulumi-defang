package aws

// ObjectStore is the standalone S3 bucket component for AWS. These tests verify
// that the ObjectStore component constructs successfully and surfaces non-empty
// outputs for a declared bucket name.

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

func TestConstructAwsObjectStore(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	resp, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("ObjectStore"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"objectStore": property.New(property.NewMap(map[string]property.Value{
				"bucket": property.New("myproject-uploads"),
			})),
			"aws": awsConfig,
		}),
	})
	require.NoError(t, err)

	// endpoint/region/arn are cloud-computed fields the mock resource monitor
	// doesn't synthesize (it only echoes back explicit inputs), so only their
	// presence is checked, same as the Project ALB-ARN outputs in
	// project_test.go. "bucket" is an explicit input (via BucketArgs.Bucket),
	// so the mock echoes it back and its value can be asserted directly.
	for _, key := range []string{"endpoint", "bucket", "region", "arn"} {
		_, ok := resp.State.GetOk(key)
		assert.True(t, ok, "missing output %q", key)
	}

	v, ok := resp.State.GetOk("bucket")
	require.True(t, ok, "missing output %q", "bucket")
	assert.Equal(t, "myproject-uploads", v.AsString())
	assert.NotEmpty(t, v.AsString())
}

func TestConstructAwsObjectStoreNoInfra(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("ObjectStore"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"objectStore": property.New(property.NewMap(map[string]property.Value{
				"bucket": property.New("myproject-uploads"),
			})),
		}),
	})
	require.NoError(t, err)
}
