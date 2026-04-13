package azure

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/dbforpostgresql/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var ErrPostgresConfigNil = errors.New("postgres config is nil")

// sanitizePostgresName strips characters invalid in Azure PostgreSQL Flexible Server names
// (only lowercase letters, digits, and hyphens are allowed; 3-63 chars).
// It is applied to Pulumi logical names so that auto-naming produces a valid physical name.
func sanitizePostgresName(name string) string {
	name = strings.ToLower(name)
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	s := strings.Trim(b.String(), "-")
	if len(s) > 50 {
		s = strings.Trim(s[:50], "-")
	}
	return s
}

type postgresResult struct {
	Server *dbforpostgresql.Server
}

// buildPostgresServerArgs constructs the ServerArgs for an Azure PostgreSQL Flexible Server.
func buildPostgresServerArgs(
	pg *compose.PostgresConfigArgs,
	sanitized string,
	infra *SharedInfra,
	ctx *pulumi.Context,
) *dbforpostgresql.ServerArgs {
	// Backup config
	backupRetention := BackupRetentionDays.Get(ctx)
	geoBackup := dbforpostgresql.GeoRedundantBackupDisabled
	if GeoRedundantBackup.Get(ctx) {
		geoBackup = dbforpostgresql.GeoRedundantBackupEnabled
	}

	// Build a globally-unique server name from the resource group's random suffix.
	serverName := infra.ResourceGroup.Name.ApplyT(func(rgName string) string {
		suffix := rgName
		if idx := strings.LastIndexByte(rgName, '-'); idx >= 0 {
			suffix = rgName[idx+1:] // e.g. "b89e321"
		}
		n := sanitized + "-" + suffix
		if len(n) > 63 {
			n = strings.Trim(n[:63], "-")
		}
		return n
	}).(pulumi.StringOutput)

	serverArgs := &dbforpostgresql.ServerArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		Location:          infra.ResourceGroup.Location,
		ServerName:        serverName.ToStringPtrOutput(),
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
			Mode: pulumi.String(string(dbforpostgresql.PostgreSqlFlexibleServerHighAvailabilityModeZoneRedundant)),
		}
	}

	// VNet integration: attach to the delegated subnet and private DNS zone when networking is configured.
	if infra.Networking != nil && infra.DNS != nil {
		serverArgs.Network = &dbforpostgresql.NetworkArgs{
			DelegatedSubnetResourceId:   infra.Networking.PostgresSubnet.ID().ToStringOutput().ToStringPtrOutput(),
			PrivateDnsZoneArmResourceId: infra.DNS.PostgresZoneID.ToStringPtrOutput(),
		}
	}

	return serverArgs
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
		return nil, ErrPostgresConfigNil
	}

	// Admin credentials
	if pg.Username == pulumi.String("") {
		pg.Username = pulumi.String("postgres")
	}

	// Sanitize the logical name: Azure PostgreSQL server names allow only lowercase
	// letters, digits, and hyphens. StackDir-style names contain slashes etc.
	sanitized := sanitizePostgresName(serviceName)

	serverArgs := buildPostgresServerArgs(pg, sanitized, infra, ctx)

	server, err := dbforpostgresql.NewServer(ctx, sanitized, serverArgs, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating PostgreSQL Flexible Server: %w", err)
	}

	// Allowlist the pgvector extension. Azure Postgres Flexible Server blocks CREATE EXTENSION
	// unless the extension is listed in the azure.extensions server parameter first.
	_, err = dbforpostgresql.NewConfiguration(ctx, sanitized+"-pgvector", &dbforpostgresql.ConfigurationArgs{
		ResourceGroupName:  infra.ResourceGroup.Name,
		ServerName:         server.Name,
		ConfigurationName:  pulumi.String("azure.extensions"),
		Value:              pulumi.String("VECTOR"),
		Source:             pulumi.String("user-override"),
	}, append(opts, pulumi.Parent(server))...)
	if err != nil {
		return nil, fmt.Errorf("enabling pgvector extension: %w", err)
	}

	// Create database if non-default; "postgres" already exists on every new Flexible Server.
	if pg.DBNameStr != "" && pg.DBNameStr != compose.DEFAULT_POSTGRES_DB {
		_, err := dbforpostgresql.NewDatabase(ctx, sanitized+"-db", &dbforpostgresql.DatabaseArgs{
			ResourceGroupName: infra.ResourceGroup.Name,
			ServerName:        server.Name,
			DatabaseName:      pg.DBName,
		}, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating PostgreSQL database: %w", err)
		}
	}

	// Without VNet, allow Azure services to connect via the firewall (0.0.0.0 rule).
	// With VNet integration, public access is disabled and all traffic is routed through the VNet.
	if infra.Networking == nil {
		_, err = dbforpostgresql.NewFirewallRule(ctx, sanitized+"-allow-azure", &dbforpostgresql.FirewallRuleArgs{
			ResourceGroupName: infra.ResourceGroup.Name,
			ServerName:        server.Name,
			StartIpAddress:    pulumi.String("0.0.0.0"),
			EndIpAddress:      pulumi.String("0.0.0.0"),
		}, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating PostgreSQL firewall rule: %w", err)
		}
	}

	return &postgresResult{Server: server}, nil
}
