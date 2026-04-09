package gcp

// Tests for Compute Engine (MIG) service routing and resource creation.
// Services with 0 ports, multiple ports, or host-mode ports route to Compute Engine;
// services with exactly 1 ingress port route to Cloud Run.

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

const (
	gcpInstanceGroupManagerType = "gcp:compute/regionInstanceGroupManager:RegionInstanceGroupManager"
	gcpInstanceTemplateType     = "gcp:compute/instanceTemplate:InstanceTemplate"
	gcpHealthCheckType          = "gcp:compute/healthCheck:HealthCheck"
	gcpFirewallType             = "gcp:compute/firewall:Firewall"
)

// --- Routing predicate ---

func TestPortlessServiceRoutesToComputeEngine(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)
	assert.Equal(t, 0, countType(*records, gcpCloudRunServiceType), "portless service should not use Cloud Run")
	assert.Equal(t, 1, countType(*records, gcpInstanceGroupManagerType), "portless service should use Compute Engine MIG")
}

func TestSingleIngressPortRoutesToCloudRun(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app": testutil.ServiceWithPorts("nginx:latest", testutil.IngressPort(8080)),
		}),
	})

	require.NoError(t, err)
	assert.Equal(t, 1, countType(*records, gcpCloudRunServiceType),
		"single ingress port should use Cloud Run")
	assert.Equal(t, 0, countType(*records, gcpInstanceGroupManagerType),
		"single ingress port should not use Compute Engine")
}

func TestHostPortRoutesToComputeEngine(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"db": testutil.ServiceWithPorts("postgres:16", testutil.HostPort(5432)),
		}),
	})

	require.NoError(t, err)
	assert.Equal(t, 0, countType(*records, gcpCloudRunServiceType), "host port should not use Cloud Run")
	assert.Equal(t, 1, countType(*records, gcpInstanceGroupManagerType), "host port should use Compute Engine MIG")
}

func TestMultipleIngressPortsRoutesToComputeEngine(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app": testutil.ServiceWithPorts("nginx:latest", testutil.IngressPort(80), testutil.IngressPort(443)),
		}),
	})

	require.NoError(t, err)
	assert.Equal(t, 0, countType(*records, gcpCloudRunServiceType),
		"multiple ingress ports should not use Cloud Run")
	assert.Equal(t, 1, countType(*records, gcpInstanceGroupManagerType),
		"multiple ingress ports should use Compute Engine MIG")
}

// --- Compute Engine resource creation ---

func TestComputeEngineCreatesInstanceTemplateAndMIG(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)
	assert.Equal(t, 1, countType(*records, gcpInstanceTemplateType))
	assert.Equal(t, 1, countType(*records, gcpInstanceGroupManagerType))
}

func TestComputeEnginePortlessServiceCreatesHTTPHealthCheckSidecar(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)

	// An HTTP health check (not TCP) should be created for portless services
	hc := findTypeWhere(*records, gcpHealthCheckType, func(m property.Map) bool {
		return !m.Get("httpHealthCheck").IsNull()
	})
	require.NotNil(t, hc, "expected an HTTP health check for portless Compute Engine service")
	assert.InDelta(t, 8080.0, hc.inputs.Get("httpHealthCheck").AsMap().Get("port").AsNumber(), 0,
		"HTTP health check should probe port 8080 (sidecar)")

	// The health check sidecar systemd socket (port 8080) is enabled via cloud-init,
	// not a separate GCP resource — verify the instance template's user-data contains it.
	it := findTypeWhere(*records, gcpInstanceTemplateType, func(_ property.Map) bool { return true })
	require.NotNil(t, it, "expected an instance template")
	userData := it.inputs.Get("metadata").AsMap().Get("user-data").AsString()
	assert.Contains(t, userData, "health.socket",
		"cloud-init should include the health check socket unit for portless services")
}

