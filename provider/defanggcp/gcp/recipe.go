package gcp

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// Recipe holds GCP-specific tuning knobs with IS_DEV defaults.
type Recipe struct {
	// General
	DeletionProtection bool `json:"deletion-protection"`

	// Cloud SQL
	AvailabilityType    string `json:"availability-type"`
	BackupEnabled       bool   `json:"backup-enabled"`
	PointInTimeRecovery bool   `json:"point-in-time-recovery"`
	SslMode             string `json:"ssl-mode"`
	AllowBurstable      bool   `json:"allow-burstable"`

	// Cloud Run
	Ingress     string `json:"ingress"`
	LaunchStage string `json:"launch-stage"`
	MaxReplicas int    `json:"max-replicas"` // 0 means use deploy.replicas
}

// DefaultRecipe returns a Recipe initialized with IS_DEV defaults.
func DefaultRecipe() Recipe {
	return Recipe{
		DeletionProtection:  false,
		AvailabilityType:    "ZONAL",
		BackupEnabled:       false,
		PointInTimeRecovery: false,
		SslMode:             "ALLOW_UNENCRYPTED_AND_ENCRYPTED",
		AllowBurstable:      true,
		Ingress:             "INGRESS_TRAFFIC_ALL",
		LaunchStage:         "BETA",
		MaxReplicas:         0,
	}
}

// LoadRecipe reads defang:gcp-recipe from stack config, falling back to IS_DEV defaults.
func LoadRecipe(ctx *pulumi.Context) Recipe {
	r := DefaultRecipe()
	cfg := config.New(ctx, "defang")
	_ = cfg.TryObject("gcp-recipe", &r)
	return r
}
