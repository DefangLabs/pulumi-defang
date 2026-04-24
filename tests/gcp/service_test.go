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

	// Without reservations, the provider either omits the Resources block
	// entirely or sets it without a Limits map. Accept all three "no limits"
	// shapes: missing resources, missing resources.limits, or empty limits.
	resources := containers.Get(0).AsMap().Get("resources")
	if resources.IsNull() {
		return
	}
	limits := resources.AsMap().Get("limits")
	if limits.IsNull() {
		return
	}
	assert.Equal(t, 0, limits.AsMap().Len(), "expected no resource limits when no reservations defined")
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

// constructGcpServiceEnvs constructs a standalone Cloud Run Service with the
// given environment map and returns (records, envs-by-name) from the resulting
// Cloud Run service's container. Shared between the secret-ref and
// DEFANG_SERVICE env tests.
func constructGcpServiceEnvs(t *testing.T, env map[string]property.Value) ([]resourceRecord, map[string]property.Map) {
	t.Helper()
	inputs := map[string]property.Value{
		"image": property.New("myapp:latest"),
	}
	if env != nil {
		inputs["environment"] = property.New(property.NewMap(env))
	}
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))
	_, err := server.Construct(p.ConstructRequest{
		Urn:    testutil.GcpURN("Service"),
		Inputs: property.NewMap(inputs),
	})
	require.NoError(t, err)

	cr := findTypeWhere(*records, gcpCloudRunServiceType, func(property.Map) bool { return true })
	require.NotNil(t, cr, "expected Cloud Run service to be registered")

	containers := cr.inputs.Get("template").AsMap().Get("containers").AsArray()
	require.Equal(t, 1, containers.Len())
	envs := containers.Get(0).AsMap().Get("envs").AsArray()

	byName := map[string]property.Map{}
	for i := range envs.Len() {
		e := envs.Get(i).AsMap()
		byName[e.Get("name").AsString()] = e
	}
	return *records, byName
}

// TestConstructGcpCloudRunServiceEmitsSecretRefs is the mocks-based complement
// to the example's integration test: it verifies that env vars matching the
// bare ${VAR} pattern (per compose.GetConfigName) are emitted as Cloud Run
// SecretKeyRef entries, not inlined as plaintext values — and that multiple
// references to the same secret share one IAM binding.
func TestConstructGcpCloudRunServiceEmitsSecretRefs(t *testing.T) {
	records, byName := constructGcpServiceEnvs(t, map[string]property.Value{
		"LITERAL": property.New("plain-value"),
		"SECRET":  property.New("${CONFIG}"),             // bare ref → SecretKeyRef
		"OTHER":   property.New("${CONFIG}"),             // same secret, second var → same ref
		"MIXED":   property.New("prefix${CONFIG}suffix"), // not bare → resolved value, not ref
	})

	// LITERAL: plain value, no valueSource
	literal, ok := byName["LITERAL"]
	require.True(t, ok, "LITERAL env var missing")
	assert.Equal(t, "plain-value", literal.Get("value").AsString())
	assert.True(t, literal.Get("valueSource").IsNull())

	// SECRET: SecretKeyRef, NO inline value
	sec, ok := byName["SECRET"]
	require.True(t, ok, "SECRET env var missing")
	assert.True(t, sec.Get("value").IsNull(),
		"secret env var must not have inline value (would leak plaintext into state)")
	secRef := sec.Get("valueSource").AsMap().Get("secretKeyRef").AsMap()
	require.NotEqual(t, 0, secRef.Len(), "SECRET must have valueSource.secretKeyRef")
	assert.NotEmpty(t, secRef.Get("secret").AsString())
	assert.Equal(t, "latest", secRef.Get("version").AsString())

	// OTHER: same secret, also a ref — and the underlying Secret ID matches
	other, ok := byName["OTHER"]
	require.True(t, ok, "OTHER env var missing")
	assert.True(t, other.Get("value").IsNull())
	otherRef := other.Get("valueSource").AsMap().Get("secretKeyRef").AsMap()
	assert.Equal(t, secRef.Get("secret").AsString(), otherRef.Get("secret").AsString(),
		"two env vars pointing at the same secret should reference the same Secret ID")

	// MIXED: interpolation (not a bare ref) — resolved via GetConfigValue, ends up
	// as a plain value, NOT a SecretKeyRef.
	mixed, ok := byName["MIXED"]
	require.True(t, ok, "MIXED env var missing")
	assert.True(t, mixed.Get("valueSource").IsNull(),
		"MIXED is prefix${CONFIG}suffix — not a bare ref, so no SecretKeyRef")

	// Exactly one IamMember should exist — deduped even though two env vars
	// reference the same secret (regression test for the URN collision bug).
	iamCount := countType(records, "gcp:secretmanager/secretIamMember:SecretIamMember")
	assert.Equal(t, 1, iamCount, "expected one IamMember per unique secret")
}

// TestConstructGcpCloudRunServiceInjectsDefangServiceEnv verifies that the
// Cloud Run container's env array always contains DEFANG_SERVICE set to the
// service name — runtime code (health checks, log filters, telemetry) relies
// on it being present.
func TestConstructGcpCloudRunServiceInjectsDefangServiceEnv(t *testing.T) {
	// The testutil URN helpers hardcode the resource name as "name".
	const serviceName = "name"
	_, byName := constructGcpServiceEnvs(t, nil)

	defang, ok := byName["DEFANG_SERVICE"]
	require.True(t, ok, "DEFANG_SERVICE env var not found on Cloud Run container")
	assert.Equal(t, serviceName, defang.Get("value").AsString(),
		"DEFANG_SERVICE value should match the service name")
}
