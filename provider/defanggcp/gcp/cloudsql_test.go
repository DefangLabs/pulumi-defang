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

		_, err = CreateCloudSQL(ctx, NewConfigProvider("myproject"), "myproject", "database", compose.ServiceConfig{
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

// aliasSpy records the legacy names each registered resource declared.
type aliasSpy struct {
	noopMocks
	aliases map[string][]string
}

func (m *aliasSpy) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if m.aliases == nil {
		m.aliases = map[string][]string{}
	}
	for _, alias := range args.RegisterRPC.GetAliases() {
		if spec := alias.GetSpec(); spec != nil {
			m.aliases[args.Name] = append(m.aliases[args.Name], spec.GetName())
		}
	}
	return args.Name + "_id", args.Inputs, nil
}

// Without these aliases an `up` over a legacy defang-mvp stack plans a create
// plus a delete for the database rather than an update, and the legacy
// DatabaseInstance carries no retainOnDelete — the delete takes the data too.
func TestCloudSQLAdoptsLegacyMvpNames(t *testing.T) {
	spy := &aliasSpy{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := CreateCloudSQL(ctx, &compose.DryRunConfigProvider{}, "martekio-mig", "database",
			compose.ServiceConfig{
				Image:    pulumi.String("postgres:16"),
				Postgres: &compose.PostgresConfig{},
				Environment: compose.Environment{
					"POSTGRES_PASSWORD": pulumi.String("literal-password"),
					"POSTGRES_USER":     pulumi.String("martekio"),
					"POSTGRES_DB":       pulumi.String("martekioDB"),
				},
			}, nil, pulumi.Parent(nil))
		return err
	}, pulumi.WithMocks("martekio-mig", "cyc", spy))
	require.NoError(t, err)

	assert.Contains(t, spy.aliases["database"], "defang-martekio-mig-cyc-database-postgres",
		"Cloud SQL instance would be replaced instead of adopted")
	assert.Contains(t, spy.aliases["database-user"], "martekio-mig-database-postgres-user")
	assert.Contains(t, spy.aliases["database-db"], "martekio-mig-database-postgres-db")
}
