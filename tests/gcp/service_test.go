package gcp

// Service is the standalone Cloud Run component for GCP. These tests verify
// that the Service component correctly handles a variety of input
// configurations: image, ports, build config, environment vars, deploy config.

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

func TestConstructGcpCloudRunServiceWithImage(t *testing.T) {
	server := testutil.MakeGcpTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("nginx:latest"),
		}),
	})

	require.NoError(t, err)
}

func TestConstructGcpCloudRunServiceWithIngressPort(t *testing.T) {
	server := testutil.MakeGcpTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("myapp:latest"),
			"ports": property.New(property.NewArray([]property.Value{
				testutil.IngressPort(8080),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructGcpCloudRunServiceWithBuild(t *testing.T) {
	t.Skip("build support for GCP standalone Service not yet implemented")
	server := testutil.MakeGcpTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"build": property.New(property.NewMap(map[string]property.Value{
				"context": property.New("./app"),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructGcpCloudRunServiceWithEnvironment(t *testing.T) {
	server := testutil.MakeGcpTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Service"),
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

func TestConstructGcpCloudRunServiceWithDeploy(t *testing.T) {
	server := testutil.MakeGcpTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("myapp:latest"),
			"deploy": property.New(property.NewMap(map[string]property.Value{
				"replicas": property.New(float64(3)),
				"resources": property.New(property.NewMap(map[string]property.Value{
					"reservations": property.New(property.NewMap(map[string]property.Value{
						"cpus":   property.New(1.0),
						"memory": property.New("512Mi"),
					})),
				})),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructGcpCloudRunServiceWithHealthCheck(t *testing.T) {
	server := testutil.MakeGcpTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("myapp:latest"),
			"ports": property.New(property.NewArray([]property.Value{
				testutil.IngressPort(8080),
			})),
			"healthCheck": property.New(property.NewMap(map[string]property.Value{
				"test": property.New(property.NewArray([]property.Value{
					property.New("CMD"),
					property.New("curl"),
					property.New("-f"),
					property.New("http://localhost:8080/health"),
				})),
				"intervalSeconds": property.New(float64(15)),
				"timeoutSeconds":  property.New(float64(3)),
				"retries":         property.New(float64(5)),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructGcpCloudRunServiceWithDomainName(t *testing.T) {
	server := testutil.MakeGcpTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"image":      property.New("myapp:latest"),
			"domainName": property.New("api.example.com"),
		}),
	})

	require.NoError(t, err)
}
