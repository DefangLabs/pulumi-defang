package aws

// Build is a custom resource (not a component) that triggers an AWS
// CodeBuild build and waits for completion. These tests exercise the Check path,
// which validates inputs without performing any real AWS API calls.

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

func TestCheckBuildMinimal(t *testing.T) {
	server := testutil.MakeTestServer()

	resp, err := server.Check(p.CheckRequest{
		Urn: testutil.AwsURN("Build"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("my-codebuild-project"),
		}),
	})

	require.NoError(t, err)
	assert.Empty(t, resp.Failures)
}

func TestCheckBuildComplete(t *testing.T) {
	server := testutil.MakeTestServer()

	resp, err := server.Check(p.CheckRequest{
		Urn: testutil.AwsURN("Build"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("my-codebuild-project"),
			"region":      property.New("us-east-1"),
			"destination": property.New("123456789012.dkr.ecr.us-east-1.amazonaws.com/myrepo:latest"),
			"maxWaitTime": property.New(float64(1800)),
			"triggers": property.New(property.NewArray([]property.Value{
				property.New("sha256:abc123"),
				property.New("sha256:def456"),
			})),
		}),
	})

	require.NoError(t, err)
	assert.Empty(t, resp.Failures)
}

func TestCheckBuildMissingProjectName(t *testing.T) {
	server := testutil.MakeTestServer()

	resp, err := server.Check(p.CheckRequest{
		Urn:    testutil.AwsURN("Build"),
		Inputs: property.NewMap(map[string]property.Value{}),
	})

	require.NoError(t, err)
	assert.NotEmpty(t, resp.Failures)
}
