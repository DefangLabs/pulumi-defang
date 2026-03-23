package tests

// Postgres component tests across all three providers. These tests verify that
// Postgres, and Postgres each construct correctly under
// variety of configurations (minimal, with image/version, with snapshot, with
// allowDowntime).

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
)

// --- AWS Postgres ---

func TestConstructAwsPostgres(t *testing.T) {
	server := makeTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: awsURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("postgres:16"),
			"postgres":     property.New(property.NewMap(map[string]property.Value{})),
			"aws": property.New(property.NewMap(map[string]property.Value{
				"vpcId": property.New("vpc-12345"),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsPostgresWithAllowDowntime(t *testing.T) {
	server := makeTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: awsURN("Postgres"),
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

func TestConstructAwsPostgresWithSnapshot(t *testing.T) {
	server := makeTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: awsURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("postgres:16"),
			"postgres": property.New(property.NewMap(map[string]property.Value{
				"fromSnapshot": property.New("rds:myproject-db-2024-01-01"),
			})),
			"aws": property.New(property.NewMap(map[string]property.Value{
				"vpcId": property.New("vpc-12345"),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAwsPostgresWithEnvironment(t *testing.T) {
	server := makeTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: awsURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("postgres:16"),
			"postgres":     property.New(property.NewMap(map[string]property.Value{})),
			"environment": property.New(property.NewMap(map[string]property.Value{
				"POSTGRES_DB":       property.New("mydb"),
				"POSTGRES_USER":     property.New("admin"),
				"POSTGRES_PASSWORD": property.New("secret"),
			})),
			"aws": property.New(property.NewMap(map[string]property.Value{
				"vpcId": property.New("vpc-12345"),
			})),
		}),
	})

	require.NoError(t, err)
}

// --- GCP Cloud SQL ---

func TestConstructGcpCloudSql(t *testing.T) {
	server := makeGcpTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: gcpURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("postgres:15"),
			"postgres":     property.New(property.NewMap(map[string]property.Value{})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructGcpCloudSqlWithAllowDowntime(t *testing.T) {
	server := makeGcpTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: gcpURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("postgres:16"),
			"postgres": property.New(property.NewMap(map[string]property.Value{
				"allowDowntime": property.New(true),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructGcpCloudSqlWithEnvironment(t *testing.T) {
	server := makeGcpTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: gcpURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
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

// --- Azure Postgres ---

func TestConstructAzurePostgres(t *testing.T) {
	server := makeAzureTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: azureURN("Postgres"),
		Inputs: property.NewMap(map[string]property.Value{
			"project_name": property.New("myproject"),
			"image":        property.New("postgres:16"),
			"postgres":     property.New(property.NewMap(map[string]property.Value{})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAzurePostgresWithAllowDowntime(t *testing.T) {
	server := makeAzureTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: azureURN("Postgres"),
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
	server := makeAzureTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: azureURN("Postgres"),
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
