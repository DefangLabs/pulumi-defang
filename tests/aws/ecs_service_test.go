package aws

// Service is the standalone ECS service component for AWS. These tests verify
// that the Service component correctly handles a variety of input configurations.
// Tests cover: ports, deploy config, build config, VPC, health check, environment vars.

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

func TestConstructAwsEcsServiceWithImage(t *testing.T) {
	server := testutil.MakeTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("nginx:latest"),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsEcsServiceWithIngressPort(t *testing.T) {
	server := testutil.MakeTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("nginx:latest"),
			"ports": property.New(property.NewArray([]property.Value{
				testutil.IngressPort(8080),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsEcsServiceWithMultiplePorts(t *testing.T) {
	server := testutil.MakeTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("myapp:latest"),
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
	server := testutil.MakeTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("myapp:latest"),
			"build": property.New(property.NewMap(map[string]property.Value{
				"context": property.New("./app"),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsEcsServiceWithBuildAndDockerfile(t *testing.T) {
	server := testutil.MakeTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("myapp:latest"),
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
	server := testutil.MakeTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("nginx:latest"),
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
	server := testutil.MakeTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("nginx:latest"),
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
	server := testutil.MakeTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("myapp:latest"),
			"environment": property.New(property.NewMap(map[string]property.Value{
				"APP_ENV": property.New("production"),
				"PORT":    property.New("8080"),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsEcsServiceWithDeploy(t *testing.T) {
	server := testutil.MakeTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("myapp:latest"),
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
