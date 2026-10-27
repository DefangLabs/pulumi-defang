package scaleway

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/containers"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/databases"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/network"
)

var (
	ErrPostgresConfigNil       = errors.New("postgres config is nil")
	ErrPostgresPasswordMissing = errors.New(
		"postgres password is required for Scaleway Managed Database for PostgreSQL",
	)
)

// Scaleway reserves the default PostgreSQL user/database name "postgres" for
// managed instances. Use Defang-specific defaults unless the Compose service
// explicitly provides POSTGRES_USER or POSTGRES_DB.
const defaultScalewayPostgresUser = "defang"
const defaultScalewayPostgresDB = "defang"

// SharedInfra contains Scaleway resources shared by project-level components.
type SharedInfra struct {
	PluginIdentity common.PluginIdentity
	Region         string
	Zone           string
	ProjectID      string
	Namespace      *containers.Namespace
	PrivateNetwork *network.PrivateNetwork
	ConfigProvider compose.ConfigProvider
	Etag           string
	BuildInfra     *BuildInfra
	// ManagedHosts maps Compose service names (e.g., "database") to the actual
	// hostname of the managed resource. Used by container services to rewrite
	// environment variables that reference managed services by Compose name.
	ManagedHosts map[string]pulumi.StringOutput
	// ManagedConnectionURLs maps Compose service names to their full connection
	// URLs (e.g., "postgres://user:pass@host:port/db"). Used to replace env
	// values that contain URLs pointing to managed services.
	ManagedConnectionURLs map[string]pulumi.StringOutput
}

type PostgresResult struct {
	Instance      *databases.Instance
	Database      *databases.Database
	Privilege     *databases.Privilege
	Host          pulumi.StringOutput
	Port          pulumi.IntOutput
	ConnectionURL pulumi.StringOutput
}

func postgresEngine(version int) string {
	switch version {
	case 14, 15, 16, 17:
		return fmt.Sprintf("PostgreSQL-%d", version)
	default:
		return "PostgreSQL-17"
	}
}

type postgresNode struct {
	name       string
	vcpus      int
	memMiB     int
	haRequired bool // Scaleway only offers this node with a standby
}

// Managed Postgres node types by size, as Scaleway's RDB API lists them
// (memory there is in decimal GB, so DB-DEV-S's "2 GB" is 1907 MiB).
var (
	postgresDevNodes = []postgresNode{
		{"DB-DEV-S", 2, 1907, false},
		{"DB-DEV-M", 3, 3814, false},
		{"DB-DEV-L", 4, 7629, false},
		{"DB-DEV-XL", 4, 11444, false},
	}
	postgresGPNodes = []postgresNode{
		{"DB-GP-XS", 4, 15258, false},
		{"DB-GP-S", 8, 30517, false},
		{"DB-GP-M", 16, 61035, false},
		{"DB-GP-L", 32, 122070, true},
		{"DB-GP-XL", 48, 244140, true},
	}
	postgresNodeTypeCatalogs = map[string][]postgresNode{
		"dev": append(slices.Clone(postgresDevNodes), postgresGPNodes...),
		"gp":  postgresGPNodes,
	}
)

// postgresNodeType returns the smallest node in the recipe's family with at
// least the service's CPUs and memory, or the family's largest.
func postgresNodeType(family string, cpus float64, memMiB int) postgresNode {
	nodes, ok := postgresNodeTypeCatalogs[strings.ToLower(family)]
	if !ok {
		nodes = postgresNodeTypeCatalogs["dev"]
	}
	for _, n := range nodes {
		if float64(n.vcpus) >= cpus && n.memMiB >= memMiB {
			return n
		}
	}
	return nodes[len(nodes)-1]
}

func postgresPassword(password pulumi.StringInput) pulumi.StringPtrInput {
	return password.ToStringOutput().ApplyT(func(p string) (*string, error) {
		if p == "" {
			return nil, fmt.Errorf("%w; set it with `defang config set POSTGRES_PASSWORD`", ErrPostgresPasswordMissing)
		}
		return &p, nil
	}).(pulumi.StringPtrOutput)
}

func scalewayPostgresUsername(
	ctx *pulumi.Context,
	svc compose.ServiceConfig,
	username pulumi.StringInput,
) pulumi.StringInput {
	if _, ok := svc.Environment["POSTGRES_USER"]; !ok {
		return pulumi.String(defaultScalewayPostgresUser)
	}
	return username.ToStringOutput().ApplyT(func(u string) string {
		if strings.EqualFold(u, "postgres") {
			_ = ctx.Log.Warn(
				fmt.Sprintf("POSTGRES_USER %q is reserved by Scaleway; using %q instead", u, defaultScalewayPostgresUser),
				nil,
			)
			return defaultScalewayPostgresUser
		}
		return u
	}).(pulumi.StringOutput)
}

func scalewayPostgresDBName(
	ctx *pulumi.Context,
	svc compose.ServiceConfig,
	pg *compose.PostgresConfigArgs,
) (pulumi.StringInput, string) {
	if _, ok := svc.Environment["POSTGRES_DB"]; !ok {
		return pulumi.String(defaultScalewayPostgresDB), defaultScalewayPostgresDB
	}
	if strings.EqualFold(pg.DBNameStr, compose.DEFAULT_POSTGRES_DB) {
		_ = ctx.Log.Warn(
			fmt.Sprintf("POSTGRES_DB %q is reserved by Scaleway; using %q instead", pg.DBNameStr, defaultScalewayPostgresDB),
			nil,
		)
		return pulumi.String(defaultScalewayPostgresDB), defaultScalewayPostgresDB
	}
	return pg.DBName, pg.DBNameStr
}

