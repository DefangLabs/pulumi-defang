package azure

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/dbforpostgresql/v3"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var (
	ErrPostgresConfigNil = errors.New("postgres config is nil")
	// Hint rendered via %w wrapping; see CreatePostgresFlexible.
	ErrPostgresPasswordMissing = errors.New("POSTGRES_PASSWORD is required for Azure PostgreSQL Flexible Server")
)

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

	// Readiness lists resources that must be fully applied before downstream
	// consumers (Container Apps running migrations, etc.) can safely use the
	// server. Includes the `azure.extensions` (pgvector) and
	// `require_secure_transport` configurations; without waiting for these, a
	// consumer can connect to the server but fail queries that depend on the
	// parameter change (e.g. CREATE EXTENSION vector).
	Readiness []pulumi.Resource
}

// buildPostgresServerArgs constructs the ServerArgs for an Azure PostgreSQL Flexible Server.
func buildPostgresServerArgs(
	pg *compose.PostgresConfigArgs,
	sanitized string,
	infra *SharedInfra,
	ctx *pulumi.Context,
	opts ...pulumi.ResourceOption,
) (*dbforpostgresql.ServerArgs, error) {
	// Backup config
	backupRetention := BackupRetentionDays.Get(ctx)
	geoBackup := dbforpostgresql.GeoRedundantBackupDisabled
	if GeoRedundantBackup.Get(ctx) {
		geoBackup = dbforpostgresql.GeoRedundantBackupEnabled
	}

	// ServerName is a required Azure API parameter (azure-native doesn't
	// auto-generate it) and must be a globally unique DNS label. Use
	// RandomString for the suffix: Pulumi persists it in state so subsequent
	// `up` runs reuse the same name, but a `down` wipes it — the next `up`
	// generates a fresh suffix, which avoids collisions with any server of
	// the same name that Azure might still be releasing after deletion.
	suffix, err := random.NewRandomString(ctx, sanitized+"-server-suffix", &random.RandomStringArgs{
		Length:  pulumi.Int(8),
		Lower:   pulumi.Bool(true),
		Upper:   pulumi.Bool(false),
		Numeric: pulumi.Bool(true),
		Special: pulumi.Bool(false),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating random server name suffix: %w", err)
	}
	// Postgres Flexible Server names must be 3–63 chars; truncate the prefix
	// so there's room for the "-<8-char suffix>" tail.
	prefix := sanitized
	if len(prefix) > 54 {
		prefix = strings.Trim(prefix[:54], "-")
	}
	serverName := suffix.Result.ApplyT(func(s string) string {
		return prefix + "-" + s
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
		AdministratorLogin: pg.Username,
		AdministratorLoginPassword: pg.Password.ToStringOutput().ApplyT(func(p string) (string, error) {
			if p == "" {
				return "", fmt.Errorf("%w; set it with `defang config set POSTGRES_PASSWORD`", ErrPostgresPasswordMissing)
			}
			return p, nil
		}).(pulumi.StringOutput),
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

	return serverArgs, nil
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

	serverArgs, err := buildPostgresServerArgs(pg, sanitized, infra, ctx, opts...)
	if err != nil {
		return nil, err
	}

	server, err := dbforpostgresql.NewServer(ctx, sanitized, serverArgs, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating PostgreSQL Flexible Server: %w", err)
	}

	serverChildOpts := append([]pulumi.ResourceOption{pulumi.Parent(server)}, opts...)

	// Allowlist the pgvector extension. Azure Postgres Flexible Server blocks CREATE EXTENSION
	// unless the extension is listed in the azure.extensions server parameter first.
	pgvectorCfg, err := dbforpostgresql.NewConfiguration(ctx, "pgvector", &dbforpostgresql.ConfigurationArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		ServerName:        server.Name,
		ConfigurationName: pulumi.String("azure.extensions"),
		Value:             pulumi.String("VECTOR"),
		Source:            pulumi.String("user-override"),
	}, serverChildOpts...)
	if err != nil {
		return nil, fmt.Errorf("enabling pgvector extension: %w", err)
	}

	// Allow non-TLS connections. Azure defaults require_secure_transport=ON, which
	// rejects non-SSL clients with "no pg_hba.conf entry ... no encryption". AWS/GCP
	// managed Postgres allow both, and samples (e.g. nextjs-postgres with
	// POSTGRES_SSL=disable) expect the same here. Clients can still negotiate TLS.
	//
	// DependsOn(pgvectorCfg): Azure serializes server-parameter changes — applying
	// two Configurations in parallel yields `ServerIsBusy` on the loser. Make this
	// one wait for pgvector to finish.
	noTlsOpts := append([]pulumi.ResourceOption{pulumi.DependsOn([]pulumi.Resource{pgvectorCfg})}, serverChildOpts...)
	noTlsCfg, err := dbforpostgresql.NewConfiguration(ctx, "no-tls", &dbforpostgresql.ConfigurationArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		ServerName:        server.Name,
		ConfigurationName: pulumi.String("require_secure_transport"),
		Value:             pulumi.String("OFF"),
		Source:            pulumi.String("user-override"),
	}, noTlsOpts...)
	if err != nil {
		return nil, fmt.Errorf("disabling require_secure_transport: %w", err)
	}

	// Create database if non-default; "postgres" already exists on every new Flexible Server.
	if pg.DBNameStr != "" && pg.DBNameStr != compose.DEFAULT_POSTGRES_DB {
		_, err := dbforpostgresql.NewDatabase(ctx, sanitized, &dbforpostgresql.DatabaseArgs{
			ResourceGroupName: infra.ResourceGroup.Name,
			ServerName:        server.Name,
			DatabaseName:      pg.DBName,
		}, serverChildOpts...)
		if err != nil {
			return nil, fmt.Errorf("creating PostgreSQL database: %w", err)
		}
	}

	// Without VNet, allow Azure services to connect via the firewall (0.0.0.0 rule).
	// With VNet integration, public access is disabled and all traffic is routed through the VNet.
	if infra.Networking == nil {
		_, err = dbforpostgresql.NewFirewallRule(ctx, sanitized, &dbforpostgresql.FirewallRuleArgs{
			ResourceGroupName: infra.ResourceGroup.Name,
			ServerName:        server.Name,
			StartIpAddress:    pulumi.String("0.0.0.0"),
			EndIpAddress:      pulumi.String("0.0.0.0"),
		}, serverChildOpts...)
		if err != nil {
			return nil, fmt.Errorf("creating PostgreSQL firewall rule: %w", err)
		}
	}

	return &postgresResult{
		Server:    server,
		Readiness: []pulumi.Resource{pgvectorCfg, noTlsCfg},
	}, nil
}
