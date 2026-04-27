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
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

func TestConstructAwsProject(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app":    testutil.ServiceWithPorts("nginx:latest", testutil.IngressPort(8080)),
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)
}

// TestConstructAwsProjectAllResourcesAreChildren asserts that every resource
// created inside a Project descends from the Project component in the Pulumi
// hierarchy. Runs a rich Construct that exercises shared infra (VPC, ALB,
// DNS), container services, build-from-source, managed Postgres, and managed
// Redis so the assertion covers most resource-creation paths.
func TestConstructAwsProjectAllResourcesAreChildren(t *testing.T) {
	mock, tracker := testutil.NewParentTracker()
	server := testutil.MakeAwsTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Project"),
		Inputs: property.NewMap(map[string]property.Value{
			"domain": property.New("example.com"),
			"services": property.New(property.NewMap(map[string]property.Value{
				"app": property.New(property.NewMap(map[string]property.Value{
					"image": property.New("nginx:latest"),
					"ports": property.New(property.NewArray([]property.Value{testutil.IngressPort(8080)})),
				})),
				"worker": testutil.ServiceWithImage("myapp:worker"),
				"builder": property.New(property.NewMap(map[string]property.Value{
					"build": property.New(property.NewMap(map[string]property.Value{
						"context": property.New("./app"),
					})),
				})),
				"db": property.New(property.NewMap(map[string]property.Value{
					"image":    property.New("postgres:17"),
					"postgres": property.New(property.NewMap(map[string]property.Value{})),
				})),
				"cache": property.New(property.NewMap(map[string]property.Value{
					"image": property.New("redis:7"),
					"redis": property.New(property.NewMap(map[string]property.Value{})),
				})),
			})),
		}),
	})
	require.NoError(t, err)

	tracker.AssertAllDescendFrom(t, testutil.AwsURN("Project"))
}
