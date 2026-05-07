package scaleway

import (
	"sync"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockConfigProvider struct {
	values map[string]string
}

func (m *mockConfigProvider) GetConfigValue(
	_ *pulumi.Context, key string, _ ...pulumi.InvokeOption,
) pulumi.StringOutput {
	if v, ok := m.values[key]; ok {
		return pulumi.String(v).ToStringOutput()
	}
	return compose.ConfigNotFoundOutput(key)
}

func (m *mockConfigProvider) GetSecretRef(
	_ *pulumi.Context, key string, _ ...pulumi.InvokeOption,
) (string, error) {
	return "mock-secret-" + key, nil
}

type resourceRecord struct {
	typ    string
	name   string
	inputs resource.PropertyMap
}

type recordingMocks struct {
	mu      sync.Mutex
	records []resourceRecord
}

func (m *recordingMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	outputs := resource.PropertyMap{}
	for k, v := range args.Inputs {
		outputs[k] = v
	}
	switch string(args.TypeToken) {
	case "scaleway:databases/instance:Instance":
		outputs[resource.PropertyKey("endpointIp")] = resource.NewStringProperty("10.0.0.5")
		outputs[resource.PropertyKey("endpointPort")] = resource.NewNumberProperty(5432)
	case "scaleway:redis/cluster:Cluster":
		outputs[resource.PropertyKey("connectionString")] = resource.NewStringProperty("redis://10.0.0.7:6379")
	}

	m.mu.Lock()
	m.records = append(m.records, resourceRecord{
		typ:    string(args.TypeToken),
		name:   args.Name,
		inputs: args.Inputs,
	})
	m.mu.Unlock()
	return args.Name + "_id", outputs, nil
}

func (m *recordingMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func (m *recordingMocks) findType(typ string) *resourceRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.records {
		if m.records[i].typ == typ {
			return &m.records[i]
		}
	}
	return nil
}

func TestPostgresEngine(t *testing.T) {
	assert.Equal(t, "PostgreSQL-14", postgresEngine(14))
	assert.Equal(t, "PostgreSQL-17", postgresEngine(17))
	assert.Equal(t, "PostgreSQL-17", postgresEngine(0))
}

func TestPostgresNodeType(t *testing.T) {
	assert.Equal(t, "DB-DEV-S", postgresNodeType(0.25, 512))
	assert.Equal(t, "DB-GP-S", postgresNodeType(2, 4096))
	assert.Equal(t, "DB-GP-M", postgresNodeType(4, 8192))
	assert.Equal(t, "DB-GP-L", postgresNodeType(8, 16384))
	assert.Equal(t, "DB-GP-XL", postgresNodeType(16, 32768))
}

func TestCreatePostgresManagedResources(t *testing.T) {
	mocks := &recordingMocks{}
	password := "secret"
	dbName := "app"
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := CreatePostgres(ctx, &mockConfigProvider{}, "db", compose.ServiceConfig{
			Image:    ptr("postgres:16"),
			Postgres: &compose.PostgresConfig{},
			Environment: map[string]*string{
				"POSTGRES_PASSWORD": &password,
				"POSTGRES_DB":       &dbName,
			},
		}, nil)
		return err
	}, pulumi.WithMocks("proj", "stack", mocks))

	require.NoError(t, err)
	inst := mocks.findType("scaleway:databases/instance:Instance")
	require.NotNil(t, inst)
	assert.Equal(t, "DB-DEV-S", inst.inputs[resource.PropertyKey("nodeType")].StringValue())
	assert.Equal(t, "PostgreSQL-16", inst.inputs[resource.PropertyKey("engine")].StringValue())
	assert.True(t, inst.inputs[resource.PropertyKey("encryptionAtRest")].BoolValue())
	assert.False(t, inst.inputs[resource.PropertyKey("disableBackup")].BoolValue())

	db := mocks.findType("scaleway:databases/database:Database")
	require.NotNil(t, db)
	assert.Equal(t, "app", db.inputs[resource.PropertyKey("name")].StringValue())

	privilege := mocks.findType("scaleway:databases/privilege:Privilege")
	require.NotNil(t, privilege)
	assert.Equal(t, "all", privilege.inputs[resource.PropertyKey("permission")].StringValue())
}

func TestCreatePostgresAttachesPrivateNetwork(t *testing.T) {
	mocks := &recordingMocks{}
	password := "secret"
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		pn, err := network.NewPrivateNetwork(ctx, "pn", &network.PrivateNetworkArgs{})
		if err != nil {
			return err
		}
		_, err = CreatePostgres(ctx, &mockConfigProvider{}, "db", compose.ServiceConfig{
			Image:    ptr("postgres:16"),
			Postgres: &compose.PostgresConfig{},
			Environment: map[string]*string{
				"POSTGRES_PASSWORD": &password,
			},
		}, &SharedInfra{Zone: "fr-par-1", PrivateNetwork: pn})
		return err
	}, pulumi.WithMocks("proj", "stack", mocks))

	require.NoError(t, err)
	inst := mocks.findType("scaleway:databases/instance:Instance")
	require.NotNil(t, inst)
	privateNetwork := inst.inputs[resource.PropertyKey("privateNetwork")].ObjectValue()
	assert.True(t, privateNetwork[resource.PropertyKey("enableIpam")].BoolValue())
	assert.Equal(t, "fr-par-1", privateNetwork[resource.PropertyKey("zone")].StringValue())
}
