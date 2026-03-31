package gcp

// Service is the standalone Cloud Run component for GCP. These tests verify
// that the Service component correctly handles a variety of input
// configurations: image, ports, build config, environment vars, deploy config.

import (
	"strings"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

const gcpCloudRunServiceType = "gcp:cloudrunv2/service:Service"
const gcpVPCPeeringPurpose = "VPC_PEERING"

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

const gcpBuildType = "defang-gcp:defanggcp:Build"

func TestConstructGcpCloudRunServiceWithBuild(t *testing.T) {
	var capturedBuild property.Map
	mocks := &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			if args.TypeToken == gcpBuildType {
				capturedBuild = args.Inputs
			}
			return args.Name, args.Inputs, nil
		},
	}
	server := testutil.MakeGcpTestServer(integration.WithMocks(mocks))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"build": property.New(property.NewMap(map[string]property.Value{
				"context": property.New("./app"),
			})),
		}),
	})

	require.NoError(t, err)
	require.NotEqual(t, 0, capturedBuild.Len(), "expected defang-gcp:defanggcp:Build to be registered")
	// Local context paths are uploaded to GCS; source should be a gs:// URI
	assert.True(t, strings.HasPrefix(capturedBuild.Get("source").AsString(), "gs://"),
		"source should be a GCS URI, got: %s", capturedBuild.Get("source").AsString())
	assert.NotEmpty(t, capturedBuild.Get("sourceDigest").AsString(), "sourceDigest should be set")
	assert.NotEmpty(t, capturedBuild.Get("machineType").AsString(), "machineType should be set")
	assert.NotEqual(t, float64(0), capturedBuild.Get("diskSizeGb").AsNumber(), "diskSizeGb should be set")
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

func TestConstructGcpCloudRunServiceWithoutReservationsHasNoLimits(t *testing.T) {
	var capturedTemplate property.Map
	mocks := &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			if args.TypeToken == gcpCloudRunServiceType {
				capturedTemplate = args.Inputs
			}
			return args.Name, args.Inputs, nil
		},
	}
	server := testutil.MakeGcpTestServer(integration.WithMocks(mocks))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("myapp:latest"),
		}),
	})

	require.NoError(t, err)
	require.NotEqual(t, 0, capturedTemplate.Len(), "expected gcp:cloudrunv2/service:Service to be registered")
	template := capturedTemplate.Get("template").AsMap()
	containers := template.Get("containers").AsArray()
	require.Equal(t, 1, containers.Len())
	limits := containers.Get(0).AsMap().Get("resources").AsMap().Get("limits").AsMap()
	assert.Equal(t, 0, limits.Len(), "expected no resource limits when no reservations defined")
}

func TestConstructGcpCloudRunServiceWithReservationsSetsLimits(t *testing.T) {
	var capturedTemplate property.Map
	mocks := &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			if args.TypeToken == gcpCloudRunServiceType {
				capturedTemplate = args.Inputs
			}
			return args.Name, args.Inputs, nil
		},
	}
	server := testutil.MakeGcpTestServer(integration.WithMocks(mocks))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("myapp:latest"),
			"deploy": property.New(property.NewMap(map[string]property.Value{
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
	require.NotEqual(t, 0, capturedTemplate.Len(), "expected gcp:cloudrunv2/service:Service to be registered")
	template := capturedTemplate.Get("template").AsMap()
	containers := template.Get("containers").AsArray()
	require.Equal(t, 1, containers.Len())
	limits := containers.Get(0).AsMap().Get("resources").AsMap().Get("limits").AsMap()
	assert.Equal(t, "1", limits.Get("cpu").AsString())
	assert.Equal(t, "512Mi", limits.Get("memory").AsString())
}

func TestConstructGcpCloudRunServiceSetsVpcAccess(t *testing.T) {
	var capturedTemplate property.Map
	mocks := &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			if args.TypeToken == gcpCloudRunServiceType {
				capturedTemplate = args.Inputs
			}
			return args.Name, args.Inputs, nil
		},
	}
	server := testutil.MakeGcpTestServer(integration.WithMocks(mocks))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("myapp:latest"),
		}),
	})

	require.NoError(t, err)
	require.NotEqual(t, 0, capturedTemplate.Len(), "expected gcp:cloudrunv2/service:Service to be registered")
	template := capturedTemplate.Get("template").AsMap()
	vpcAccess := template.Get("vpcAccess").AsMap()
	assert.Equal(t, "PRIVATE_RANGES_ONLY", vpcAccess.Get("egress").AsString())
	networkInterfaces := vpcAccess.Get("networkInterfaces").AsArray()
	assert.Equal(t, 1, networkInterfaces.Len())
}

func TestConstructGcpServiceCreatesServiceAccountWithIAMRoles(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("myapp:latest"),
		}),
	})

	require.NoError(t, err)

	// The standalone Service component (service.go) calls createServiceAccount independently;
	// verify the SA is created and the 4 basic IAM roles are granted.
	sa := findTypeWhere(*records, "gcp:serviceaccount/account:Account", func(m property.Map) bool {
		d := m.Get("description")
		return !d.IsNull() && strings.HasPrefix(d.AsString(), "Service Account used by run services of")
	})
	require.NotNil(t, sa, "expected a service account for the standalone Service component")

	requiredRoles := []string{
		"roles/artifactregistry.reader",
		"roles/logging.logWriter",
		"roles/monitoring.metricWriter",
		"roles/cloudtrace.agent",
	}
	for _, role := range requiredRoles {
		found := findTypeWhere(*records, "gcp:projects/iAMMember:IAMMember", func(m property.Map) bool {
			return m.Get("role").AsString() == role
		})
		assert.NotNil(t, found, "expected IAM member for role %s on standalone Service component", role)
	}
}

func TestConstructGcpCloudRunServiceSetsMaxInstanceRequestConcurrency(t *testing.T) {
	var capturedTemplate property.Map
	mocks := &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			if args.TypeToken == gcpCloudRunServiceType {
				capturedTemplate = args.Inputs
			}
			return args.Name, args.Inputs, nil
		},
	}
	server := testutil.MakeGcpTestServer(integration.WithMocks(mocks))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("myapp:latest"),
		}),
	})

	require.NoError(t, err)
	require.NotEqual(t, 0, capturedTemplate.Len(), "expected gcp:cloudrunv2/service:Service to be registered")
	template := capturedTemplate.Get("template").AsMap()
	assert.InDelta(t, float64(80), template.Get("maxInstanceRequestConcurrency").AsNumber(), 0)
}
