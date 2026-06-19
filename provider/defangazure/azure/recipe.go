package azure

import "github.com/DefangLabs/pulumi-defang/provider/common"

// Azure recipe config accessors. Cloud-specific keys read from defang-azure:<key>.
var recipe = common.NewRecipe("defang-azure")

var (
	BackupRetentionDays = recipe.Int("backup-retention-days", 7)
	// Azure declares no deletion-protection key: nothing in the Azure provider
	// consumes it yet. Add recipe.Bool("deletion-protection", false) here when it does.
	GeoRedundantBackup = recipe.Bool("geo-redundant-backup", false)
	HighAvailability   = recipe.Bool("high-availability", false)
	LogRetentionDays   = recipe.Int("log-retention-days", 1)
	LogWorkspaceSku    = recipe.String("log-workspace-sku", "PerGB2018")
	// LogWorkspaceDailyQuotaGb caps daily ingestion in GB. 0 = no cap.
	// Default 1 GB/day = ~30 GB/mo, ~$70 ceiling on PerGB2018 (~$2.30/GB).
	// Chatty workloads (AI agents) override upward; an unbounded default
	// has been seen to bill >$700/mo against a single workspace.
	LogWorkspaceDailyQuotaGb = recipe.Int("log-workspace-daily-quota-gb", 1)
	MaxReplicas              = recipe.Int("max-replicas", 0)
	PostgresTier             = recipe.String("postgres-tier", "burstable")
	PostgresStorageSizeGB    = recipe.Int("storage-size-gb", 32)
	RegistrySku              = recipe.String("registry-sku", "Basic")
)
