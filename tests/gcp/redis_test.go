package gcp

// Redis tests cover the standalone Redis component (defang-gcp:index:Redis) backed by
// GCP Memorystore. Detailed helper function unit tests live in
// provider/defanggcp/gcp/memorystore_test.go.

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

const gcpRedisInstanceType = "gcp:redis/instance:Instance"

func TestConstructGcpMemorystore(t *testing.T) {
	server := testutil.MakeGcpTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Redis"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("redis:7"),
			"redis": property.New(property.NewMap(map[string]property.Value{})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructGcpMemorystoreCreatesInstance(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Redis"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("redis:7"),
			"redis": property.New(property.NewMap(map[string]property.Value{})),
		}),
	})

	require.NoError(t, err)
	assert.Equal(t, 1, countType(*records, gcpRedisInstanceType))
}

func TestConstructGcpMemorystoreInstanceConfig(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Redis"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("redis:7"),
			"redis": property.New(property.NewMap(map[string]property.Value{})),
		}),
	})

	require.NoError(t, err)

	inst := findTypeWhere(*records, gcpRedisInstanceType, func(m property.Map) bool { return true })
	require.NotNil(t, inst, "expected a Redis instance")
	assert.Equal(t, "STANDARD_HA", inst.inputs.Get("tier").AsString())
	assert.Equal(t, "PRIVATE_SERVICE_ACCESS", inst.inputs.Get("connectMode").AsString())
	assert.InDelta(t, 1.0, inst.inputs.Get("memorySizeGb").AsNumber(), 0)
}

func TestConstructGcpMemorystoreVersionFromImage(t *testing.T) {
	tests := []struct {
		image   string
		wantVer string
	}{
		{"redis:7", "REDIS_7_0"},
		{"redis:7.2", "REDIS_7_2"},
		{"redis:6.2.6", "REDIS_6_X"},
		{"redis:6", "REDIS_6_X"},
	}

	for _, tc := range tests {
		t.Run(tc.image, func(t *testing.T) {
			mock, records := collectResources()
			server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

			_, err := server.Construct(p.ConstructRequest{
				Urn: testutil.GcpURN("Redis"),
				Inputs: property.NewMap(map[string]property.Value{
					"image": property.New(tc.image),
					"redis": property.New(property.NewMap(map[string]property.Value{})),
				}),
			})

			require.NoError(t, err)
			inst := findTypeWhere(*records, gcpRedisInstanceType, func(m property.Map) bool { return true })
			require.NotNil(t, inst)
			assert.Equal(t, tc.wantVer, inst.inputs.Get("redisVersion").AsString())
		})
	}
}

func TestConstructGcpMemorystoreNoImageUsesDefaultVersion(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Redis"),
		Inputs: property.NewMap(map[string]property.Value{
			"redis": property.New(property.NewMap(map[string]property.Value{})),
		}),
	})

	require.NoError(t, err)
	inst := findTypeWhere(*records, gcpRedisInstanceType, func(m property.Map) bool { return true })
	require.NotNil(t, inst)
	// No image tag → no redisVersion set, GCP picks the default
	assert.True(t, inst.inputs.Get("redisVersion").IsNull(), "expected redisVersion to be unset when no image tag")
}

// TestConstructGcpMemorystoreStandaloneNoVPCPeering verifies that the standalone Redis
// component (used outside a Project) does not create VPC peering infrastructure,
// since it has no project-level VPC context.
func TestConstructGcpMemorystoreStandaloneNoVPCPeering(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Redis"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("redis:7"),
			"redis": property.New(property.NewMap(map[string]property.Value{})),
		}),
	})

	require.NoError(t, err)

	peering := findTypeWhere(*records, "gcp:compute/globalAddress:GlobalAddress", func(m property.Map) bool {
		v := m.Get("purpose")
		return !v.IsNull() && v.AsString() == gcpVPCPeeringPurpose
	})
	assert.Nil(t, peering, "standalone Redis component should not create VPC peering infrastructure")
	assert.Equal(t, 0, countType(*records, "gcp:servicenetworking/connection:Connection"))
}
