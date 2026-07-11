package aws

import (
	"encoding/json"
	"testing"

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

func TestRedisAliasURNs(t *testing.T) {
	tests := []struct {
		name    string
		config  string // value for defang-aws:redis-aliases
		service string
		kind    string
		want    string // expected URN, "" for no alias
		wantErr bool
	}{
		{name: "unset yields no aliases"},
		{
			name:    "returns URN for configured service and kind",
			config:  `{"redis":{"cluster":"urn:pulumi:staging::ecs::aws:memorydb/cluster:Cluster::fabric"}}`,
			service: "redis",
			kind:    "cluster",
			want:    "urn:pulumi:staging::ecs::aws:memorydb/cluster:Cluster::fabric",
		},
		{
			name:    "no alias for other service",
			config:  `{"redis":{"cluster":"urn:x"}}`,
			service: "other",
			kind:    "cluster",
		},
		{
			name:    "no alias for other kind",
			config:  `{"redis":{"cluster":"urn:x"}}`,
			service: "redis",
			kind:    "subnetGroup",
		},
		{
			name:    "invalid JSON errors",
			config:  `{not json`,
			service: "redis",
			kind:    "cluster",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config != "" {
				t.Setenv("PULUMI_CONFIG", `{"defang-aws:redis-aliases": `+mustJSONString(t, tt.config)+`}`)
			}
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				aliases, err := redisAliasURNs(ctx, tt.service, tt.kind)
				if tt.wantErr {
					require.Error(t, err)
					return nil
				}
				require.NoError(t, err)
				if tt.want == "" {
					assert.Empty(t, aliases)
				} else {
					require.Len(t, aliases, 1)
					assert.Equal(t, pulumi.URN(tt.want), aliases[0].URN)
				}
				return nil
			}, pulumi.WithMocks("ecs", "staging", ssmMocks{}))
			require.NoError(t, err)
		})
	}
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

// mustJSONString wraps a raw string as a JSON string literal.
func mustJSONString(t *testing.T, s string) string {
	t.Helper()
	b, err := json.Marshal(s)
	require.NoError(t, err)
	return string(b)
}
