package azure

import "github.com/DefangLabs/pulumi-defang/provider/common"

// Azure recipe config accessors. Each reads from defang:<key> in stack config.
var (
	BackupRetentionDays   = common.Int("backup-retention-days", 7)
	DeletionProtection    = common.DeletionProtection
	GeoRedundantBackup    = common.Bool("geo-redundant-backup", false)
	HighAvailability      = common.Bool("high-availability", false)
	LogWorkspaceSku       = common.String("log-workspace-sku", "PerGB2018")
	MaxReplicas           = common.Int("max-replicas", 0)
	PostgresTier          = common.String("postgres-tier", "burstable")
	PostgresStorageSizeGB = common.Int("storage-size-gb", 32)
	RegistrySku           = common.String("registry-sku", "Basic")
)
