package azure

import (
	"errors"
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/network/v3"
	redis "github.com/pulumi/pulumi-azure-native-sdk/redisenterprise/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var ErrRedisConfigNil = errors.New("redis config is nil")

// RedisResult holds the Azure Managed Redis cluster and database resources.
type RedisResult struct {
	Cluster       *redis.RedisEnterprise
	Database      *redis.Database
	ConnectionURL pulumi.StringOutput // rediss://:<key>@<host>:10000
}

// selectEnterpriseSkuName picks the Azure Managed Redis SKU name based on memory in MiB.
//
// Strategy:
//   - Balanced_B (standard memory-to-compute ratio, 4:1) for ≤ 12 GiB workloads.
//   - MemoryOptimized_M (high memory-to-compute ratio, 8:1) for > 12 GiB workloads.
//
// Approximate memory sizes per SKU:
//
//	Balanced:        B0=0.5, B1=1, B3=3, B5=6, B10=12 GiB
//	MemoryOptimized: M10=12, M20=24, M50=60, M100=120, M150=175, M250=235 GiB
func selectEnterpriseSkuName(memoryMiB int) string {
	switch {
	case memoryMiB <= 512:
		return "Balanced_B0" // 0.5 GiB
	case memoryMiB <= 1024:
		return "Balanced_B1" // 1 GiB
	case memoryMiB <= 3072:
		return "Balanced_B3" // 3 GiB
	case memoryMiB <= 6144:
		return "Balanced_B5" // 6 GiB
	case memoryMiB <= 12288:
		return "Balanced_B10" // 12 GiB
	case memoryMiB <= 24576:
		return "MemoryOptimized_M20" // 24 GiB
	case memoryMiB <= 61440:
		return "MemoryOptimized_M50" // 60 GiB
	case memoryMiB <= 122880:
		return "MemoryOptimized_M100" // 120 GiB
	case memoryMiB <= 179200:
		return "MemoryOptimized_M150" // 175 GiB
	default:
		return "MemoryOptimized_M250" // 235 GiB
	}
}

// CreateRedisEnterprise creates an Azure Managed Redis cluster and database.
//
// Tier selection: Balanced_B for ≤ 12 GiB, MemoryOptimized_M for > 12 GiB.
// High availability is enabled by default; set HighAvailability=false in recipe
// config to disable it (data loss risk, dev/test only).
// The endpoint is on TLS port 10000.
func CreateRedisEnterprise(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*RedisResult, error) {
	if svc.Redis == nil {
		return nil, ErrRedisConfigNil
	}

	skuName := selectEnterpriseSkuName(svc.GetMemoryMiB())

	// HighAvailability is enabled by default (data is replicated).
	// Disable only when HighAvailability recipe config is explicitly false
	// AND replicas == 1 (i.e. no redundancy required).
	haValue := "Enabled"
	if !HighAvailability.Get(ctx) && svc.GetReplicas() <= 1 {
		haValue = "Disabled"
	}

	cluster, err := redis.NewRedisEnterprise(ctx, serviceName, &redis.RedisEnterpriseArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		// Location:          pulumi.StringPtr(location),
		MinimumTlsVersion: pulumi.String("1.2"),
		HighAvailability:  pulumi.String(haValue),
		Sku: redis.SkuArgs{
			Name: pulumi.String(skuName),
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Azure Managed Redis cluster: %w", err)
	}

	// Azure Redis Enterprise only allows one database per cluster, and it must be named "default".
	// ClusteringPolicy=EnterpriseCluster presents the database as a single Redis instance,
	// which is required for non-cluster-aware clients (Celery/Kombu, channels_redis).
	// OSSCluster (the default) requires all keys in a pipeline to hash to the same slot,
	// which breaks Celery's Kombu transport.
	//
	// When a VNet private endpoint is available the database uses Plaintext protocol so
	// that apps can connect with a plain redis:// URL — the VNet provides network isolation
	// so TLS is not required for security. Without a VNet, Encrypted (TLS) is used.
	useVNet := infra.Networking != nil && infra.DNS != nil && infra.DNS.RedisPrivateZone != nil
	clientProtocol := "Plaintext"
	urlScheme := "redis"
	if !useVNet {
		clientProtocol = "Encrypted"
		urlScheme = "rediss"
	}

	db, err := redis.NewDatabase(ctx, serviceName, &redis.DatabaseArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		ClusterName:       cluster.Name,
		DatabaseName:      pulumi.String("default"),
		ClientProtocol:    pulumi.String(clientProtocol),
		ClusteringPolicy:  pulumi.String("EnterpriseCluster"),
		Port:              pulumi.Int(10000),
	}, append(opts,
		pulumi.Parent(cluster),
		pulumi.ReplaceOnChanges([]string{"clusteringPolicy", "clientProtocol"}),
		pulumi.DeleteBeforeReplace(true),
	)...)
	if err != nil {
		return nil, fmt.Errorf("creating Azure Managed Redis database: %w", err)
	}

	// Retrieve the database access key so callers can build a full connection URL.
	// ListDatabaseKeysOutput is an invoke that runs during the deployment (after
	// the DB exists). Explicit pulumi.Parent(cluster) routes the invoke through
	// the cluster's provider — required because pulumi:disable-default-providers
	// excludes azure-native (see cd/main.go projectConfig).
	keysOut := redis.ListDatabaseKeysOutput(ctx, redis.ListDatabaseKeysOutputArgs{
		ClusterName:       cluster.Name,
		DatabaseName:      db.Name,
		ResourceGroupName: infra.ResourceGroup.Name,
	}, pulumi.Parent(cluster))

	if useVNet {
		if err := createRedisVNetEndpoint(ctx, serviceName, cluster, infra, opts...); err != nil {
			return nil, err
		}
	}

	// Build connection URL using urlScheme (redis:// for VNet/Plaintext, rediss:// for public/TLS).
	connectionURL := pulumi.All(cluster.HostName, keysOut.PrimaryKey()).ApplyT(
		func(args []any) string {
			host := args[0].(string)
			key := args[1].(string)
			return urlScheme + "://:" + key + "@" + host + ":10000"
		},
	).(pulumi.StringOutput)

	return &RedisResult{Cluster: cluster, Database: db, ConnectionURL: connectionURL}, nil
}

// createRedisVNetEndpoint creates a private endpoint and DNS zone group so that traffic to
// the Redis cluster stays within the VNet (Plaintext protocol, plain redis:// URL).
func createRedisVNetEndpoint(
	ctx *pulumi.Context,
	serviceName string,
	cluster *redis.RedisEnterprise,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) error {
	pe, err := network.NewPrivateEndpoint(ctx, serviceName, &network.PrivateEndpointArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		// Location:          pulumi.StringPtr(location),
		Subnet: &network.SubnetTypeArgs{
			Id: infra.Networking.PrivateEndpointsSubnet.ID().ToStringOutput(),
		},
		PrivateLinkServiceConnections: network.PrivateLinkServiceConnectionArray{
			&network.PrivateLinkServiceConnectionArgs{
				Name:                 pulumi.String(serviceName),
				PrivateLinkServiceId: cluster.ID().ToStringOutput(),
				GroupIds:             pulumi.StringArray{pulumi.String("redisEnterprise")},
			},
		},
	}, opts...)
	if err != nil {
		return fmt.Errorf("creating Redis private endpoint: %w", err)
	}

	// Zone group auto-registers an A record in privatelink.redis.azure.net mapping
	// the cluster's private-link FQDN to the private endpoint IP.
	_, err = network.NewPrivateDnsZoneGroup(ctx, serviceName, &network.PrivateDnsZoneGroupArgs{
		ResourceGroupName:       infra.ResourceGroup.Name,
		PrivateEndpointName:     pe.Name,
		PrivateDnsZoneGroupName: pulumi.String("default"),
		PrivateDnsZoneConfigs: network.PrivateDnsZoneConfigArray{
			&network.PrivateDnsZoneConfigArgs{
				Name:             pulumi.String("redis"),
				PrivateDnsZoneId: infra.DNS.RedisPrivateZone.ID().ToStringOutput(),
			},
		},
	}, append(opts, pulumi.Parent(pe))...)
	if err != nil {
		return fmt.Errorf("creating Redis private DNS zone group: %w", err)
	}
	return nil
}
