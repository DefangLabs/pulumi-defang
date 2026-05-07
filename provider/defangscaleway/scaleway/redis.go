package scaleway

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/redis"
)

var (
	ErrRedisConfigNil       = errors.New("redis config is nil")
	ErrRedisPasswordMissing = errors.New("REDIS_PASSWORD is required for Scaleway Managed Redis")
)

type RedisResult struct {
	Cluster       *redis.Cluster
	ConnectionURL pulumi.StringOutput
}

func redisNodeType(memMiB int) string {
	switch {
	case memMiB <= 512:
		return "RED1-MICRO"
	case memMiB <= 2048:
		return "RED1-S"
	case memMiB <= 4096:
		return "RED1-M"
	case memMiB <= 8192:
		return "RED1-L"
	default:
		return "RED1-XL"
	}
}

func redisVersionFromImage(image *string) string {
	if image == nil {
		return "7.2.5"
	}
	tag := compose.ParseImageTag(*image)
	if tag == "" {
		return "7.2.5"
	}
	parts := strings.Split(tag, ".")
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return "7.2.5"
	}
	if major <= 6 {
		return "6.2.7"
	}
	return "7.2.5"
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

func redisAddressFromConnectionString(s string) string {
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

	password := compose.GetConfigOrEnvValue(ctx, configProvider, svc, "REDIS_PASSWORD", "")
	args := &redis.ClusterArgs{
		Name:              pulumi.String(serviceName),
		Version:           pulumi.String(redisVersionFromImage(svc.Image)),
		NodeType:          pulumi.String(redisNodeType(svc.GetMemoryMiB())),
		UserName:          pulumi.String("default"),
		PasswordWo:        redisPassword(password),
		PasswordWoVersion: pulumi.Int(1),
		ClusterSize:       pulumi.Int(redisClusterSize(svc)),
		TlsEnabled:        pulumi.Bool(false),
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
			args.TlsEnabled = pulumi.Bool(true)
			args.Acls = redis.ClusterAclArray{
				&redis.ClusterAclArgs{
					Ip:          pulumi.String("0.0.0.0/0"),
					Description: pulumi.String("Allow public Redis access"),
				},
			}
		}
	} else {
		args.TlsEnabled = pulumi.Bool(true)
		args.Acls = redis.ClusterAclArray{
			&redis.ClusterAclArgs{
				Ip:          pulumi.String("0.0.0.0/0"),
				Description: pulumi.String("Allow public Redis access"),
			},
		}
	}

	cluster, err := redis.NewCluster(ctx, serviceName, args, opts...)
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
		cluster.ConnectionString.ApplyT(redisAddressFromConnectionString).(pulumi.StringOutput),
	)).(pulumi.StringOutput)

	return &RedisResult{Cluster: cluster, ConnectionURL: connectionURL}, nil
}