func TestComputeEngineWithPortCreatesTCPHealthCheck(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"db": testutil.ServiceWithPorts("postgres:16", testutil.HostPort(5432)),
		}),
	})

	require.NoError(t, err)

	// A TCP health check should be created for services with a port
	hc := findTypeWhere(*records, gcpHealthCheckType, func(m property.Map) bool {
		return !m.Get("tcpHealthCheck").IsNull()
	})
	require.NotNil(t, hc, "expected a TCP health check for Compute Engine service with port")
	assert.InDelta(t, 5432.0, hc.inputs.Get("tcpHealthCheck").AsMap().Get("port").AsNumber(), 0)
}

func TestComputeEngineCreatesHealthCheckFirewallRule(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)

	// GCP health check probes come from 130.211.0.0/22 and 35.191.0.0/16
	hcFw := findTypeWhere(*records, gcpFirewallType, func(m property.Map) bool {
		sourceRanges := m.Get("sourceRanges").AsArray()
		for i := range sourceRanges.Len() {
			if sourceRanges.Get(i).AsString() == "130.211.0.0/22" {
				return true
			}
		}
		return false
	})
	require.NotNil(t, hcFw, "expected a firewall rule allowing GCP health check source ranges")
	assert.Equal(t, "INGRESS", hcFw.inputs.Get("direction").AsString())
}

func TestComputeEngineUsesContainerOptimizedOS(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)

	it := findTypeWhere(*records, gcpInstanceTemplateType, func(_ property.Map) bool { return true })
	require.NotNil(t, it)
	disks := it.inputs.Get("disks").AsArray()
	require.Equal(t, 1, disks.Len())
	sourceImage := disks.Get(0).AsMap().Get("sourceImage").AsString()
	assert.Contains(t, sourceImage, "cos-cloud",
		"instance template should use Container-Optimized OS, got: %s", sourceImage)
}

func TestComputeEngineCloudInitContainsImage(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)

	it := findTypeWhere(*records, gcpInstanceTemplateType, func(_ property.Map) bool { return true })
	require.NotNil(t, it)
	userData := it.inputs.Get("metadata").AsMap().Get("user-data").AsString()
	assert.Contains(t, userData, "myapp:worker",
		"cloud-init user-data should contain the service image")
	assert.Contains(t, userData, "docker run",
		"cloud-init user-data should contain a docker run command")
}

func TestComputeEngineCloudInitContainsEnvironment(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"worker": property.New(property.NewMap(map[string]property.Value{
				"image": property.New("myapp:worker"),
				"environment": property.New(property.NewMap(map[string]property.Value{
					"APP_ENV":   property.New("production"),
					"LOG_LEVEL": property.New("info"),
				})),
			})),
		}),
	})

	require.NoError(t, err)

	it := findTypeWhere(*records, gcpInstanceTemplateType, func(_ property.Map) bool { return true })
	require.NotNil(t, it)
	userData := it.inputs.Get("metadata").AsMap().Get("user-data").AsString()
	assert.Contains(t, userData, "APP_ENV=production",
		"cloud-init should embed APP_ENV in the service unit")
	assert.Contains(t, userData, "LOG_LEVEL=info",
		"cloud-init should embed LOG_LEVEL in the service unit")
}

func TestComputeEngineMachineTypeDefaultsToE2MicroWhenNoReservations(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)

	it := findTypeWhere(*records, gcpInstanceTemplateType, func(_ property.Map) bool { return true })
	require.NotNil(t, it)
	assert.Equal(t, "e2-micro", it.inputs.Get("machineType").AsString())
}

func TestComputeEngineMachineTypeSelectedFromReservations(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"worker": property.New(property.NewMap(map[string]property.Value{
				"image": property.New("myapp:worker"),
				"deploy": property.New(property.NewMap(map[string]property.Value{
					"resources": property.New(property.NewMap(map[string]property.Value{
						"reservations": property.New(property.NewMap(map[string]property.Value{
							"cpus":   property.New(1.0),
							"memory": property.New("512M"),
						})),
					})),
				})),
			})),
		}),
	})

	require.NoError(t, err)

	it := findTypeWhere(*records, gcpInstanceTemplateType, func(_ property.Map) bool { return true })
	require.NotNil(t, it)
	// 1 CPU + 512MiB fits e2-medium (1 vCPU, 4 GiB)
	assert.Equal(t, "e2-medium", it.inputs.Get("machineType").AsString())
}

