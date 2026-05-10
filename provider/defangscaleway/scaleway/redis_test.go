package scaleway

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T {
	return &v
}

func TestRedisVersionFromImage(t *testing.T) {
	// Scaleway only offers Redis 8.4.0 as of 2026-05
	assert.Equal(t, "8.4.0", redisVersionFromImage(nil))
	assert.Equal(t, "8.4.0", redisVersionFromImage(ptr("redis:7")))
	assert.Equal(t, "8.4.0", redisVersionFromImage(ptr("redis:6")))
}

func TestRedisAddressFromConnectionString(t *testing.T) {
	assert.Equal(t, "10.0.0.7:6379", RedisAddressFromConnectionString("redis://10.0.0.7:6379"))
	assert.Equal(t, "10.0.0.7:6379", RedisAddressFromConnectionString("rediss://default:secret@10.0.0.7:6379/0"))
}

func TestRedisNodeType(t *testing.T) {
	assert.Equal(t, "RED1-MICRO", redisNodeType(512))
	assert.Equal(t, "RED1-S", redisNodeType(2048))
	assert.Equal(t, "RED1-M", redisNodeType(4096))
	assert.Equal(t, "RED1-L", redisNodeType(8192))
	assert.Equal(t, "RED1-XL", redisNodeType(16384))
}

func TestCreateRedisCluster(t *testing.T) {
	mocks := &recordingMocks{}
	password := "secret"
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := CreateRedis(ctx, &mockConfigProvider{}, "cache", compose.ServiceConfig{
			Image: &[]string{"redis:7"}[0],
			Redis: &compose.RedisConfig{},
			Environment: map[string]*string{
				"REDIS_PASSWORD": &password,
			},
		}, nil)
		return err
	}, pulumi.WithMocks("proj", "stack", mocks))

	require.NoError(t, err)
	cluster := mocks.findType("scaleway:redis/cluster:Cluster")
	require.NotNil(t, cluster)
	assert.Equal(t, "8.4.0", cluster.inputs[resource.PropertyKey("version")].StringValue())
	assert.Equal(t, "RED1-MICRO", cluster.inputs[resource.PropertyKey("nodeType")].StringValue())
	assert.Equal(t, "default", cluster.inputs[resource.PropertyKey("userName")].StringValue())
	assert.True(t, cluster.inputs[resource.PropertyKey("tlsEnabled")].BoolValue())
	assert.Len(t, cluster.inputs[resource.PropertyKey("acls")].ArrayValue(), 1)
}

func TestCreateRedisAttachesPrivateNetwork(t *testing.T) {
	mocks := &recordingMocks{}
	password := "secret"
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		pn, err := network.NewPrivateNetwork(ctx, "pn", &network.PrivateNetworkArgs{})
		if err != nil {
			return err
		}
		_, err = CreateRedis(ctx, &mockConfigProvider{}, "cache", compose.ServiceConfig{
			Image: &[]string{"redis:7"}[0],
			Redis: &compose.RedisConfig{},
			Environment: map[string]*string{
				"REDIS_PASSWORD": &password,
			},
		}, &SharedInfra{Zone: "fr-par-1", PrivateNetwork: pn})
		return err
	}, pulumi.WithMocks("proj", "stack", mocks))

	require.NoError(t, err)
	cluster := mocks.findType("scaleway:redis/cluster:Cluster")
	require.NotNil(t, cluster)
	assert.False(t, cluster.inputs[resource.PropertyKey("tlsEnabled")].BoolValue())
	assert.True(t, cluster.inputs[resource.PropertyKey("acls")].IsNull())
	privateNetworks := cluster.inputs[resource.PropertyKey("privateNetworks")].ArrayValue()
	require.Len(t, privateNetworks, 1)
	assert.Equal(t, "fr-par-1", privateNetworks[0].ObjectValue()[resource.PropertyKey("zone")].StringValue())
}
