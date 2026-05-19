package azure

import "github.com/DefangLabs/pulumi-defang/provider/common"

// Azure recipe config accessors. Each reads from defang:<key> in stack config.
var (
	BackupRetentionDays   = common.Int("backup-retention-days", 7)
	DeletionProtection    = common.DeletionProtection
	GeoRedundantBackup    = common.Bool("geo-redundant-backup", false)
	HighAvailability      = common.Bool("high-availability", false)
	LogWorkspaceSku       = common.String("log-workspace-sku", "PerGB2018")
	// LogWorkspaceDailyQuotaGb caps daily ingestion in GB. 0 = no cap.
	// Default 1 GB/day = ~30 GB/mo, ~$70 ceiling on PerGB2018 (~$2.30/GB).
	// Chatty workloads (AI agents) override upward; an unbounded default
	// has been seen to bill >$700/mo against a single workspace.
	LogWorkspaceDailyQuotaGb = common.Int("log-workspace-daily-quota-gb", 1)	
	MaxReplicas              = common.Int("max-replicas", 0)
	PostgresTier             = common.String("postgres-tier", "burstable")
	PostgresStorageSizeGB    = common.Int("storage-size-gb", 32)
	RegistrySku              = common.String("registry-sku", "Basic")
)
