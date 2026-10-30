package scaleway

import (
	"os"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const redisPasswordKey = "REDIS_PASSWORD"

func ptr[T any](v T) *T {
	return &v
}

type redisCatalogMocks struct {
	recordingMocks
	version string
	err     error
	zone    string
}

func (m *redisCatalogMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	if args.Token != "scaleway:redis/getClusterVersions:getClusterVersions" {
		return m.recordingMocks.Call(args)
	}
	m.zone = args.Args["zone"].StringValue()
	versions := []any{}
	if m.version != "" {
		versions = append(versions, map[string]any{"version": m.version})
	}
	return resource.NewPropertyMapFromMap(map[string]any{"versions": versions}), m.err
}

func TestCreateRedisUsesSupportedVersion(t *testing.T) {
	for _, tt := range []struct {
		name    string
		version string
		apiErr  error
		wantErr string
	}{
		{name: "current catalog", version: "8.6.6"},
		{name: "catalog changes", version: "9.0.1"},
		{name: "no available versions", wantErr: "no supported Scaleway Redis versions"},
		{name: "API failure", apiErr: os.ErrPermission, wantErr: "permission denied"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mocks := &redisCatalogMocks{version: tt.version, err: tt.apiErr}
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				pn, err := network.NewPrivateNetwork(ctx, "pn", &network.PrivateNetworkArgs{})
				if err != nil {
					return err
				}
				_, err = CreateRedis(ctx, &mockConfigProvider{}, "cache", compose.ServiceConfig{
					Image:       compose.ImageFromPtr(ptr("redis:6.2")),
					Redis:       &compose.RedisConfig{},
					Environment: compose.Environment{redisPasswordKey: pulumi.String("secret")},
				}, &SharedInfra{Zone: "nl-ams-1", PrivateNetwork: pn})
				return err
			}, pulumi.WithMocks("proj", "stack", mocks))
			assert.Equal(t, "nl-ams-1", mocks.zone)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, mocks.findType("scaleway:redis/cluster:Cluster"))
				return
			}
			require.NoError(t, err)
			cluster := mocks.findType("scaleway:redis/cluster:Cluster")
			require.NotNil(t, cluster)
			assert.Equal(t, tt.version, cluster.inputs["version"].StringValue())
		})
	}
}

func TestRedisAddressFromConnectionString(t *testing.T) {
	assert.Equal(t, "10.0.0.7:6379", RedisAddressFromConnectionString("redis://10.0.0.7:6379"))
	assert.Equal(t, "10.0.0.7:6379", RedisAddressFromConnectionString("rediss://default:secret@10.0.0.7:6379/0"))
}

func TestRedisNodeType(t *testing.T) {
	assert.Equal(t, "RED1-MICRO", redisNodeType(512))
	assert.Equal(t, "RED1-MICRO", redisNodeType(2048))
	assert.Equal(t, "RED1-2XS", redisNodeType(2049))
	assert.Equal(t, "RED1-2XS", redisNodeType(4096))
	assert.Equal(t, "RED1-XS", redisNodeType(8192))
	assert.Equal(t, "RED1-S", redisNodeType(12288))
	assert.Equal(t, "RED1-M", redisNodeType(16384))
	assert.Equal(t, "RED1-L", redisNodeType(32768))
	assert.Equal(t, "RED1-XL", redisNodeType(65536))
	assert.Equal(t, "RED1-XL", redisNodeType(1<<20))
}

func TestCreateRedisClusterRequiresPrivateNetwork(t *testing.T) {
	mocks := &recordingMocks{}
	password := "secret"
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := CreateRedis(ctx, &mockConfigProvider{}, "cache", compose.ServiceConfig{
			Image: compose.ImageFromPtr(&[]string{"redis:7"}[0]),
			Redis: &compose.RedisConfig{},
			Environment: compose.Environment{
				redisPasswordKey: pulumi.String(password),
			},
		}, nil)
		return err
	}, pulumi.WithMocks("proj", "stack", mocks))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "private network")
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
			Image: compose.ImageFromPtr(&[]string{"redis:7"}[0]),
			Redis: &compose.RedisConfig{},
			Environment: compose.Environment{
				redisPasswordKey: pulumi.String(password),
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

func TestCreateRedisTransitEncryptionRecipe(t *testing.T) {
	t.Setenv("PULUMI_CONFIG", `{"defang-scaleway:transit-encryption-disabled": "false"}`)
	mocks := &recordingMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		pn, err := network.NewPrivateNetwork(ctx, "pn", &network.PrivateNetworkArgs{})
		if err != nil {
			return err
		}
		_, err = CreateRedis(ctx, &mockConfigProvider{}, "cache", compose.ServiceConfig{
			Image:       compose.ImageFromPtr(&[]string{"redis:7"}[0]),
			Redis:       &compose.RedisConfig{},
			Environment: compose.Environment{redisPasswordKey: pulumi.String("secret")},
		}, &SharedInfra{Zone: "fr-par-1", PrivateNetwork: pn})
		return err
	}, pulumi.WithMocks("proj", "stack", mocks))

	require.NoError(t, err)
	cluster := mocks.findType("scaleway:redis/cluster:Cluster")
	require.NotNil(t, cluster)
	assert.True(t, cluster.inputs[resource.PropertyKey("tlsEnabled")].BoolValue())
}
