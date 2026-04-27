package azure

// Project is the top-level orchestration component for Azure. These tests verify
// that the Project component correctly wires up a set of services using the
// mock resource monitor. Detailed behaviour of each sub-component (Container
// App, Postgres, etc.) lives in their own dedicated test files.

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

func TestConstructAzureProject(t *testing.T) {
	server := testutil.MakeAzureTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AzureURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app":    testutil.ServiceWithPorts("nginx:latest", testutil.IngressPort(8080)),
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)
}

// TestConstructAzureProjectAllResourcesAreChildren asserts that every resource
// created inside a Project descends from the Project component in the Pulumi
// hierarchy. Runs a rich Construct that exercises shared infra (resource
// group, VNet, LAW, private DNS), Container Apps, managed Postgres, and
// Redis Enterprise so the assertion covers most resource-creation paths.
func TestConstructAzureProjectAllResourcesAreChildren(t *testing.T) {
	mock, tracker := testutil.NewParentTracker()
	server := testutil.MakeAzureTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AzureURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app": property.New(property.NewMap(map[string]property.Value{
				"image": property.New("nginx:latest"),
				"ports": property.New(property.NewArray([]property.Value{testutil.IngressPort(8080)})),
			})),
			"worker": testutil.ServiceWithImage("myapp:worker"),
			"db": property.New(property.NewMap(map[string]property.Value{
				"image":    property.New("postgres:17"),
				"postgres": property.New(property.NewMap(map[string]property.Value{})),
				"environment": property.New(property.NewMap(map[string]property.Value{
					"POSTGRES_PASSWORD": property.New("secret"),
				})),
			})),
			"cache": property.New(property.NewMap(map[string]property.Value{
				"image": property.New("redis:7"),
				"redis": property.New(property.NewMap(map[string]property.Value{})),
			})),
		}),
	})
	require.NoError(t, err)

	tracker.AssertAllDescendFrom(t, testutil.AzureURN("Project"))
}
