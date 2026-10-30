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
	errRedisVersionTooOld = errors.New("no Scaleway Redis version satisfies the requested image version")
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

// requestedRedisVersion returns the minimum version implied by the image tag
// ("redis:7.2" -> 7.2.0, "7-alpine" -> 7.0.0), or nil if the tag is absent or
// not numeric (e.g. "latest").
func requestedRedisVersion(svc compose.ServiceConfig) *semver.Version {
	img := svc.StaticImage()
	if img == nil {
		return nil
	}
	tag, _, _ := strings.Cut(compose.ParseImageTag(*img), "-")
	parts := strings.Split(strings.TrimPrefix(tag, "v"), ".")
	if len(parts) > 3 {
		return nil
	}
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	v, err := semver.ParseTolerant(strings.Join(parts, "."))
	if err != nil {
		return nil
	}
	return &v
}

// selectRedisVersion picks the oldest catalog version that is not older than
// the requested one, so the cluster never runs an older engine than Compose
// asked for and is upgraded as little as possible. With no requested version,
// the newest catalog version is used.
func selectRedisVersion(requested *semver.Version, catalog []redis.GetClusterVersionsVersion) (string, error) {
	if len(catalog) == 0 {
		return "", errRedisVersionsUnavailable
	}
	var best *semver.Version
	for _, entry := range catalog {
		parsed, err := semver.Parse(entry.Version)
		if err != nil {
			return "", fmt.Errorf("parsing Scaleway Redis version %q: %w", entry.Version, err)
		}
		if requested != nil && parsed.LT(*requested) {
			continue
		}
		if best == nil || (requested != nil && parsed.LT(*best)) || (requested == nil && parsed.GT(*best)) {
			best = &parsed
		}
	}
	if best == nil {
		return "", fmt.Errorf("%w: Compose requests Redis %s or newer", errRedisVersionTooOld, requested)
	}
	return best.String(), nil
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
		} else {
			return nil, errRedisPrivateNetwork
		}
	} else {
		return nil, errRedisPrivateNetwork
	}

	// Scaleway retires engine versions independently of Compose image tags.
	// Pick from its current catalog in the cluster's own zone; the image tag
	// is a minimum, see selectRedisVersion.
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
	version, err := selectRedisVersion(requestedRedisVersion(svc), versions.Versions)
	if err != nil {
		return nil, err
	}
	args.Version = pulumi.String(version)

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
