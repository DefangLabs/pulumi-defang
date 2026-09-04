package gcp

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGcpPostgresVersion(t *testing.T) {
	tests := []struct {
		version int
		want    string
	}{
		{9, "POSTGRES_9_6"},
		{10, "POSTGRES_10"},
		{11, "POSTGRES_11"},
		{12, "POSTGRES_12"},
		{13, "POSTGRES_13"},
		{14, "POSTGRES_14"},
		{15, "POSTGRES_15"},
		{16, "POSTGRES_16"},
		{17, "POSTGRES_17"},
		{0, "POSTGRES_17"},  // unparseable tag → latest
		{99, "POSTGRES_17"}, // unknown major → latest
	}
	for _, tt := range tests {
		if got := gcpPostgresVersion(tt.version); got != tt.want {
			t.Errorf("gcpPostgresVersion(%d) = %q, want %q", tt.version, got, tt.want)
		}
	}
}

// secretProviderSpy records the provider each Secret Manager invoke was made
// with. The CD runs with pulumi:disable-default-providers, so an invoke that
// reaches the engine without one is rejected.
type secretProviderSpy struct {
	noopMocks
	providers []string
}

func (m *secretProviderSpy) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	if args.Token == getSecretVersionToken {
		m.providers = append(m.providers, args.Provider)
		return resource.PropertyMap{
			"secretData": resource.NewStringProperty("from-secret-manager"),
		}, nil
	}
	return args.Args, nil
}

// A managed Postgres whose POSTGRES_PASSWORD comes from `defang config` (an
// empty YAML value, not a literal) has to read it out of Secret Manager. That
// lookup must carry an explicit provider, or the deploy dies with
// `Default provider for 'gcp' disabled` — surfaced as a misleading
// `config "POSTGRES_PASSWORD" not found`.
func TestCloudSQLResolvesConfigPasswordWithExplicitProvider(t *testing.T) {
	spy := &secretProviderSpy{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		prov, err := gcp.NewProvider(ctx, "explicit-gcp", &gcp.ProviderArgs{})
		require.NoError(t, err)

		_, err = CreateCloudSQL(ctx, NewConfigProvider("myproject"), "database", compose.ServiceConfig{
			Image:    pulumi.String("postgres:16"),
			Postgres: &compose.PostgresConfig{},
			Environment: compose.Environment{
				// nil == "resolve from config at deploy time"
				"POSTGRES_PASSWORD": nil,
				"POSTGRES_USER":     pulumi.String("martekio"),
			},
		}, nil, pulumi.Provider(prov))
		return err
	}, pulumi.WithMocks("myproject", "mystack", spy))
	require.NoError(t, err)

	require.NotEmpty(t, spy.providers, "expected a Secret Manager lookup for POSTGRES_PASSWORD")
	for _, p := range spy.providers {
		assert.NotEmpty(t, p, "Secret Manager invoke was made without an explicit provider")
	}
}

// aliasSpy records the aliases each registered resource carried.
type aliasSpy struct {
	noopMocks
	aliases map[string][]string
}

func (m *aliasSpy) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if m.aliases == nil {
		m.aliases = map[string][]string{}
	}
	for _, alias := range args.RegisterRPC.GetAliases() {
		if urn := alias.GetUrn(); urn != "" {
			m.aliases[args.Name] = append(m.aliases[args.Name], urn)
		}
	}
	return args.Name + "_id", args.Inputs, nil
}

// A legacy defang-mvp stack registered its Cloud SQL resources under different
// URNs than this CD does. Without aliases an `up` creates a new instance and
// deletes the old one — and the legacy DatabaseInstance is not retained, so
// that destroys the data. x-defang-aliases carries the old URNs across.
func TestCloudSQLAppliesMigrationAliases(t *testing.T) {
	const (
		instanceURN = "urn:pulumi:mig::proj::gcp:sql/databaseInstance:DatabaseInstance::legacy-postgres"
		userURN     = "urn:pulumi:mig::proj::gcp:sql/user:User::legacy-postgres-user"
		databaseURN = "urn:pulumi:mig::proj::gcp:sql/database:Database::legacy-postgres-db"
	)
	spy := &aliasSpy{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := CreateCloudSQL(ctx, &compose.DryRunConfigProvider{}, "database", compose.ServiceConfig{
			Image:    pulumi.String("postgres:16"),
			Postgres: &compose.PostgresConfig{},
			Environment: compose.Environment{
				"POSTGRES_PASSWORD": pulumi.String("literal-password"),
				"POSTGRES_USER":     pulumi.String("martekio"),
				"POSTGRES_DB":       pulumi.String("martekioDB"),
			},
			Aliases: map[string]string{
				compose.AliasInstance: instanceURN,
				compose.AliasUser:     userURN,
				compose.AliasDatabase: databaseURN,
			},
		}, nil, pulumi.Parent(nil))
		return err
	}, pulumi.WithMocks("proj", "mig", spy))
	require.NoError(t, err)

	assert.Contains(t, spy.aliases["database"], instanceURN, "Cloud SQL instance lost its migration alias")
	assert.Contains(t, spy.aliases["database-user"], userURN, "Cloud SQL user lost its migration alias")
	assert.Contains(t, spy.aliases["database-db"], databaseURN, "Cloud SQL database lost its migration alias")
}

// No x-defang-aliases means no aliases: a greenfield stack must not carry them.
func TestCloudSQLWithoutAliasesRegistersNone(t *testing.T) {
	spy := &aliasSpy{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := CreateCloudSQL(ctx, &compose.DryRunConfigProvider{}, "database", compose.ServiceConfig{
			Image:       pulumi.String("postgres:16"),
			Postgres:    &compose.PostgresConfig{},
			Environment: compose.Environment{"POSTGRES_PASSWORD": pulumi.String("literal-password")},
		}, nil, pulumi.Parent(nil))
		return err
	}, pulumi.WithMocks("proj", "mig", spy))
	require.NoError(t, err)

	assert.Empty(t, spy.aliases["database"])
}
