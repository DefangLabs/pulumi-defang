package aws

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryDBNodeType(t *testing.T) {
	tests := []struct {
		name     string
		cpus     float64
		memMiB   int
		nodeType string
		expected string
	}{
		{name: "no reservations picks smallest burstable", nodeType: "burstable", expected: "db.t4g.small"},
		{name: "unknown catalog falls back to burstable", nodeType: "bogus", expected: "db.t4g.small"},
		{name: "medium memory fits t4g.medium", memMiB: 2048, nodeType: "burstable", expected: "db.t4g.medium"},
		{name: "burstable overflows into memory-optimized", memMiB: 8192, nodeType: "burstable", expected: "db.r6g.large"},
		{name: "memory-optimized skips burstable", nodeType: "memory-optimized", expected: "db.r6g.large"},
		{name: "large memory scales up", memMiB: 100 * 1024, nodeType: "memory-optimized", expected: "db.r6g.4xlarge"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, memoryDBNodeType(tt.cpus, tt.memMiB, tt.nodeType))
		})
	}
}

func TestMemoryDBParameterGroupFamily(t *testing.T) {
	assert.Equal(t, "memorydb_redis7", memoryDBParameterGroupFamily("redis", ""))
	assert.Equal(t, "memorydb_redis7", memoryDBParameterGroupFamily("redis", "7.1"))
	assert.Equal(t, "memorydb_redis6", memoryDBParameterGroupFamily("redis", "6.2"))
	assert.Equal(t, "memorydb_valkey8", memoryDBParameterGroupFamily("valkey", "8.0"))
	assert.Equal(t, "memorydb_valkey7", memoryDBParameterGroupFamily("valkey", ""))
}

func TestAliasOptions(t *testing.T) {
	urn := "urn:pulumi:staging::ecs::aws:memorydb/cluster:Cluster::fabric"
	svc := compose.ServiceConfig{
		Aliases: map[string]string{
			compose.AliasCluster:     urn,
			compose.AliasSubnetGroup: "",
		},
	}

	assert.Len(t, svc.AliasOptions(compose.AliasCluster), 1)
	assert.Empty(t, svc.AliasOptions(compose.AliasSubnetGroup), "empty URN yields no alias")
	assert.Empty(t, svc.AliasOptions(compose.AliasParameterGroup), "unset kind yields no alias")

	var none compose.ServiceConfig
	assert.Empty(t, none.AliasOptions(compose.AliasCluster), "nil map yields no alias")
}

func TestGetSecretID_ConfigPathRecipe(t *testing.T) {
	t.Run("default path", func(t *testing.T) {
		err := pulumi.RunErr(func(ctx *pulumi.Context) error {
			cp := NewConfigProvider("myproj")
			assert.Equal(t, "/Defang/myproj/mystack/KEY", cp.getSecretID(ctx, "KEY"))
			return nil
		}, pulumi.WithMocks("myproj", "mystack", ssmMocks{}))
		require.NoError(t, err)
	})

	t.Run("config-path override", func(t *testing.T) {
		t.Setenv("PULUMI_CONFIG", `{"defang-aws:config-path": "/defang/ecs/prod/"}`)
		err := pulumi.RunErr(func(ctx *pulumi.Context) error {
			cp := NewConfigProvider("myproj")
			assert.Equal(t, "/defang/ecs/prod/KEY", cp.getSecretID(ctx, "KEY"))
			return nil
		}, pulumi.WithMocks("myproj", "mystack", ssmMocks{}))
		require.NoError(t, err)
	})
}

