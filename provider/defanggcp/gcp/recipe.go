package gcp

import "github.com/DefangLabs/pulumi-defang/provider/common"

// GCP recipe config accessors. Each reads from defang:<key> in stack config.
var (
	AllowBurstable      = common.Bool("allow-burstable", true)
	AvailabilityType    = common.String("availability-type", "ZONAL")
	BackupEnabled       = common.Bool("backup-enabled", false)
	DeletionProtection  = common.DeletionProtection
	Ingress             = common.String("ingress", "INGRESS_TRAFFIC_ALL")
	LaunchStage         = common.String("launch-stage", "BETA")
	MaxReplicas         = common.Int("max-replicas", 0)
	PointInTimeRecovery = common.Bool("point-in-time-recovery", false)
	SslMode                   = common.String("ssl-mode", "ALLOW_UNENCRYPTED_AND_ENCRYPTED")
	TransitEncryptionDisabled = common.Bool("transit-encryption-disabled", false)
)
