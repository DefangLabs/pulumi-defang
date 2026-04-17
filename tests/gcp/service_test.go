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

// Standalone Service has no shared VPC (Infra is not passed from the SDK), so VpcAccess
// must not be set on the Cloud Run template. VPC-backed deployments go through the
// Project component, which provisions the full GlobalConfig.
func TestConstructGcpCloudRunServiceSkipsVpcAccessWhenStandalone(t *testing.T) {
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
	assert.True(t, template.Get("vpcAccess").IsNull(),
		"standalone Service should not attach VpcAccess; got %v", template.Get("vpcAccess"))
}

func TestConstructGcpServiceCreatesServiceAccount(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("myapp:latest"),
		}),
	})

	require.NoError(t, err)

	sa := findTypeWhere(*records, "gcp:serviceaccount/account:Account", func(m property.Map) bool {
		d := m.Get("description")
		return !d.IsNull() && strings.HasPrefix(d.AsString(), "Service Account used by run services of")
	})
	require.NotNil(t, sa, "expected a service account for the standalone Service component")
}

func TestConstructGcpServiceDoesNotGrantComputeIAMRoles(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("myapp:latest"),
		}),
	})

	require.NoError(t, err)

	// Cloud Run uses the serverless-robot service agent for image pulls and has built-in
	// logging/metrics/tracing — the user SA does not need explicit project-level IAM roles.
	computeOnlyRoles := []string{
		"roles/artifactregistry.reader",
		"roles/logging.logWriter",
		"roles/monitoring.metricWriter",
		"roles/cloudtrace.agent",
	}
	for _, role := range computeOnlyRoles {
		found := findTypeWhere(*records, "gcp:projects/iAMMember:IAMMember", func(m property.Map) bool {
			return m.Get("role").AsString() == role
		})
		assert.Nil(t, found, "Cloud Run Service component should not grant IAM role %s", role)
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
