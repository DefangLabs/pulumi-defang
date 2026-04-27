package aws

// Service is the standalone ECS service component for AWS. These tests verify
// that the Service component correctly handles a variety of input configurations.
// Tests cover: ports, deploy config, build config, VPC, health check, environment vars.

import (
	"encoding/json"
	"strings"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	awsprov "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

func TestConstructAwsEcsServiceWithImage(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":       property.New("nginx:latest"),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsEcsServiceWithIngressPort(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":       property.New("nginx:latest"),
			"ports": property.New(property.NewArray([]property.Value{
				testutil.IngressPort(8080),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsEcsServiceWithMultiplePorts(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":       property.New("myapp:latest"),
			"ports": property.New(property.NewArray([]property.Value{
				testutil.IngressPort(8080),
				property.New(property.NewMap(map[string]property.Value{
					"target":      property.New(float64(9090)),
					"mode":        property.New("host"),
					"appProtocol": property.New("http"),
				})),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsEcsServiceWithBuild(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":       property.New("myapp:latest"),
			"build": property.New(property.NewMap(map[string]property.Value{
				"context": property.New("./app"),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsEcsServiceWithBuildAndDockerfile(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":       property.New("myapp:latest"),
			"build": property.New(property.NewMap(map[string]property.Value{
				"context":    property.New("./app"),
				"dockerfile": property.New("docker/Dockerfile.prod"),
				"target":     property.New("production"),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsEcsServiceWithVPC(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":       property.New("nginx:latest"),
			"aws": property.New(property.NewMap(map[string]property.Value{
				"vpcId": property.New("vpc-0123456789abcdef0"),
				"subnetIds": property.New(property.NewArray([]property.Value{
					property.New("subnet-0123456789abcdef0"),
					property.New("subnet-0123456789abcdef1"),
				})),
				"privateSubnetIds": property.New(property.NewArray([]property.Value{
					property.New("subnet-private-0"),
					property.New("subnet-private-1"),
				})),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsEcsServiceWithHealthCheck(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":       property.New("nginx:latest"),
			"ports": property.New(property.NewArray([]property.Value{
				testutil.IngressPort(80),
			})),
			"healthCheck": property.New(property.NewMap(map[string]property.Value{
				"test": property.New(property.NewArray([]property.Value{
					property.New("CMD"),
					property.New("curl"),
					property.New("-f"),
					property.New("http://localhost/"),
				})),
				"intervalSeconds": property.New(float64(30)),
				"timeoutSeconds":  property.New(float64(5)),
				"retries":         property.New(float64(3)),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsEcsServiceWithEnvironment(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":       property.New("myapp:latest"),
			"environment": property.New(property.NewMap(map[string]property.Value{
				"APP_ENV": property.New("production"),
				"PORT":    property.New("8080"),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsEcsServiceWithDeploy(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":       property.New("myapp:latest"),
			"deploy": property.New(property.NewMap(map[string]property.Value{
				"replicas": property.New(float64(2)),
				"resources": property.New(property.NewMap(map[string]property.Value{
					"reservations": property.New(property.NewMap(map[string]property.Value{
						"cpus":   property.New(0.5),
						"memory": property.New("1Gi"),
					})),
				})),
			})),
		}),
	})

	require.NoError(t, err)
}

// constructAwsServiceContainerDef constructs a standalone ECS Service with the
// given environment map and returns the single ContainerDefinition parsed from
// the TaskDefinition's containerDefinitions JSON. Shared between the
// secret-ref and DEFANG_SERVICE env tests.
func constructAwsServiceContainerDef(t *testing.T, env map[string]property.Value) awsprov.ContainerDefinition {
	t.Helper()
	inputs := map[string]property.Value{
		"projectName": property.New("myproject"),
		"image":       property.New("myapp:latest"),
	}
	if env != nil {
		inputs["environment"] = property.New(property.NewMap(env))
	}
	var taskDef property.Map
	mock := &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			if args.TypeToken == "aws:ecs/taskDefinition:TaskDefinition" {
				taskDef = args.Inputs
			}
			return args.Name, args.Inputs, nil
		},
	}
	server := testutil.MakeAwsTestServer(integration.WithMocks(mock))
	_, err := server.Construct(p.ConstructRequest{
		Urn:    testutil.AwsURN("Service"),
		Inputs: property.NewMap(inputs),
	})
	require.NoError(t, err)
	require.NotEqual(t, 0, taskDef.Len(), "expected TaskDefinition to be registered")

	// containerDefinitions is a JSON-encoded string; decode using the provider's
	// ContainerDefinition type so field/tag drift stays in sync automatically.
	var defs []awsprov.ContainerDefinition
	require.NoError(t, json.Unmarshal([]byte(taskDef.Get("containerDefinitions").AsString()), &defs))
	require.Len(t, defs, 1)
	return defs[0]
}

// TestConstructAwsEcsServiceEmitsSecretRefs is the mocks-based complement to
// the example's integration test: it verifies that env vars matching the bare
// ${VAR} pattern (per compose.GetConfigName) are emitted as ECS TaskDefinition
// `secrets` entries with an SSM `valueFrom` ARN, NOT as plaintext `environment`
// entries (which would leak into state).
func TestConstructAwsEcsServiceEmitsSecretRefs(t *testing.T) {
	def := constructAwsServiceContainerDef(t, map[string]property.Value{
		"LITERAL": property.New("plain-value"),
		"SECRET":  property.New("${CONFIG}"), // bare ref → Secrets entry
		"OTHER":   property.New("${CONFIG}"), // same secret, second env var
		// Non-bare interpolation (prefix${CONFIG}suffix) isn't covered here
		// because AWS's GetConfigValue errors when the SSM param isn't
		// fetchable in the mock — which breaks the synchronous
		// containerDefinitions JSON build.
	})

	envByName := map[string]string{}
	for _, e := range def.Environment {
		envByName[e.Name] = e.Value
	}
	secByName := map[string]string{}
	for _, s := range def.Secrets {
		secByName[s.Name] = s.ValueFrom
	}

	// LITERAL: plaintext in environment, not in secrets
	assert.Equal(t, "plain-value", envByName["LITERAL"])
	_, inSecrets := secByName["LITERAL"]
	assert.False(t, inSecrets)

	// SECRET: in secrets (SSM ARN), NOT in environment
	_, inEnv := envByName["SECRET"]
	assert.False(t, inEnv, "secret env var must not appear in plaintext environment array")
	secretRef, inSecrets := secByName["SECRET"]
	require.True(t, inSecrets, "SECRET must appear in container 'secrets' as a valueFrom ref")
	assert.Contains(t, secretRef, "parameter/Defang/myproject/stack/CONFIG",
		"valueFrom should point at the SSM parameter for CONFIG")
	assert.True(t, strings.HasPrefix(secretRef, "arn:aws:ssm:"))

	// OTHER: second env var, same secret — separate secrets entry with matching SSM path
	otherRef, ok := secByName["OTHER"]
	require.True(t, ok)
	assert.Equal(t, secretRef, otherRef,
		"two env vars pointing at the same secret should share a valueFrom ARN")
}

// TestConstructAwsEcsServiceInjectsDefangServiceEnv verifies that the ECS
// container's environment always contains DEFANG_SERVICE set to the service
// name — runtime code (health checks, log filters, telemetry) relies on it.
func TestConstructAwsEcsServiceInjectsDefangServiceEnv(t *testing.T) {
	// The testutil URN helpers hardcode the resource name as "name".
	const serviceName = "name"
	def := constructAwsServiceContainerDef(t, nil)

	for _, e := range def.Environment {
		if e.Name == "DEFANG_SERVICE" {
			assert.Equal(t, serviceName, e.Value,
				"DEFANG_SERVICE value should match the service name")
			return
		}
	}
	t.Fatal("DEFANG_SERVICE env var not found on ECS container")
}
