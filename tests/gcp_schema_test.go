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
	assert.Contains(t, schema.Schema, "defang-gcp:defanggcp:Project")
	assert.Contains(t, schema.Schema, "endpoints")
	assert.Contains(t, schema.Schema, "services")
}

func TestGcpCloudRunServiceSchemaRegistered(t *testing.T) {
	server := makeGcpTestServer()

	schema, err := server.GetSchema(p.GetSchemaRequest{})
	require.NoError(t, err)
	assert.Contains(t, schema.Schema, "defang-gcp:defanggcp:GcpCloudRunService")
	assert.Contains(t, schema.Schema, "endpoint")
	assert.Contains(t, schema.Schema, "image")
}

func TestGcpCloudSqlSchemaRegistered(t *testing.T) {
	server := makeGcpTestServer()

	schema, err := server.GetSchema(p.GetSchemaRequest{})
	require.NoError(t, err)
	assert.Contains(t, schema.Schema, "defang-gcp:defanggcp:GcpCloudSql")
	assert.Contains(t, schema.Schema, "endpoint")
}
