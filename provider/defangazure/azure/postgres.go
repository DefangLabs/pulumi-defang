package azure

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/dbforpostgresql/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type postgresResult struct {
	Server *dbforpostgresql.Server
}

// CreatePostgresFlexible creates an Azure Database for PostgreSQL Flexible Server.
func CreatePostgresFlexible(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	serviceName string,
	svc compose.ServiceConfig,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*postgresResult, error) {
	pg := svc.ResolvePostgres(ctx, configProvider)
	if pg == nil {
		return nil, fmt.Errorf("postgres config is nil")
	}

	// Backup config
	backupRetention := BackupRetentionDays.Get(ctx)
	geoBackup := dbforpostgresql.GeoRedundantBackupEnumDisabled
	if GeoRedundantBackup.Get(ctx) {
		geoBackup = dbforpostgresql.GeoRedundantBackupEnumEnabled
	}

	// Admin credentials
	username := pg.Username
	if username == pulumi.String("") {
		username = pulumi.String("postgres")
	}

	serverArgs := &dbforpostgresql.ServerArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		Version:           pg.Version,
		Sku: &dbforpostgresql.SkuArgs{
			Name: pulumi.String(SkuName.Get(ctx)),
			Tier: pulumi.String(string(dbforpostgresql.SkuTierBurstable)),
		},
		Storage: &dbforpostgresql.StorageArgs{
			StorageSizeGB: pulumi.Int(StorageSizeGB.Get(ctx)),
		},
		Backup: &dbforpostgresql.BackupTypeArgs{
			BackupRetentionDays: pulumi.Int(backupRetention),
			GeoRedundantBackup:  pulumi.String(string(geoBackup)),
		},
		AdministratorLogin:         pg.Username,
		AdministratorLoginPassword: pg.Password,
	}
	if HighAvailability.Get(ctx) {
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
			ResourceGroupName: infra.ResourceGroup.Name,
			ServerName:        server.Name,
			DatabaseName:      pg.DBName,
		}, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating PostgreSQL database: %w", err)
		}
	}

	// Allow Azure services to connect (firewall rule for 0.0.0.0)
	_, err = dbforpostgresql.NewFirewallRule(ctx, serviceName+"-allow-azure", &dbforpostgresql.FirewallRuleArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		ServerName:        server.Name,
		StartIpAddress:    pulumi.String("0.0.0.0"),
		EndIpAddress:      pulumi.String("0.0.0.0"),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating PostgreSQL firewall rule: %w", err)
	}

	return &postgresResult{Server: server}, nil
}
