package scaleway

import (
	"errors"
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/databases"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/network"
)

var (
	ErrPostgresConfigNil       = errors.New("postgres config is nil")
	ErrPostgresPasswordMissing = errors.New("POSTGRES_PASSWORD is required for Scaleway Managed Database for PostgreSQL")
)

// SharedInfra contains Scaleway resources shared by project-level components.
type SharedInfra struct {
	Region         string
	Zone           string
	ProjectID      string
	PrivateNetwork *network.PrivateNetwork
	ConfigProvider compose.ConfigProvider
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

func postgresNodeType(cpus float64, memMiB int) string {
	switch {
	case cpus <= 1 && memMiB <= 2048:
		return "DB-DEV-S"
	case cpus <= 2 && memMiB <= 4096:
		return "DB-GP-S"
	case cpus <= 4 && memMiB <= 8192:
		return "DB-GP-M"
	case cpus <= 8 && memMiB <= 16384:
		return "DB-GP-L"
	default:
		return "DB-GP-XL"
	}
}

func postgresPassword(password pulumi.StringInput) pulumi.StringPtrInput {
	return password.ToStringOutput().ApplyT(func(p string) (*string, error) {
		if p == "" {
			return nil, fmt.Errorf("%w; set it with `defang config set POSTGRES_PASSWORD`", ErrPostgresPasswordMissing)
		}
		return &p, nil
	}).(pulumi.StringPtrOutput)
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
	pg := svc.ResolvePostgres(ctx, configProvider)
	if pg == nil {
		return nil, ErrPostgresConfigNil
	}

	engine := pg.Version.ToStringPtrOutput().ApplyT(func(version *string) string {
		if version == nil || *version == "" {
			return postgresEngine(0)
		}
		return postgresEngine(compose.GetPostgresVersion(*version))
	}).(pulumi.StringOutput)

	args := &databases.InstanceArgs{
		Name:              pulumi.String(serviceName),
		NodeType:          pulumi.String(postgresNodeType(svc.GetCPUs(), svc.GetMemoryMiB())),
		Engine:            engine,
		UserName:          pg.Username.ToStringOutput().ToStringPtrOutput(),
		PasswordWo:        postgresPassword(pg.Password),
		PasswordWoVersion: pulumi.Int(1),
		IsHaCluster:       pulumi.Bool(svc.GetReplicas() > 1),
		DisableBackup:     pulumi.Bool(false),
		EncryptionAtRest:  pulumi.Bool(true),
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

	dbName := pg.DBName
	if pg.DBNameStr != "" && pg.DBNameStr != compose.DEFAULT_POSTGRES_DB {
		db, err := databases.NewDatabase(ctx, serviceName+"-db", &databases.DatabaseArgs{
			InstanceId: instance.ID(),
			Name:       pg.DBName.ToStringOutput().ToStringPtrOutput(),
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
		UserName:     pg.Username,
		Permission:   pulumi.String("all"),
		Region:       args.Region,
	}, append(opts, pulumi.Parent(instance))...)
	if err != nil {
		return nil, fmt.Errorf("creating Scaleway PostgreSQL privilege: %w", err)
	}
	result.Privilege = privilege

	result.ConnectionURL = pulumi.ToSecret(pulumi.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=require",
		pg.Username, pg.Password, host, port, dbName,
	)).(pulumi.StringOutput)

	return result, nil
}
