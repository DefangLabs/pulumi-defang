package common

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// AWSRecipe holds AWS-specific tuning knobs with IS_DEV defaults.
type AWSRecipe struct {
	LogRetentionDays         int    `json:"log-retention-days"`
	DeletionProtection       bool   `json:"deletion-protection"`
	StorageEncrypted         bool   `json:"storage-encrypted"`
	BackupRetentionDays      int    `json:"backup-retention-days"`
	FargateCapacityProvider  string `json:"fargate-capacity-provider"`
	MinHealthyPercent        int    `json:"min-healthy-percent"`
	DeregistrationDelay      int    `json:"deregistration-delay"`
	HealthCheckInterval      int    `json:"health-check-interval"`
	HealthCheckThreshold     int    `json:"health-check-threshold"`
}

// GCPRecipe holds GCP-specific tuning knobs with IS_DEV defaults.
type GCPRecipe struct {
	DeletionProtection bool `json:"deletion-protection"`
}

// DefaultAWSRecipe returns an AWSRecipe initialized with IS_DEV defaults.
func DefaultAWSRecipe() AWSRecipe {
	return AWSRecipe{
		LogRetentionDays:         1,
		DeletionProtection:       false,
		StorageEncrypted:         false,
		BackupRetentionDays:      0,
		FargateCapacityProvider:  "FARGATE_SPOT",
		MinHealthyPercent:        0,
		DeregistrationDelay:      0,
		HealthCheckInterval:      5,
		HealthCheckThreshold:     2,
	}
}

// DefaultGCPRecipe returns a GCPRecipe initialized with IS_DEV defaults.
func DefaultGCPRecipe() GCPRecipe {
	return GCPRecipe{
		DeletionProtection: false,
	}
}

// LoadAWSRecipe reads defang:aws-recipe from stack config, falling back to IS_DEV defaults.
func LoadAWSRecipe(ctx *pulumi.Context) AWSRecipe {
	r := DefaultAWSRecipe()
	cfg := config.New(ctx, "defang")
	_ = cfg.TryObject("aws-recipe", &r)
	return r
}

// LoadGCPRecipe reads defang:gcp-recipe from stack config, falling back to IS_DEV defaults.
func LoadGCPRecipe(ctx *pulumi.Context) GCPRecipe {
	r := DefaultGCPRecipe()
	cfg := config.New(ctx, "defang")
	_ = cfg.TryObject("gcp-recipe", &r)
	return r
}
