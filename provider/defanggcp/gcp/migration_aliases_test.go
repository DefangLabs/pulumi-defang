package gcp

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/require"
)

type migrationAliasMocks struct {
	aliases map[string][]string
}

func (m *migrationAliasMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if m.aliases == nil {
		m.aliases = map[string][]string{}
	}
	if args.RegisterRPC != nil {
		aliases := append([]string{}, args.RegisterRPC.GetAliasURNs()...)
		for _, alias := range args.RegisterRPC.GetAliases() {
			if urn := alias.GetUrn(); urn != "" {
				aliases = append(aliases, urn)
			}
		}
		m.aliases[args.TypeToken] = aliases
	}
	return args.Name + "_id", args.Inputs, nil
}

func (*migrationAliasMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func TestCloudSQLRegistersExactMigrationAliases(t *testing.T) {
	const (
		instanceURN = "urn:pulumi:prod::app::gcp:sql/databaseInstance:DatabaseInstance::legacy-db"
		userURN     = "urn:pulumi:prod::app::gcp:sql/user:User::legacy-user"
		databaseURN = "urn:pulumi:prod::app::gcp:sql/database:Database::legacy-database"
	)
	mocks := &migrationAliasMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := CreateCloudSQL(ctx, &compose.DryRunConfigProvider{}, "db", compose.ServiceConfig{
			Image:    pulumi.String("postgres:16"),
			Postgres: &compose.PostgresConfig{},
			Environment: compose.Environment{
				"POSTGRES_PASSWORD": pulumi.String("password"),
				"POSTGRES_DB":       pulumi.String("application"),
			},
			Aliases: map[string]string{
				compose.AliasInstance: instanceURN,
				compose.AliasUser:     userURN,
				compose.AliasDatabase: databaseURN,
			},
		}, nil)
		return err
	}, pulumi.WithMocks("app", "prod", mocks))
	require.NoError(t, err)
	require.Contains(t, mocks.aliases["gcp:sql/databaseInstance:DatabaseInstance"], instanceURN)
	require.Contains(t, mocks.aliases["gcp:sql/user:User"], userURN)
	require.Contains(t, mocks.aliases["gcp:sql/database:Database"], databaseURN)
}

func TestMemorystoreRegistersExactMigrationAlias(t *testing.T) {
	const instanceURN = "urn:pulumi:prod::app::gcp:redis/instance:Instance::legacy-cache"
	mocks := &migrationAliasMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := CreateMemoryStore(ctx, "cache", compose.ServiceConfig{
			Image:   pulumi.String("redis:7.2"),
			Redis:   &compose.RedisConfig{},
			Aliases: map[string]string{compose.AliasInstance: instanceURN},
		}, nil)
		return err
	}, pulumi.WithMocks("app", "prod", mocks))
	require.NoError(t, err)
	require.Contains(t, mocks.aliases["gcp:redis/instance:Instance"], instanceURN)
}
