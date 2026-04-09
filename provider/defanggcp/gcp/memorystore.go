package gcp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/redis"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type MemorystoreResult struct {
	Instance *redis.Instance
}

// getRedisVersion maps an image tag to a GCP Memorystore Redis version string.
// Supported versions: https://cloud.google.com/memorystore/docs/redis/supported-versions
func getRedisVersion(tag string) *string {
	// Tag may be e.g. "7", "7.2", "7.2.5", "6.2.6"
	parts := strings.SplitN(tag, ".", 3)
	if len(parts) == 0 || parts[0] == "" {
		return nil
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil
	}
	var ver string
	if major == 6 {
		ver = "REDIS_6_X"
	} else {
		minor := 0
		if len(parts) >= 2 {
			minor, _ = strconv.Atoi(parts[1])
		}
		ver = fmt.Sprintf("REDIS_%d_%d", major, minor)
	}
	return &ver
}

// getRedisMemoryGiB returns the memory reservation in GiB, defaulting to 1.
func getRedisMemoryGiB(svc compose.ServiceConfig) int {
	memGiB := svc.GetMemoryMiB() / 1024
	if memGiB < 1 {
		return 1
	}
	return memGiB
}

// CreateMemoryStore creates a managed GCP Memorystore Redis instance.
func CreateMemoryStore(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	infra *GlobalConfig,
	opts ...pulumi.ResourceOption,
) (*MemorystoreResult, error) {
	var transitEncryptionMode pulumi.StringPtrInput
	if TransitEncryptionDisabled.Get(ctx) {
		transitEncryptionMode = pulumi.StringPtr("DISABLED")
	}

	instanceOpts := opts
	if infra != nil && infra.ServiceConnection != nil {
		instanceOpts = append([]pulumi.ResourceOption{
			pulumi.DependsOn([]pulumi.Resource{infra.ServiceConnection}),
		}, opts...)
	}

	var authorizedNetwork pulumi.StringPtrInput
	if infra != nil {
		authorizedNetwork = infra.VpcId.ApplyT(func(v string) *string { return &v }).(pulumi.StringPtrOutput)
	}

	var region pulumi.StringPtrInput
	if infra != nil && infra.Region != "" {
		region = pulumi.StringPtr(infra.Region)
	}

	imageTag := ""
	if svc.Image != nil {
		imageTag = compose.ParseImageTag(*svc.Image)
	}

	instance, err := redis.NewInstance(ctx, serviceName, &redis.InstanceArgs{
		AuthorizedNetwork:     authorizedNetwork,
		ConnectMode:           pulumi.String("PRIVATE_SERVICE_ACCESS"),
		DisplayName:           pulumi.String(serviceName),
		MemorySizeGb:          pulumi.Int(getRedisMemoryGiB(svc)),
		RedisVersion:          pulumi.StringPtrFromPtr(getRedisVersion(imageTag)),
		Tier:                  pulumi.String("STANDARD_HA"),
		TransitEncryptionMode: transitEncryptionMode,
		Region:                region,
	}, instanceOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating Memorystore instance: %w", err)
	}

	return &MemorystoreResult{Instance: instance}, nil
}
