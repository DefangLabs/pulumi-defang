package azure

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/pulumi/pulumi-azure-native-sdk/dbforpostgresql/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type postgresResult struct {
	server *dbforpostgresql.Server
}

// azurePostgresVersion maps a major version number to the Azure server version string.
func azurePostgresVersion(version int) string {
	switch version {
	case 13:
		return "13"
	case 14:
		return "14"
	case 15:
		return "15"
	case 16:
		return "16"
	case 17:
		return "17"
	default:
		return "16"
	}
}

// createPostgresFlexible creates an Azure Database for PostgreSQL Flexible Server.
func createPostgresFlexible(
	ctx *pulumi.Context,
	configProvider shared.ConfigProvider,
	serviceName string,
	svc shared.ServiceInput,
	infra *sharedInfra,
	recipe Recipe,
	opts ...pulumi.ResourceOption,
) (*postgresResult, error) {
	pg := svc.ResolvePostgres(ctx, configProvider)
	if pg == nil {
		return nil, fmt.Errorf("postgres config is nil")
	}

	// Backup config
	backupRetention := recipe.BackupRetentionDays
	geoBackup := dbforpostgresql.GeoRedundantBackupEnumDisabled
	if recipe.GeoRedundantBackup {
		geoBackup = dbforpostgresql.GeoRedundantBackupEnumEnabled
	}

	// Admin credentials
	username := pg.Username
	if username == pulumi.String("") {
		username = pulumi.String("postgres")
	}

	serverArgs := &dbforpostgresql.ServerArgs{
		ResourceGroupName: infra.resourceGroup.Name,
		Version:           pulumi.String(azurePostgresVersion(pg.Version)),
		Sku: &dbforpostgresql.SkuArgs{
			Name: pulumi.String(recipe.SkuName),
			Tier: pulumi.String(string(dbforpostgresql.SkuTierBurstable)),
		},
		Storage: &dbforpostgresql.StorageArgs{
			StorageSizeGB: pulumi.Int(recipe.StorageSizeGB),
		},
		Backup: &dbforpostgresql.BackupTypeArgs{
			BackupRetentionDays: pulumi.Int(backupRetention),
			GeoRedundantBackup:  pulumi.String(string(geoBackup)),
		},
		AdministratorLogin:         pg.Username,
		AdministratorLoginPassword: pg.Password,
	}
	if recipe.HighAvailability {
		serverArgs.HighAvailability = &dbforpostgresql.HighAvailabilityArgs{
			Mode: pulumi.String(string(dbforpostgresql.HighAvailabilityModeZoneRedundant)),
		}
	}

	server, err := dbforpostgresql.NewServer(ctx, serviceName, serverArgs, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating PostgreSQL Flexible Server: %w", err)
	}

	// Create database if non-default
	if pg.DBName != pulumi.String("") && pg.DBName != pulumi.String("postgres") {
		_, err := dbforpostgresql.NewDatabase(ctx, serviceName+"-db", &dbforpostgresql.DatabaseArgs{
			ResourceGroupName: infra.resourceGroup.Name,
			ServerName:        server.Name,
			DatabaseName:      pg.DBName,
		}, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating PostgreSQL database: %w", err)
		}
	}

	// Allow Azure services to connect (firewall rule for 0.0.0.0)
	_, err = dbforpostgresql.NewFirewallRule(ctx, serviceName+"-allow-azure", &dbforpostgresql.FirewallRuleArgs{
		ResourceGroupName: infra.resourceGroup.Name,
		ServerName:        server.Name,
		StartIpAddress:    pulumi.String("0.0.0.0"),
		EndIpAddress:      pulumi.String("0.0.0.0"),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating PostgreSQL firewall rule: %w", err)
	}

	return &postgresResult{server: server}, nil
}
