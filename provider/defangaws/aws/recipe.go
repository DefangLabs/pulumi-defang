package aws

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// Recipe holds AWS-specific tuning knobs with IS_DEV defaults.
type Recipe struct {
	AllowBurstable          bool   `json:"allow-burstable"`
	AllowOverwriteRecords   bool   `json:"allow-overwrite-records"`
	BackupRetentionDays     int    `json:"backup-retention-days"`
	BackupWindow            string `json:"backup-window"` // UTC window for automated backups, e.g. "04:00-05:00"
	DeletionProtection      bool   `json:"deletion-protection"`
	DeregistrationDelay     int    `json:"deregistration-delay"`
	FargateCapacityProvider string `json:"fargate-capacity-provider"`
	ForceDestroyHostedzone  bool   `json:"force-destroy-hostedzone"`
	HealthCheckInterval     int    `json:"health-check-interval"`
	HealthCheckThreshold    int    `json:"health-check-threshold"`
	LogRetentionDays        int    `json:"log-retention-days"`
	MinHealthyPercent       int    `json:"min-healthy-percent"`
	RDSNodeType             string `json:"rds-node-type"` // "burstable", "general", "memory-optimized"
	RetainDnsOnDelete       bool   `json:"retain-dns-on-delete"`
	StorageEncrypted        bool   `json:"storage-encrypted"`
}

// DefaultRecipe returns a Recipe initialized with IS_DEV defaults.
func DefaultRecipe() Recipe {
	return Recipe{
		LogRetentionDays:        1,
		DeletionProtection:      false,
		StorageEncrypted:        false,
		BackupRetentionDays:     0,
		BackupWindow:            "04:00-05:00",
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
