package azure

import "github.com/DefangLabs/pulumi-defang/provider/common"

// Azure recipe config accessors. Each reads from defang:<key> in stack config.
var (
	BackupRetentionDays = common.Int("backup-retention-days", 7)
	DeletionProtection  = common.DeletionProtection
	GeoRedundantBackup  = common.Bool("geo-redundant-backup", false)
	HighAvailability    = common.Bool("high-availability", false)
	MaxReplicas         = common.Int("max-replicas", 0)
	SkuName             = common.String("sku-name", "Standard_B1ms")
	StorageSizeGB       = common.Int("storage-size-gb", 32)
)
