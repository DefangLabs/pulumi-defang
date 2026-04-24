package gcp

import (
	"errors"
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/sql"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var ErrPostgresConfigNil = errors.New("postgres config is nil")

type CloudSQLResult struct {
	Instance *sql.DatabaseInstance
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
	switch {
	case cpu <= 1:
		cpu = 1
	case cpu > 96:
		cpu = 96
	default:
		cpu = (cpu + 1) / 2 * 2 // Even numbers only above 1
	}

	mem := memMiB
	switch {
	case mem < 3840:
		mem = 3840
	case mem > 98304:
		mem = 98304
	default:
		mem = (mem + 255) / 256 * 256 // Round up to nearest 256 MiB
	}

	return fmt.Sprintf("db-custom-%d-%d", cpu, mem)
}

// sqlBackupConfig returns the backup configuration from recipe settings, or nil if backups are disabled.
func sqlBackupConfig(ctx *pulumi.Context) *sql.DatabaseInstanceSettingsBackupConfigurationArgs {
	if !BackupEnabled.Get(ctx) {
		return nil
	}
	return &sql.DatabaseInstanceSettingsBackupConfigurationArgs{
		Enabled:                    pulumi.Bool(true),
		PointInTimeRecoveryEnabled: pulumi.Bool(PointInTimeRecovery.Get(ctx)),
		BackupRetentionSettings: &sql.DatabaseInstanceSettingsBackupConfigurationBackupRetentionSettingsArgs{
			RetainedBackups: pulumi.Int(30),
		},
		StartTime:                   pulumi.String("04:00"),
		TransactionLogRetentionDays: pulumi.Int(7),
	}
}

// CreateCloudSQL creates a managed Cloud SQL Postgres instance.
func CreateCloudSQL(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	serviceName string,
	svc compose.ServiceConfig,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*CloudSQLResult, error) {
	pg := svc.ResolvePostgres(ctx, configProvider)
	if pg == nil {
		return nil, ErrPostgresConfigNil
	}

	tier := cloudSQLTier(svc.GetCPUs(), svc.GetMemoryMiB())

	// Enforce burstable restriction from recipe
	if !AllowBurstable.Get(ctx) && (tier == "db-f1-micro" || tier == "db-g1-small") {
		tier = "db-custom-1-3840"
	}

	backupConf := sqlBackupConfig(ctx)

	databaseVersion := pg.Version.ToStringPtrOutput().ApplyT(func(version *string) string {
		if version == nil {
			return gcpPostgresVersion(0) // default to latest
		}
		v := compose.GetPostgresVersion(*version)
		return gcpPostgresVersion(v)
	}).(pulumi.StringOutput)

	instanceOpts := opts
	if infra != nil && infra.ServiceConnection != nil {
		instanceOpts = append([]pulumi.ResourceOption{
			pulumi.DependsOn([]pulumi.Resource{infra.ServiceConnection}),
		}, opts...)
	}

	var regionInput pulumi.StringPtrInput
	if infra != nil && infra.Region != "" {
		regionInput = pulumi.StringPtr(infra.Region)
	}

	_, onlyPrivateIP := svc.Networks["private"]
	var privateNetwork pulumi.StringPtrInput
	if infra != nil && infra.ServiceConnection != nil {
		privateNetwork = infra.VpcId.ApplyT(func(v string) *string { return &v }).(pulumi.StringPtrOutput)
	}

	instance, err := sql.NewDatabaseInstance(ctx, serviceName, &sql.DatabaseInstanceArgs{
		DatabaseVersion: databaseVersion,
		Region:          regionInput,
		Settings: &sql.DatabaseInstanceSettingsArgs{
			Tier:                pulumi.String(tier),
			Edition:             pulumi.String("ENTERPRISE"),
			AvailabilityType:    pulumi.String(AvailabilityType.Get(ctx)),
			BackupConfiguration: backupConf,
			IpConfiguration: &sql.DatabaseInstanceSettingsIpConfigurationArgs{
				Ipv4Enabled:    pulumi.Bool(!onlyPrivateIP),
				PrivateNetwork: privateNetwork,
				SslMode:        pulumi.StringPtr(SslMode.Get(ctx)),
			},
		},
		DeletionProtection: pulumi.Bool(DeletionProtection.Get(ctx)),
	}, instanceOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating Cloud SQL instance: %w", err)
	}

	// Create user only if a password is explicitly provided as a literal.
	// A nil *string (YAML "POSTGRES_PASSWORD:" with no value) means "resolve
	// at runtime from config" — at this point we can't tell if a password
	// exists, so skip custom user creation and let the default path handle it.
	if pw := svc.Environment["POSTGRES_PASSWORD"]; pw != nil && *pw != "" {
		_, err := sql.NewUser(ctx, serviceName+"-user", &sql.UserArgs{
			Name:           pg.Username,
			Instance:       instance.Name,
			Password:       pg.Password,
			Type:           pulumi.String("BUILT_IN"),
			DeletionPolicy: pulumi.String("ABANDON"),
		}, append(opts, pulumi.RetainOnDelete(true))...)
		if err != nil {
			return nil, fmt.Errorf("creating Cloud SQL user: %w", err)
		}
	}

	// Create database only if explicitly set to a non-default literal name.
	// Same rationale as POSTGRES_PASSWORD above: nil (*string zero) is "resolve
	// at runtime", which we treat as "no custom DB requested" for scheduling.
	if rawDB := svc.Environment["POSTGRES_DB"]; rawDB != nil && *rawDB != "" && *rawDB != compose.DEFAULT_POSTGRES_DB {
		_, err := sql.NewDatabase(ctx, serviceName+"-db", &sql.DatabaseArgs{
			Name:           pg.DBName,
			Instance:       instance.Name,
			DeletionPolicy: pulumi.String("ABANDON"),
		}, append(opts, pulumi.RetainOnDelete(true))...)
		if err != nil {
			return nil, fmt.Errorf("creating Cloud SQL database: %w", err)
		}
	}

	return &CloudSQLResult{
		Instance: instance,
	}, nil
}
