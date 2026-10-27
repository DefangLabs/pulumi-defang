package defangscaleway

import (
	"sync"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type projectResourceRecord struct {
	typ       string
	name      string
	inputs    resource.PropertyMap
	pluginURL string
	version   string
}

type projectRecordingMocks struct {
	mu      sync.Mutex
	records []projectResourceRecord
}

func (m *projectRecordingMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	outputs := resource.PropertyMap{}
	for k, v := range args.Inputs {
		outputs[k] = v
	}
	switch args.TypeToken {
	case "scaleway:containers/namespace:Namespace":
		outputs[resource.PropertyKey("registryEndpoint")] = resource.NewStringProperty("rg.fr-par.scw.cloud/defang")
	case "scaleway:containers/container:Container":
		outputs[resource.PropertyKey("domainName")] = resource.NewStringProperty("app.functions.fnc.fr-par.scw.cloud")
	case "scaleway:network/privateNetwork:PrivateNetwork":
		outputs[resource.PropertyKey("id")] = resource.NewStringProperty(args.Name + "_id")
	case "scaleway:databases/instance:Instance":
		outputs[resource.PropertyKey("endpointIp")] = resource.NewStringProperty("10.0.0.5")
		outputs[resource.PropertyKey("endpointPort")] = resource.NewNumberProperty(5432)
		outputs[resource.PropertyKey("id")] = resource.NewStringProperty(args.Name + "_id")
	case "scaleway:databases/database:Database":
		outputs[resource.PropertyKey("name")] = resource.NewStringProperty("defang")
	case "scaleway:databases/privilege:Privilege":
		// no extra outputs needed
	case "defang-scaleway:index:Build":
		outputs[resource.PropertyKey("buildId")] = resource.NewStringProperty("build-123")
		outputs[resource.PropertyKey("image")] = resource.NewStringProperty("rg.fr-par.scw.cloud/defang/app:latest")
	}
	m.mu.Lock()
	m.records = append(m.records, projectResourceRecord{
		typ: args.TypeToken, name: args.Name, inputs: args.Inputs,
		pluginURL: args.RegisterRPC.GetPluginDownloadURL(), version: args.RegisterRPC.GetVersion(),
	})
	m.mu.Unlock()
	return args.Name + "_id", outputs, nil
}

func (m *projectRecordingMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func (m *projectRecordingMocks) findType(typ string) *projectResourceRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.records {
		if m.records[i].typ == typ {
			return &m.records[i]
		}
	}
	return nil
}

func (m *projectRecordingMocks) findAllType(typ string) []projectResourceRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	var results []projectResourceRecord
	for _, r := range m.records {
		if r.typ == typ {
			results = append(results, r)
		}
	}
	return results
}

func strPtr(s string) *string { return &s }

func TestBuildProjectCreatesServerlessContainerResources(t *testing.T) {
	mocks := &projectRecordingMocks{}
	image := "nginx:latest"
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := buildProject(ctx, "demo", ProjectInputs{
			Etag: "etag-1",
			Services: compose.Services{
				"app": {
					Image: compose.ImageFromPtr(&image),
					Ports: []compose.ServicePortConfig{{
						Target:      80,
						Mode:        compose.PortModeIngress,
						AppProtocol: compose.PortAppProtocolHTTP,
					}},
				},
			},
		}, common.PluginIdentityFrom(nil))
		return err
	}, pulumi.WithMocks("proj", "stack", mocks))

	require.NoError(t, err)
	require.NotNil(t, mocks.findType("scaleway:containers/namespace:Namespace"))
	require.NotNil(t, mocks.findType("scaleway:network/privateNetwork:PrivateNetwork"))
	container := mocks.findType("scaleway:containers/container:Container")
	require.NotNil(t, container)
	assert.Equal(t, "nginx:latest", container.inputs[resource.PropertyKey("registryImage")].StringValue())
	assert.InDelta(t, 80, container.inputs[resource.PropertyKey("port")].NumberValue(), 0)
	assert.Equal(t, "public", container.inputs[resource.PropertyKey("privacy")].StringValue())
	assert.False(t, container.inputs[resource.PropertyKey("privateNetworkId")].IsNull())
	env := container.inputs[resource.PropertyKey("environmentVariables")].ObjectValue()
	assert.Equal(t, "app", env[resource.PropertyKey("DEFANG_SERVICE")].StringValue())
	assert.Equal(t, "etag-1", env[resource.PropertyKey("DEFANG_ETAG")].StringValue())
}