func TestComputeEngineCreatesServiceAccount(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)

	// A dedicated service account should be created for the instance
	sa := findTypeWhere(*records, "gcp:serviceaccount/account:Account", func(m property.Map) bool {
		return strings.Contains(m.Get("displayName").AsString(), "worker")
	})
	require.NotNil(t, sa, "expected a service account for the Compute Engine worker")
}

func TestComputeEngineGrantsRequiredIAMRoles(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)

	iamMembers := *records
	requiredRoles := []string{
		"roles/artifactregistry.reader",
		"roles/logging.logWriter",
		"roles/monitoring.metricWriter",
		"roles/cloudtrace.agent",
	}
	for _, role := range requiredRoles {
		found := findTypeWhere(iamMembers, "gcp:projects/iAMMember:IAMMember", func(m property.Map) bool {
			return m.Get("role").AsString() == role
		})
		assert.NotNil(t, found, "expected IAM member for role %s", role)
	}
}

func TestCloudRunServiceDoesNotGrantComputeIAMRoles(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app": testutil.ServiceWithPorts("nginx:latest", testutil.IngressPort(8080)),
		}),
	})

	require.NoError(t, err)

	// Cloud Run uses the serverless-robot service agent to pull images and has built-in
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
		assert.Nil(t, found, "Cloud Run service should not grant IAM role %s", role)
	}
}

// --- Load balancer integration ---

func TestComputeEngineWithoutIngressPortSkipsLB(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)
	// A portless Compute Engine service should not be added to the load balancer
	assert.Equal(t, 0, countType(*records, "gcp:compute/backendService:BackendService"))
	assert.Equal(t, 0, countType(*records, "gcp:compute/uRLMap:URLMap"))
}

func TestComputeEngineWithIngressPortCreatesGCEBackendInLB(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	// A service with both a host port and an ingress port: two ports means IsCloudRunService=false
	// (→ Compute Engine), but HasIngressPorts()=true (→ LB entry for the ingress port).
	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app": testutil.ServiceWithPorts("myapp:latest", testutil.HostPort(9090), testutil.IngressPort(8080)),
		}),
	})

	require.NoError(t, err)

	// Should create a backend service using the MIG (not a SERVERLESS NEG)
	assert.Equal(t, 0, countType(*records, "gcp:compute/regionNetworkEndpointGroup:RegionNetworkEndpointGroup"),
		"Compute Engine backend should not use a SERVERLESS NEG")
	backend := findTypeWhere(*records, "gcp:compute/backendService:BackendService", func(m property.Map) bool {
		return m.Get("protocol").AsString() == "HTTP"
	})
	require.NotNil(t, backend, "expected an HTTP backend service for Compute Engine with ingress port")
	assert.Equal(t, "EXTERNAL_MANAGED", backend.inputs.Get("loadBalancingScheme").AsString())

	// URL map and forwarding rules should be created
	assert.Equal(t, 2, countType(*records, "gcp:compute/uRLMap:URLMap"))
	assert.Equal(t, 2, countType(*records, "gcp:compute/globalForwardingRule:GlobalForwardingRule"))
}

func TestMixedProjectCloudRunAndComputeEngine(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"web":    testutil.ServiceWithPorts("nginx:latest", testutil.IngressPort(80)),
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)

	// web → Cloud Run, worker → Compute Engine
	assert.Equal(t, 1, countType(*records, gcpCloudRunServiceType))
	assert.Equal(t, 1, countType(*records, gcpInstanceGroupManagerType))

	// Only the Cloud Run service (web) gets an LB entry (worker has no ingress port)
	assert.Equal(t, 1, countType(*records, "gcp:compute/regionNetworkEndpointGroup:RegionNetworkEndpointGroup"))
	assert.Equal(t, 1, countType(*records, "gcp:compute/backendService:BackendService"))
}
