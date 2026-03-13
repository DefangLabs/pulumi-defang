package gcp

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/sql"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type cloudSQLResult struct {
	instance *sql.DatabaseInstance
}

// gcpPostgresVersion maps a major version number to the GCP database version string.
func gcpPostgresVersion(version int) string {
	switch version {
	case 14:
		return "POSTGRES_14"
	case 15:
		return "POSTGRES_15"
	case 16:
		return "POSTGRES_16"
	case 17:
		return "POSTGRES_17"
	default:
		return "POSTGRES_17"
	}
}

// cloudSQLTier maps CPU/memory to a Cloud SQL tier.
func cloudSQLTier(cpus float64, memMiB int) string {
	if cpus <= 1 && memMiB <= 600 {
		return "db-f1-micro"
	}
	if cpus <= 1 && memMiB <= 1700 {
		return "db-g1-small"
	}

	// Custom tier
	cpu := int(cpus)
	if cpu <= 1 {
		cpu = 1
	} else if cpu > 96 {
		cpu = 96
	} else {
		cpu = (cpu + 1) / 2 * 2 // Even numbers only above 1
	}

	mem := memMiB
	if mem < 3840 {
		mem = 3840
	} else if mem > 98304 {
		mem = 98304
	} else {
		mem = (mem + 255) / 256 * 256 // Round up to nearest 256 MiB
	}

	return fmt.Sprintf("db-custom-%d-%d", cpu, mem)
}

// createCloudSQL creates a managed Cloud SQL Postgres instance.
func createCloudSQL(
	ctx *pulumi.Context,
	serviceName string,
	svc common.ServiceConfig,
	recipe Recipe,
	opts ...pulumi.ResourceOption,
) (*cloudSQLResult, error) {
	pg := svc.Postgres
	if pg == nil {
		return nil, fmt.Errorf("postgres config is nil")
	}

	tier := cloudSQLTier(svc.GetCPUs(), svc.GetMemoryMiB())

	// Enforce burstable restriction from recipe
	if !recipe.AllowBurstable && (tier == "db-f1-micro" || tier == "db-g1-small") {
		tier = "db-custom-1-3840"
	}

	// Configure backups from recipe
	var backupConf *sql.DatabaseInstanceSettingsBackupConfigurationArgs
	if recipe.BackupEnabled {
		backupConf = &sql.DatabaseInstanceSettingsBackupConfigurationArgs{
			Enabled:                    pulumi.Bool(true),
			PointInTimeRecoveryEnabled: pulumi.Bool(recipe.PointInTimeRecovery),
			BackupRetentionSettings: &sql.DatabaseInstanceSettingsBackupConfigurationBackupRetentionSettingsArgs{
				RetainedBackups: pulumi.Int(30),
			},
			StartTime:                  pulumi.String("04:00"),
			TransactionLogRetentionDays: pulumi.Int(7),
		}
	}

	instance, err := sql.NewDatabaseInstance(ctx, serviceName, &sql.DatabaseInstanceArgs{
		DatabaseVersion: pulumi.String(gcpPostgresVersion(pg.Version)),
		Settings: &sql.DatabaseInstanceSettingsArgs{
			Tier:                pulumi.String(tier),
			Edition:             pulumi.String("ENTERPRISE"),
			AvailabilityType:    pulumi.String(recipe.AvailabilityType),
			BackupConfiguration: backupConf,
			IpConfiguration: &sql.DatabaseInstanceSettingsIpConfigurationArgs{
				Ipv4Enabled: pulumi.Bool(true),
				SslMode:     pulumi.StringPtr(recipe.SslMode),
			},
		},
		DeletionProtection: pulumi.Bool(recipe.DeletionProtection),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Cloud SQL instance: %w", err)
	}

	// Create user
	if pg.Password != "" {
		username := pg.Username
		if username == "" {
			username = "postgres"
		}
		_, err := sql.NewUser(ctx, serviceName+"-user", &sql.UserArgs{
			Name:           pulumi.String(username),
			Instance:       instance.Name,
			Password:       pulumi.String(pg.Password),
			Type:           pulumi.String("BUILT_IN"),
			DeletionPolicy: pulumi.String("ABANDON"),
		}, append(opts, pulumi.RetainOnDelete(true))...)
		if err != nil {
			return nil, fmt.Errorf("creating Cloud SQL user: %w", err)
		}
	}

	// Create database if non-default
	if pg.DBName != "" && pg.DBName != "postgres" {
		_, err := sql.NewDatabase(ctx, serviceName+"-db", &sql.DatabaseArgs{
			Name:           pulumi.String(pg.DBName),
			Instance:       instance.Name,
			DeletionPolicy: pulumi.String("ABANDON"),
		}, append(opts, pulumi.RetainOnDelete(true))...)
		if err != nil {
			return nil, fmt.Errorf("creating Cloud SQL database: %w", err)
		}
	}

	return &cloudSQLResult{
		instance: instance,
	}, nil
}
