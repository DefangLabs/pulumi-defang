package scaleway

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/blang/semver"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/redis"
)

var (
	ErrRedisConfigNil       = errors.New("redis config is nil")
	ErrRedisPasswordMissing = errors.New("redis password is required for Scaleway Managed Redis")
	errRedisPrivateNetwork  = errors.New(
		"redis requires a private network for security; public access is not allowed",
	)
	errRedisVersionsUnavailable = errors.New(
		"no supported Scaleway Redis versions available; choose another deployment zone",
	)
)

type RedisResult struct {
	Cluster       *redis.Cluster
	ConnectionURL pulumi.StringOutput
}

const redisVersionProperty = "version"

// redisNodeTypes are the managed Redis node types by size. Memory is the size
// the Scaleway console shows ("2 GB"), taken as GiB, so `memory: 2g` in
// compose.yaml selects the node labelled 2 GB; see postgresDevNodes.
var redisNodeTypes = []struct {
	name   string
	memMiB int
}{
	{"RED1-MICRO", 2048},
	{"RED1-2XS", 4096},
	{"RED1-XS", 8192},
	{"RED1-S", 12288},
	{"RED1-M", 16384},
	{"RED1-L", 32768},
	{"RED1-XL", 65536},
}

// redisNodeType returns the smallest node type with at least memMiB of
// memory, or the largest.
func redisNodeType(memMiB int) string {
	for _, n := range redisNodeTypes {
		if n.memMiB >= memMiB {
			return n.name
		}
	}
	return redisNodeTypes[len(redisNodeTypes)-1].name
}

func redisPassword(password pulumi.StringInput) pulumi.StringPtrInput {
	return password.ToStringOutput().ApplyT(func(p string) (*string, error) {
		if p == "" {
			return nil, fmt.Errorf("%w; set it with `defang config set REDIS_PASSWORD`", ErrRedisPasswordMissing)
		}
		return &p, nil
	}).(pulumi.StringPtrOutput)
}

func redisClusterSize(svc compose.ServiceConfig) int {
	replicas := svc.GetReplicas()
	if replicas <= 1 {
		return 1
	}
	if replicas == 2 {
		return 2
	}
	return int(replicas)
}

func RedisAddressFromConnectionString(s string) string {
	s = strings.TrimPrefix(s, "redis://")
	s = strings.TrimPrefix(s, "rediss://")
	if at := strings.LastIndex(s, "@"); at >= 0 {
		s = s[at+1:]
	}
	if slash := strings.IndexByte(s, '/'); slash >= 0 {
		s = s[:slash]
	}
	return s
}

func CreateRedis(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	serviceName string,
	svc compose.ServiceConfig,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*RedisResult, error) {
	if svc.Redis == nil {
		return nil, ErrRedisConfigNil
	}
	if configProvider == nil {
		configProvider = &compose.PulumiConfigProvider{}
	}

	password, _ := compose.GetConfigOrEnvValue(ctx, configProvider, svc, "REDIS_PASSWORD", "")
	args := &redis.ClusterArgs{
		Name:              pulumi.String(serviceName),
		NodeType:          pulumi.String(redisNodeType(svc.GetMemoryMiB())),
		UserName:          pulumi.String("default"),
		PasswordWo:        redisPassword(password),
		PasswordWoVersion: pulumi.Int(1),
		ClusterSize:       pulumi.Int(redisClusterSize(svc)),
		TlsEnabled:        pulumi.Bool(!TransitEncryptionDisabled.Get(ctx)),
	}
	if infra != nil {
		if infra.ProjectID != "" {
			args.ProjectId = pulumi.StringPtr(infra.ProjectID)
		}
		if infra.Zone != "" {
			args.Zone = pulumi.StringPtr(infra.Zone)
		}
		if infra.PrivateNetwork != nil {
			args.PrivateNetworks = redis.ClusterPrivateNetworkArray{
				&redis.ClusterPrivateNetworkArgs{
					Id: infra.PrivateNetwork.ID(),
				},
			}
			if infra.Zone != "" {
				args.PrivateNetworks = redis.ClusterPrivateNetworkArray{
					&redis.ClusterPrivateNetworkArgs{
						Id:   infra.PrivateNetwork.ID(),
						Zone: pulumi.StringPtr(infra.Zone),
					},
				}
			}
		} else {
			return nil, errRedisPrivateNetwork
		}
	} else {
		return nil, errRedisPrivateNetwork
	}

	// Scaleway retires engine versions independently of Compose image tags.
	// Resolve its current supported version in the cluster's own zone.
	var invokeOpts []pulumi.InvokeOption
	for _, option := range opts {
		if invokeOption, ok := option.(pulumi.InvokeOption); ok {
			invokeOpts = append(invokeOpts, invokeOption)
		}
	}
	var zone *string
	if infra.Zone != "" {
		zone = &infra.Zone
	}
	versions, err := redis.GetClusterVersions(ctx, &redis.GetClusterVersionsArgs{Zone: zone}, invokeOpts...)
	if err != nil {
		return nil, fmt.Errorf("getting supported Scaleway Redis versions: %w", err)
	}
	if len(versions.Versions) == 0 {
		return nil, errRedisVersionsUnavailable
	}
	var latest semver.Version
	for _, version := range versions.Versions {
		parsed, parseErr := semver.Parse(version.Version)
		if parseErr != nil {
			return nil, fmt.Errorf("parsing Scaleway Redis version %q: %w", version.Version, parseErr)
		}
		if parsed.GT(latest) {
			latest = parsed
		}
	}
	args.Version = pulumi.String(latest.String())

	// Catalog rotation must not migrate an existing cluster on an unchanged deploy.
	clusterOpts := common.MergeOptions(opts, pulumi.IgnoreChanges([]string{redisVersionProperty}))
	cluster, err := redis.NewCluster(ctx, serviceName, args, clusterOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating Scaleway Redis cluster: %w", err)
	}

	connectionURL := pulumi.ToSecret(pulumi.Sprintf("%s://default:%s@%s",
		cluster.TlsEnabled.ApplyT(func(enabled *bool) string {
			if enabled != nil && *enabled {
				return "rediss"
			}
			return "redis"
		}).(pulumi.StringOutput),
		password,
		cluster.ConnectionString.ApplyT(RedisAddressFromConnectionString).(pulumi.StringOutput),
	)).(pulumi.StringOutput)

	return &RedisResult{Cluster: cluster, ConnectionURL: connectionURL}, nil
}
