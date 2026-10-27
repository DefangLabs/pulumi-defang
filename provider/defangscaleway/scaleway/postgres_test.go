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
	switch args.TypeToken {
	case "scaleway:databases/instance:Instance":
		outputs[resource.PropertyKey("endpointIp")] = resource.NewStringProperty("10.0.0.5")
		outputs[resource.PropertyKey("endpointPort")] = resource.NewNumberProperty(5432)
	case "scaleway:redis/cluster:Cluster":
		outputs[resource.PropertyKey("connectionString")] = resource.NewStringProperty("redis://10.0.0.7:6379")
	case "scaleway:containers/namespace:Namespace":
		outputs[resource.PropertyKey("registryEndpoint")] = resource.NewStringProperty("rg.fr-par.scw.cloud/defang")
	case "scaleway:containers/container:Container":
		outputs[resource.PropertyKey("domainName")] = resource.NewStringProperty("https://app.functions.fnc.fr-par.scw.cloud")
	}

	m.mu.Lock()
	m.records = append(m.records, resourceRecord{
		typ:    args.TypeToken,
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
	tests := []struct {
		family string
		cpus   float64
		memMiB int
		want   string
	}{
		{"burstable", 0.25, 512, "DB-DEV-S"},
		{"burstable", 2, 2048, "DB-DEV-S"},
		{"burstable", 2, 2049, "DB-DEV-M"},
		{"burstable", 2, 4096, "DB-DEV-M"},
		{"burstable", 4, 8192, "DB-DEV-L"},
		{"burstable", 4, 12288, "DB-DEV-XL"},
		{"burstable", 4, 16384, "DB-GP-XS"},
		{"burstable", 8, 16384, "DB-GP-S"},
		{"burstable", 16, 32768, "DB-GP-M"},
		{"burstable", 64, 1 << 20, "DB-GP-XL"},
		{"general", 0.25, 512, "DB-GP-XS"},
		{"GENERAL", 8, 16384, "DB-GP-S"},
		{"unknown", 0.25, 512, "DB-DEV-S"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, postgresNodeType(tt.family, tt.cpus, tt.memMiB).name,
			"family=%s cpus=%v memMiB=%d", tt.family, tt.cpus, tt.memMiB)
	}
}

func TestCreatePostgresManagedResources(t *testing.T) {
	mocks := &recordingMocks{}
	password := "Secret123!"
	dbName := "app"
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := CreatePostgres(ctx, &mockConfigProvider{}, "db", compose.ServiceConfig{
			Image:    compose.ImageFromPtr(ptr("postgres:16")),
			Postgres: &compose.PostgresConfig{},
			Environment: compose.Environment{
				"POSTGRES_PASSWORD": pulumi.String(password),
				"POSTGRES_DB":       pulumi.String(dbName),
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
	assert.True(t, inst.inputs[resource.PropertyKey("disableBackup")].BoolValue(), "backups off by default")
	assert.False(t, inst.inputs[resource.PropertyKey("isHaCluster")].BoolValue(), "standalone by default")
	assert.Equal(t, defaultScalewayPostgresUser, inst.inputs[resource.PropertyKey("userName")].StringValue())

	db := mocks.findType("scaleway:databases/database:Database")
	require.NotNil(t, db)
	assert.Equal(t, "app", db.inputs[resource.PropertyKey("name")].StringValue())

	privilege := mocks.findType("scaleway:databases/privilege:Privilege")
	require.NotNil(t, privilege)
	assert.Equal(t, "all", privilege.inputs[resource.PropertyKey("permission")].StringValue())
}

func TestCreatePostgresRecipe(t *testing.T) {
	t.Setenv("PULUMI_CONFIG", `{
		"defang-scaleway:backup-retention-days": "7",
		"defang-scaleway:postgres-node-type": "general",
		"defang-scaleway:high-availability": "true"
	}`)
	mocks := &recordingMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := CreatePostgres(ctx, &mockConfigProvider{}, "db", compose.ServiceConfig{
			Image:       compose.ImageFromPtr(ptr("postgres:16")),
			Postgres:    &compose.PostgresConfig{},
			Environment: compose.Environment{"POSTGRES_PASSWORD": pulumi.String("Secret123!")},
		}, nil)
		return err
	}, pulumi.WithMocks("proj", "stack", mocks))

	require.NoError(t, err)
	inst := mocks.findType("scaleway:databases/instance:Instance")
	require.NotNil(t, inst)
	assert.Equal(t, "DB-GP-XS", inst.inputs[resource.PropertyKey("nodeType")].StringValue())
	assert.False(t, inst.inputs[resource.PropertyKey("disableBackup")].BoolValue())
	assert.Equal(t, 7, int(inst.inputs[resource.PropertyKey("backupScheduleRetention")].NumberValue()))
	assert.Equal(t, 24, int(inst.inputs[resource.PropertyKey("backupScheduleFrequency")].NumberValue()))
	assert.True(t, inst.inputs[resource.PropertyKey("isHaCluster")].BoolValue())
}

func TestPostgresNodeTypeRequiresHA(t *testing.T) {
	assert.False(t, postgresNodeType("burstable", 16, 65536).haRequired)
	assert.True(t, postgresNodeType("burstable", 16, 65537).haRequired, "DB-GP-L is only offered with a standby")
}

func TestCreatePostgresAttachesPrivateNetwork(t *testing.T) {
	mocks := &recordingMocks{}
	password := "Secret123!"
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		pn, err := network.NewPrivateNetwork(ctx, "pn", &network.PrivateNetworkArgs{})
		if err != nil {
			return err
		}
		_, err = CreatePostgres(ctx, &mockConfigProvider{}, "db", compose.ServiceConfig{
			Image:    compose.ImageFromPtr(ptr("postgres:16")),
			Postgres: &compose.PostgresConfig{},
			Environment: compose.Environment{
				"POSTGRES_PASSWORD": pulumi.String(password),
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

func TestPostgresPreviewUsesCommonPlaceholderWithoutCloudCalls(t *testing.T) {
	mocks := &recordingMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		assert.True(t, ctx.DryRun())
		_, err := CreatePostgres(ctx, &compose.DryRunConfigProvider{}, "db", compose.ServiceConfig{
			Postgres:    &compose.PostgresConfig{},
			Environment: compose.Environment{"POSTGRES_PASSWORD": nil},
		}, nil)
		return err
	}, pulumi.WithMocks("proj", "stack", mocks), func(info *pulumi.RunInfo) { info.DryRun = true })
	require.NoError(t, err)
	instance := mocks.findType("scaleway:databases/instance:Instance")
	require.NotNil(t, instance)
	password := instance.inputs["passwordWo"]
	require.True(t, password.IsSecret())
	assert.Equal(t, "dry-run-POSTGRES_PASSWORD", password.SecretValue().Element.StringValue())
}
