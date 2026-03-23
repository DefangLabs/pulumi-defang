package tests

// AzureContainerApp is the standalone Container App component for Azure. These tests
// verify that the AzureContainerApp component correctly handles a variety of input
// configurations: image, ports, build config, health check, environment vars.

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

func TestConstructAzureContainerAppWithImage(t *testing.T) {
	server := makeAzureTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: azureURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("nginx:latest"),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAzureContainerAppWithIngressPort(t *testing.T) {
	server := makeAzureTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: azureURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("myapp:latest"),
			"ports": property.New(property.NewArray([]property.Value{
				ingressPort(8080),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAzureContainerAppWithBuild(t *testing.T) {
	t.Skip("build support for Azure standalone Service not yet implemented")
	server := makeAzureTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: azureURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"build": property.New(property.NewMap(map[string]property.Value{
				"context": property.New("./app"),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAzureContainerAppWithHealthCheck(t *testing.T) {
	server := makeAzureTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: azureURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("myapp:latest"),
			"ports": property.New(property.NewArray([]property.Value{
				ingressPort(8080),
			})),
			"healthCheck": property.New(property.NewMap(map[string]property.Value{
				"test": property.New(property.NewArray([]property.Value{
					property.New("CMD"),
					property.New("curl"),
					property.New("-f"),
					property.New("http://localhost:8080/health"),
				})),
				"intervalSeconds": property.New(float64(30)),
				"timeoutSeconds":  property.New(float64(5)),
				"retries":         property.New(float64(3)),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAzureContainerAppWithEnvironment(t *testing.T) {
	server := makeAzureTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: azureURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("myapp:latest"),
			"environment": property.New(property.NewMap(map[string]property.Value{
				"APP_ENV": property.New("production"),
				"PORT":    property.New("8080"),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAzureContainerAppWithDeploy(t *testing.T) {
	server := makeAzureTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: azureURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("myapp:latest"),
			"deploy": property.New(property.NewMap(map[string]property.Value{
				"replicas": property.New(float64(2)),
				"resources": property.New(property.NewMap(map[string]property.Value{
					"reservations": property.New(property.NewMap(map[string]property.Value{
						"cpus":   property.New(0.5),
						"memory": property.New("512Mi"),
					})),
				})),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAzureContainerAppWithDomainName(t *testing.T) {
	server := makeAzureTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: azureURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"image":      property.New("myapp:latest"),
			"domainName": property.New("api.example.com"),
		}),
	})

	require.NoError(t, err)
}
