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
	assert.Contains(t, schema.Schema, "defang-azure:defangazure:Project")
	assert.Contains(t, schema.Schema, "endpoints")
	assert.Contains(t, schema.Schema, "services")
}

func TestAzureContainerAppSchemaRegistered(t *testing.T) {
	server := makeAzureTestServer()

	schema, err := server.GetSchema(p.GetSchemaRequest{})
	require.NoError(t, err)
	assert.Contains(t, schema.Schema, "defang-azure:defangazure:AzureContainerApp")
	assert.Contains(t, schema.Schema, "endpoint")
	assert.Contains(t, schema.Schema, "image")
}

func TestAzurePostgresSchemaRegistered(t *testing.T) {
	server := makeAzureTestServer()

	schema, err := server.GetSchema(p.GetSchemaRequest{})
	require.NoError(t, err)
	assert.Contains(t, schema.Schema, "defang-azure:defangazure:AzurePostgres")
	assert.Contains(t, schema.Schema, "endpoint")
}
