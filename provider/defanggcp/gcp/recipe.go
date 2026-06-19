package gcp

import "github.com/DefangLabs/pulumi-defang/provider/common"

// GCP recipe config accessors. Cloud-specific keys read from defang-gcp:<key>.
var recipe = common.NewRecipe("defang-gcp")

var (
	AllowBurstable            = recipe.Bool("allow-burstable", true)
	AvailabilityType          = recipe.String("availability-type", "ZONAL")
	BackupEnabled             = recipe.Bool("backup-enabled", false)
	DeletionProtection        = recipe.Bool("deletion-protection", false)
	Ingress                   = recipe.String("ingress", "INGRESS_TRAFFIC_ALL")
	LaunchStage               = recipe.String("launch-stage", "")
	MaxReplicas               = recipe.Int("max-replicas", 0)
	PointInTimeRecovery       = recipe.Bool("point-in-time-recovery", false)
	SslMode                   = recipe.String("ssl-mode", "ALLOW_UNENCRYPTED_AND_ENCRYPTED")
	TransitEncryptionDisabled = recipe.Bool("transit-encryption-disabled", false)
)
