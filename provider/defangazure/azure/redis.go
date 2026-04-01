package azure

import (
	"errors"
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	redis "github.com/pulumi/pulumi-azure-native-sdk/redisenterprise/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var ErrRedisConfigNil = errors.New("redis config is nil")

// RedisResult holds the Azure Managed Redis cluster and database resources.
type RedisResult struct {
	Cluster  *redis.RedisEnterprise
	Database *redis.Database
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

	location := Location(ctx)
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
		Location:          pulumi.StringPtr(location),
		MinimumTlsVersion: pulumi.String("1.2"),
		HighAvailability:  pulumi.String(haValue),
		Sku: redis.SkuArgs{
			Name: pulumi.String(skuName),
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Azure Managed Redis cluster: %w", err)
	}

	db, err := redis.NewDatabase(ctx, serviceName+"-db", &redis.DatabaseArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		ClusterName:       cluster.Name,
		ClientProtocol:    pulumi.String("Encrypted"),
		Port:              pulumi.Int(10000),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Azure Managed Redis database: %w", err)
	}

	return &RedisResult{Cluster: cluster, Database: db}, nil
}