func TestBuildProjectMultiServiceWithPostgres(t *testing.T) {
	mocks := &projectRecordingMocks{}
	image := "myapp:latest"
	workerImage := "worker:latest"
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := buildProject(ctx, "demo", ProjectInputs{
			Etag: "etag-2",
			Services: compose.Services{
				"web": {
					Image: compose.ImageFromPtr(&image),
					Ports: []compose.ServicePortConfig{{
						Target:      8080,
						Mode:        compose.PortModeIngress,
						AppProtocol: compose.PortAppProtocolHTTP,
					}},
					DependsOn: map[string]compose.ServiceDependency{
						"db": {},
					},
				},
				"worker": {
					Image: compose.ImageFromPtr(&workerImage),
					// No ports = private background worker
				},
				"db": {
					Postgres: &compose.PostgresConfig{},
					Environment: compose.Environment{
						"POSTGRES_PASSWORD": pulumi.String("Secret123!"),
					},
				},
			},
		}, common.PluginIdentityFrom(nil))
		return err
	}, pulumi.WithMocks("proj", "stack", mocks))

	require.NoError(t, err)

	// Verify shared infra was created
	require.NotNil(t, mocks.findType("scaleway:containers/namespace:Namespace"))
	require.NotNil(t, mocks.findType("scaleway:network/privateNetwork:PrivateNetwork"))

	// Verify postgres was created
	require.NotNil(t, mocks.findType("scaleway:databases/instance:Instance"))

	// Verify both containers were created
	containers := mocks.findAllType("scaleway:containers/container:Container")
	require.Len(t, containers, 2)

	// Find web (public) and worker (private)
	var webContainer, workerContainer *projectResourceRecord
	for i := range containers {
		switch containers[i].name {
		case "web":
			webContainer = &containers[i]
		case "worker":
			workerContainer = &containers[i]
		}
	}
	require.NotNil(t, webContainer, "web container should be created")
	require.NotNil(t, workerContainer, "worker container should be created")

	assert.Equal(t, "public", webContainer.inputs[resource.PropertyKey("privacy")].StringValue())
	assert.Equal(t, "private", workerContainer.inputs[resource.PropertyKey("privacy")].StringValue())
}

func TestBuildProjectCreatesBuildResourceForBuildOnlyServices(t *testing.T) {
	mocks := &projectRecordingMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := buildProject(ctx, "demo", ProjectInputs{
			Services: compose.Services{
				"app": {
					Build: &compose.BuildConfig{Context: pulumi.String(".")},
				},
			},
		}, common.PluginIdentityFrom(nil))
		return err
	}, pulumi.WithMocks("proj", "stack", mocks))

	require.NoError(t, err)
	build := mocks.findType("defang-scaleway:index:Build")
	assert.NotNil(t, build, "a Build resource should be created for build-only services")
}

func TestProjectBuildCarriesPluginIdentity(t *testing.T) {
	for _, tt := range []struct {
		name, url, version string
		opts               []pulumi.ResourceOption
	}{
		{name: "default", url: common.PluginDownloadURL},
		{name: "explicit", url: "https://plugins.example.com", version: "2.7.1",
			opts: []pulumi.ResourceOption{
				pulumi.PluginDownloadURL("https://plugins.example.com"), pulumi.Version("v2.7.1"),
			}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mocks := &projectRecordingMocks{}
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				_, err := (&Project{}).Construct(ctx, "demo", "defang-scaleway:index:Project", ProjectInputs{
					Services: compose.Services{
						"app": {Build: &compose.BuildConfig{Context: pulumi.String("s3://bucket/source")}},
					},
				}, pulumi.Composite(tt.opts...))
				return err
			}, pulumi.WithMocks("proj", "stack", mocks))
			require.NoError(t, err)
			build := mocks.findType("defang-scaleway:index:Build")
			require.NotNil(t, build)
			assert.Equal(t, tt.url, build.pluginURL)
			assert.Equal(t, tt.version, build.version)
			cloud := mocks.findType("scaleway:containers/container:Container")
			require.NotNil(t, cloud)
			assert.NotEqual(t, tt.url, cloud.pluginURL, "Defang identity must not leak to cloud resources")
		})
	}
}