func postgresHostAndPort(instance *databases.Instance) (pulumi.StringOutput, pulumi.IntOutput) {
	privateEndpoint := instance.PrivateNetwork.ApplyT(func(pn *databases.InstancePrivateNetwork) string {
		if pn != nil {
			if pn.Hostname != nil && *pn.Hostname != "" {
				return *pn.Hostname
			}
			if pn.Ip != nil && *pn.Ip != "" {
				return *pn.Ip
			}
		}
		return ""
	}).(pulumi.StringOutput)
	host := pulumi.All(privateEndpoint, instance.EndpointIp).ApplyT(func(args []any) string {
		if privateHost := args[0].(string); privateHost != "" {
			return privateHost
		}
		return args[1].(string)
	}).(pulumi.StringOutput)

	privatePort := instance.PrivateNetwork.ApplyT(func(pn *databases.InstancePrivateNetwork) int {
		if pn != nil && pn.Port != nil && *pn.Port > 0 {
			return *pn.Port
		}
		return 0
	}).(pulumi.IntOutput)
	port := pulumi.All(privatePort, instance.EndpointPort).ApplyT(func(args []any) int {
		if p := args[0].(int); p > 0 {
			return p
		}
		if p := args[1].(int); p > 0 {
			return p
		}
		return 5432
	}).(pulumi.IntOutput)

	return host, port
}

//nolint:funlen // Pulumi resource construction is clearer kept in one function.
func CreatePostgres(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	serviceName string,
	svc compose.ServiceConfig,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*PostgresResult, error) {
	if configProvider == nil {
		configProvider = &compose.PulumiConfigProvider{}
	}
	parent := pulumi.Parent(nil)
	for _, option := range opts {
		if shared, ok := option.(pulumi.ResourceOrInvokeOption); ok {
			parent = shared
			break
		}
	}
	pg := svc.ResolvePostgres(ctx, configProvider, parent)
	if pg == nil {
		return nil, ErrPostgresConfigNil
	}
	username := scalewayPostgresUsername(ctx, svc, pg.Username)
	dbName, dbNameStr := scalewayPostgresDBName(ctx, svc, pg)

	var engine pulumi.StringOutput
	if pg.Version != nil {
		engine = pg.Version.ToStringPtrOutput().ApplyT(func(version *string) string {
			if version == nil || *version == "" {
				return postgresEngine(0)
			}
			return postgresEngine(compose.GetPostgresVersion(*version))
		}).(pulumi.StringOutput)
	} else {
		engine = pulumi.String(postgresEngine(0)).ToStringOutput()
	}

	node := postgresNodeType(PostgresNodeType.Get(ctx), svc.GetCPUs(), svc.GetMemoryMiB())
	args := &databases.InstanceArgs{
		Name:              pulumi.String(serviceName),
		NodeType:          pulumi.String(node.name),
		Engine:            engine,
		UserName:          username.ToStringOutput().ToStringPtrOutput(),
		PasswordWo:        postgresPassword(pg.Password),
		PasswordWoVersion: pulumi.Int(1),
		IsHaCluster:       pulumi.Bool(HighAvailability.Get(ctx) || node.haRequired),
		EncryptionAtRest:  pulumi.Bool(true),
	}
	if days := BackupRetentionDays.Get(ctx); days > 0 {
		args.DisableBackup = pulumi.Bool(false)
		args.BackupScheduleFrequency = pulumi.Int(24)
		args.BackupScheduleRetention = pulumi.Int(days)
	} else {
		args.DisableBackup = pulumi.Bool(true)
	}
	if pg.FromSnapshot != "" {
		args.SnapshotId = pulumi.StringPtr(pg.FromSnapshot)
	}
	if infra != nil {
		if infra.Region != "" {
			args.Region = pulumi.StringPtr(infra.Region)
		}
		if infra.ProjectID != "" {
			args.ProjectId = pulumi.StringPtr(infra.ProjectID)
		}
		if infra.PrivateNetwork != nil {
			args.PrivateNetwork = &databases.InstancePrivateNetworkArgs{
				PnId:       infra.PrivateNetwork.ID(),
				EnableIpam: pulumi.Bool(true),
			}
			if infra.Zone != "" {
				args.PrivateNetwork.(*databases.InstancePrivateNetworkArgs).Zone = pulumi.StringPtr(infra.Zone)
			}
		}
	}

	instance, err := databases.NewInstance(ctx, serviceName, args, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Scaleway PostgreSQL instance: %w", err)
	}

	host, port := postgresHostAndPort(instance)
	result := &PostgresResult{
		Instance: instance,
		Host:     host,
		Port:     port,
	}

	if dbNameStr != "" {
		db, err := databases.NewDatabase(ctx, serviceName+"-db", &databases.DatabaseArgs{
			InstanceId: instance.ID(),
			Name:       dbName.ToStringOutput().ToStringPtrOutput(),
			Region:     args.Region,
		}, append(opts, pulumi.Parent(instance))...)
		if err != nil {
			return nil, fmt.Errorf("creating Scaleway PostgreSQL database: %w", err)
		}
		result.Database = db
		dbName = db.Name
	}

	privilege, err := databases.NewPrivilege(ctx, serviceName+"-privilege", &databases.PrivilegeArgs{
		InstanceId:   instance.ID(),
		DatabaseName: dbName,
		UserName:     username,
		Permission:   pulumi.String("all"),
		Region:       args.Region,
	}, append(opts, pulumi.Parent(instance))...)
	if err != nil {
		return nil, fmt.Errorf("creating Scaleway PostgreSQL privilege: %w", err)
	}
	result.Privilege = privilege

	result.ConnectionURL = pulumi.ToSecret(pulumi.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=require",
		username, pg.Password, host, port, dbName,
	)).(pulumi.StringOutput)

	return result, nil
}
