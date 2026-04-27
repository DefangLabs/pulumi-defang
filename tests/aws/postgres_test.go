package aws

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

func TestConstructAwsPostgres(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":        property.New("postgres:16"),
			"postgres":     property.New(property.NewMap(map[string]property.Value{})),
			"aws": property.New(property.NewMap(map[string]property.Value{
				"vpcID": property.New("vpc-12345"),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsPostgresWithAllowDowntime(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":        property.New("postgres:15"),
			"postgres": property.New(property.NewMap(map[string]property.Value{
				"allowDowntime": property.New(true),
			})),
			"aws": property.New(property.NewMap(map[string]property.Value{
				"vpcID": property.New("vpc-12345"),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsPostgresWithSnapshot(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":        property.New("postgres:16"),
			"postgres": property.New(property.NewMap(map[string]property.Value{
				"fromSnapshot": property.New("rds:myproject-db-2024-01-01"),
			})),
			"aws": property.New(property.NewMap(map[string]property.Value{
				"vpcID": property.New("vpc-12345"),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsPostgresWithEnvironment(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":        property.New("postgres:16"),
			"postgres":     property.New(property.NewMap(map[string]property.Value{})),
			"environment": property.New(property.NewMap(map[string]property.Value{
				"POSTGRES_DB":       property.New("mydb"),
				"POSTGRES_USER":     property.New("admin"),
				"POSTGRES_PASSWORD": property.New("secret"),
			})),
			"aws": property.New(property.NewMap(map[string]property.Value{
				"vpcID": property.New("vpc-12345"),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsPostgresWithVPC(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"image":        property.New("postgres:16"),
			"postgres":     property.New(property.NewMap(map[string]property.Value{})),
			"aws": property.New(property.NewMap(map[string]property.Value{
				"vpcID": property.New("vpc-12345"),
				"privateSubnetDs": property.New(property.NewArray([]property.Value{
					property.New("subnet-private-0"),
					property.New("subnet-private-1"),
				})),
			})),
		}),
	})

	require.NoError(t, err)
}
