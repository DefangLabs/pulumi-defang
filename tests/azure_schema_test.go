package tests

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAzureProjectSchemaRegistered(t *testing.T) {
	server := makeAzureTestServer()

	schema, err := server.GetSchema(p.GetSchemaRequest{})
	require.NoError(t, err)
	assert.Contains(t, schema.Schema, "defang-azure:index:Project")
	assert.Contains(t, schema.Schema, "endpoints")
	assert.Contains(t, schema.Schema, "services")
}

func TestAzureServiceSchemaRegistered(t *testing.T) {
	server := makeAzureTestServer()

	schema, err := server.GetSchema(p.GetSchemaRequest{})
	require.NoError(t, err)
	assert.Contains(t, schema.Schema, "defang-azure:index:Service")
	assert.Contains(t, schema.Schema, "endpoint")
	assert.Contains(t, schema.Schema, "image")
}

func TestAzurePostgresSchemaRegistered(t *testing.T) {
	server := makeAzureTestServer()

	schema, err := server.GetSchema(p.GetSchemaRequest{})
	require.NoError(t, err)
	assert.Contains(t, schema.Schema, "defang-azure:index:Postgres")
	assert.Contains(t, schema.Schema, "endpoint")
}
