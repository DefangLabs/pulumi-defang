package scaleway

import (
	"context"
	"testing"

	defangscaleway "github.com/DefangLabs/pulumi-defang/provider/defangscaleway"
	"github.com/blang/semver"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeScalewayTestServer(t *testing.T) integration.Server {
	t.Helper()

	server, err := integration.NewServer(
		context.Background(),
		defangscaleway.Name,
		semver.MustParse("1.0.0"),
		integration.WithProvider(defangscaleway.Provider()),
	)
	require.NoError(t, err)
	return server
}

func TestScalewayProjectSchemaRegistered(t *testing.T) {
	server := makeScalewayTestServer(t)

	schema, err := server.GetSchema(p.GetSchemaRequest{})
	require.NoError(t, err)
	assert.Contains(t, schema.Schema, "defang-scaleway:index:Project")
	assert.Contains(t, schema.Schema, "endpoints")
	assert.Contains(t, schema.Schema, "services")
}

func TestScalewayServiceSchemaRegistered(t *testing.T) {
	server := makeScalewayTestServer(t)

	schema, err := server.GetSchema(p.GetSchemaRequest{})
	require.NoError(t, err)
	assert.Contains(t, schema.Schema, "defang-scaleway:index:Service")
	assert.Contains(t, schema.Schema, "endpoint")
	assert.Contains(t, schema.Schema, "image")
}

func TestScalewayPostgresSchemaRegistered(t *testing.T) {
	server := makeScalewayTestServer(t)

	schema, err := server.GetSchema(p.GetSchemaRequest{})
	require.NoError(t, err)
	assert.Contains(t, schema.Schema, "defang-scaleway:index:Postgres")
	assert.Contains(t, schema.Schema, "endpoint")
}

func TestScalewayRedisSchemaRegistered(t *testing.T) {
	server := makeScalewayTestServer(t)

	schema, err := server.GetSchema(p.GetSchemaRequest{})
	require.NoError(t, err)
	assert.Contains(t, schema.Schema, "defang-scaleway:index:Redis")
	assert.Contains(t, schema.Schema, "endpoint")
}
