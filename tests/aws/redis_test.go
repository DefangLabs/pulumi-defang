package aws

// Redis is the standalone ElastiCache Redis component for AWS. These tests verify
// that the Redis component correctly handles a variety of input configurations.

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

// awsConfig is the minimal AWS config needed by standalone AWS components.
var awsConfig = property.New(property.NewMap(map[string]property.Value{
	"vpcID": property.New("vpc-12345"),
}))

func TestConstructAwsRedis(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Redis"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"aws":          awsConfig,
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsRedisWithImage(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Redis"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("redis:7.2"),
			"aws":          awsConfig,
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsRedisWithValkey(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Redis"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("valkey/valkey:8"),
			"aws":          awsConfig,
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsRedisWithCustomPort(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Redis"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("redis:7.2"),
			"ports": property.New(property.NewArray([]property.Value{
				property.New(property.NewMap(map[string]property.Value{
					"target": property.New(float64(6380)),
					"mode":   property.New("host"),
				})),
			})),
			"aws": awsConfig,
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsRedisWithAllowDowntime(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Redis"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("redis:7.2"),
			"redis": property.New(property.NewMap(map[string]property.Value{
				"allowDowntime": property.New(true),
			})),
			"aws": awsConfig,
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsRedisWithVPC(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Redis"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("redis:7.2"),
			"aws": property.New(property.NewMap(map[string]property.Value{
				"vpcID": property.New("vpc-0123456789abcdef0"),
				"privateSubnetDs": property.New(property.NewArray([]property.Value{
					property.New("subnet-private-0"),
					property.New("subnet-private-1"),
				})),
			})),
		}),
	})

	require.NoError(t, err)
}
