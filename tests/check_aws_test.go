package tests

// Project is the top-level orchestration component for AWS. These tests verify
// that the Project component correctly wires up a set of services using the
// mock resource monitor. Detailed behaviour of each sub-component (ECS service,
// Postgres, CodeBuildImageBuild, etc.) lives in their own dedicated test files.
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
)

func TestConstructAwsProject(t *testing.T) {
	server := makeTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: awsURN("Project"),
		Inputs: servicesMap(map[string]property.Value{
			"app":    serviceWithPorts("nginx:latest", ingressPort(8080)),
			"worker": serviceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)
}
