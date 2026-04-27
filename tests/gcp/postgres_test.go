package gcp

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

func TestConstructGcpCloudSql(t *testing.T) {
	server := testutil.MakeGcpTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":        property.New("postgres:15"),
			"postgres":     property.New(property.NewMap(map[string]property.Value{})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructGcpCloudSqlWithAllowDowntime(t *testing.T) {
	server := testutil.MakeGcpTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":        property.New("postgres:16"),
			"postgres": property.New(property.NewMap(map[string]property.Value{
				"allowDowntime": property.New(true),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructGcpCloudSqlWithEnvironment(t *testing.T) {
	server := testutil.MakeGcpTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":        property.New("postgres:16"),
			"postgres":     property.New(property.NewMap(map[string]property.Value{})),
			"environment": property.New(property.NewMap(map[string]property.Value{
				"POSTGRES_DB":   property.New("mydb"),
				"POSTGRES_USER": property.New("admin"),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructGcpCloudSqlNoUserOrDatabaseByDefault(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":        property.New("postgres:17"),
			"postgres":     property.New(property.NewMap(map[string]property.Value{})),
		}),
	})

	require.NoError(t, err)
	assert.Equal(t, 0, countType(*records, "gcp:sql/user:User"),
		"no User should be created without POSTGRES_PASSWORD")
	assert.Equal(t, 0, countType(*records, "gcp:sql/database:Database"),
		"no Database should be created without a custom POSTGRES_DB")
}

func TestConstructGcpCloudSqlCreatesUserAndDatabaseWhenSet(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":        property.New("postgres:17"),
			"postgres":     property.New(property.NewMap(map[string]property.Value{})),
			"environment": property.New(property.NewMap(map[string]property.Value{
				"POSTGRES_DB":       property.New("mydb"),
				"POSTGRES_PASSWORD": property.New("secret"),
			})),
		}),
	})

	require.NoError(t, err)
	assert.Equal(t, 1, countType(*records, "gcp:sql/user:User"),
		"User should be created when POSTGRES_PASSWORD is set")
	assert.Equal(t, 1, countType(*records, "gcp:sql/database:Database"),
		"Database should be created when POSTGRES_DB is set to a non-default name")
}

func TestConstructGcpCloudSqlDefaultDbNameSkipsDatabase(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":        property.New("postgres:17"),
			"postgres":     property.New(property.NewMap(map[string]property.Value{})),
			"environment": property.New(property.NewMap(map[string]property.Value{
				"POSTGRES_DB": property.New("postgres"), // explicit default — still no Database resource
			})),
		}),
	})

	require.NoError(t, err)
	assert.Equal(t, 0, countType(*records, "gcp:sql/database:Database"),
		"no Database should be created when POSTGRES_DB is the default 'postgres'")
}

// TestConstructGcpCloudSqlStandaloneNoVPCPeering verifies that the standalone Postgres
// component (used outside a Project) does not create VPC peering infrastructure,
// since it has no project-level VPC context.
func TestConstructGcpCloudSqlStandaloneNoVPCPeering(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":        property.New("postgres:17"),
			"postgres":     property.New(property.NewMap(map[string]property.Value{})),
		}),
	})

	require.NoError(t, err)

	peering := findTypeWhere(*records, "gcp:compute/globalAddress:GlobalAddress", func(m property.Map) bool {
		v := m.Get("purpose")
		return !v.IsNull() && v.AsString() == gcpVPCPeeringPurpose
	})
	assert.Nil(t, peering, "standalone Postgres component should not create VPC peering infrastructure")
	assert.Equal(t, 0, countType(*records, "gcp:servicenetworking/connection:Connection"))
}
