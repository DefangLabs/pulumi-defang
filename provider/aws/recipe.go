package aws

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// Recipe holds AWS-specific tuning knobs with IS_DEV defaults.
type Recipe struct {
	LogRetentionDays        int    `json:"log-retention-days"`
	DeletionProtection      bool   `json:"deletion-protection"`
	StorageEncrypted        bool   `json:"storage-encrypted"`
	BackupRetentionDays     int    `json:"backup-retention-days"`
	FargateCapacityProvider string `json:"fargate-capacity-provider"`
	MinHealthyPercent       int    `json:"min-healthy-percent"`
	DeregistrationDelay     int    `json:"deregistration-delay"`
	HealthCheckInterval     int    `json:"health-check-interval"`
	HealthCheckThreshold    int    `json:"health-check-threshold"`
	AllowBurstable          bool   `json:"allow-burstable"`
	RDSNodeType             string `json:"rds-node-type"` // "burstable", "general", "memory-optimized"
}

// DefaultRecipe returns a Recipe initialized with IS_DEV defaults.
func DefaultRecipe() Recipe {
	return Recipe{
		LogRetentionDays:        1,
		DeletionProtection:      false,
		StorageEncrypted:        false,
		BackupRetentionDays:     0,
		FargateCapacityProvider: "FARGATE_SPOT",
		MinHealthyPercent:       0,
		DeregistrationDelay:     0,
		HealthCheckInterval:     5,
		HealthCheckThreshold:    2,
		AllowBurstable:          true,
		RDSNodeType:             "burstable",
	}
}

// LoadRecipe reads defang:aws-recipe from stack config, falling back to IS_DEV defaults.
func LoadRecipe(ctx *pulumi.Context) Recipe {
	r := DefaultRecipe()
	cfg := config.New(ctx, "defang")
	_ = cfg.TryObject("aws-recipe", &r)
	return r
}
