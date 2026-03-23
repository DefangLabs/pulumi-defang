package azure

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

func TestConstructAzurePostgres(t *testing.T) {
	server := testutil.MakeAzureTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AzureURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("postgres:16"),
			"postgres":     property.New(property.NewMap(map[string]property.Value{})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAzurePostgresWithAllowDowntime(t *testing.T) {
	server := testutil.MakeAzureTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AzureURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("postgres:15"),
			"postgres": property.New(property.NewMap(map[string]property.Value{
				"allowDowntime": property.New(true),
			})),
			"aws": property.New(property.NewMap(map[string]property.Value{
				"vpcId": property.New("vpc-12345"),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAzurePostgresWithEnvironment(t *testing.T) {
	server := testutil.MakeAzureTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AzureURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("postgres:16"),
			"postgres":     property.New(property.NewMap(map[string]property.Value{})),
			"environment": property.New(property.NewMap(map[string]property.Value{
				"POSTGRES_DB":       property.New("mydb"),
				"POSTGRES_PASSWORD": property.New("secret"),
			})),
			"aws": property.New(property.NewMap(map[string]property.Value{
				"vpcId": property.New("vpc-12345"),
			})),
		}),
	})

	require.NoError(t, err)
}
