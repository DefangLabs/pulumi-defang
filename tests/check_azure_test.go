package tests

// Project is the top-level orchestration component for Azure. These tests verify
// that the Project component correctly wires up a set of services using the
// mock resource monitor. Detailed behaviour of each sub-component (Container
// App, Postgres, etc.) lives in their own dedicated test files.

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

func TestConstructAzureProject(t *testing.T) {
	server := makeAzureTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: azureURN("Project"),
		Inputs: servicesMap(map[string]property.Value{
			"app":    serviceWithPorts("nginx:latest", ingressPort(8080)),
			"worker": serviceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)
}
