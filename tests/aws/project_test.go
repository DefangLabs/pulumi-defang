package aws

// Project is the top-level orchestration component for AWS. These tests verify
// that the Project component correctly wires up a set of services using the
// mock resource monitor. Detailed behaviour of each sub-component (ECS service,
// Postgres, Build, etc.) lives in their own dedicated test files.
//
// Note: services that embed environment variables inside ECS container-
// definition JSON hit a mock-monitor limitation (StringOutputs cannot be
// marshaled to JSON). Those cases are covered in the per-service test files
// which can supply a richer mock.

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

func TestConstructAwsProject(t *testing.T) {
	server := testutil.MakeTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app":    testutil.ServiceWithPorts("nginx:latest", testutil.IngressPort(8080)),
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)
}
