package azure

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// Recipe holds Azure-specific tuning knobs with IS_DEV defaults.
type Recipe struct {
	// General
	DeletionProtection bool `json:"deletion-protection"`

	// Container Apps
	MaxReplicas int `json:"max-replicas"` // 0 means use deploy.replicas

	// Postgres Flexible Server
	SkuName             string `json:"sku-name"`               // e.g. "B_Standard_B1ms" (burstable)
	BackupRetentionDays int    `json:"backup-retention-days"`
	GeoRedundantBackup  bool   `json:"geo-redundant-backup"`
	HighAvailability    bool   `json:"high-availability"`
	StorageSizeGB       int    `json:"storage-size-gb"`
}

// DefaultRecipe returns a Recipe initialized with IS_DEV defaults.
func DefaultRecipe() Recipe {
	return Recipe{
		DeletionProtection:  false,
		MaxReplicas:         0,
		SkuName:             "B_Standard_B1ms",
		BackupRetentionDays: 7,
		GeoRedundantBackup:  false,
		HighAvailability:    false,
		StorageSizeGB:       32,
	}
}

// LoadRecipe reads defang:azure-recipe from stack config, falling back to IS_DEV defaults.
func LoadRecipe(ctx *pulumi.Context) Recipe {
	r := DefaultRecipe()
	cfg := config.New(ctx, "defang")
	_ = cfg.TryObject("azure-recipe", &r)
	return r
}
