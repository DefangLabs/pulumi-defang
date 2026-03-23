package tests

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGcpProjectSchemaRegistered(t *testing.T) {
	server := makeGcpTestServer()

	schema, err := server.GetSchema(p.GetSchemaRequest{})
	require.NoError(t, err)
	assert.Contains(t, schema.Schema, "defang-gcp:index:Project")
	assert.Contains(t, schema.Schema, "endpoints")
	assert.Contains(t, schema.Schema, "services")
}

func TestGcpServiceSchemaRegistered(t *testing.T) {
	server := makeGcpTestServer()

	schema, err := server.GetSchema(p.GetSchemaRequest{})
	require.NoError(t, err)
	assert.Contains(t, schema.Schema, "defang-gcp:index:Service")
	assert.Contains(t, schema.Schema, "endpoint")
	assert.Contains(t, schema.Schema, "image")
}

func TestGcpPostgresSchemaRegistered(t *testing.T) {
	server := makeGcpTestServer()

	schema, err := server.GetSchema(p.GetSchemaRequest{})
	require.NoError(t, err)
	assert.Contains(t, schema.Schema, "defang-gcp:index:Postgres")
	assert.Contains(t, schema.Schema, "endpoint")
}
