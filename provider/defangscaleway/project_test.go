package defangscaleway

import (
	"sync"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type projectResourceRecord struct {
	typ    string
	name   string
	inputs resource.PropertyMap
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
	switch string(args.TypeToken) {
	case "scaleway:containers/namespace:Namespace":
		outputs[resource.PropertyKey("registryEndpoint")] = resource.NewStringProperty("rg.fr-par.scw.cloud/defang")
	case "scaleway:containers/container:Container":
		outputs[resource.PropertyKey("domainName")] = resource.NewStringProperty("https://app.functions.fnc.fr-par.scw.cloud")
	case "scaleway:network/privateNetwork:PrivateNetwork":
		outputs[resource.PropertyKey("id")] = resource.NewStringProperty(args.Name + "_id")
	}
	m.mu.Lock()
	m.records = append(m.records, projectResourceRecord{typ: string(args.TypeToken), name: args.Name, inputs: args.Inputs})
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

func TestBuildProjectCreatesServerlessContainerResources(t *testing.T) {
	mocks := &projectRecordingMocks{}
	image := "nginx:latest"
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := buildProject(ctx, "demo", ProjectInputs{
			Etag: "etag-1",
			Services: compose.Services{
				"app": {
					Image: &image,
					Ports: []compose.ServicePortConfig{{
						Target:      80,
						Mode:        compose.PortModeIngress,
						AppProtocol: compose.PortAppProtocolHTTP,
					}},
				},
			},
		})
		return err
	}, pulumi.WithMocks("proj", "stack", mocks))

	require.NoError(t, err)
	require.NotNil(t, mocks.findType("scaleway:containers/namespace:Namespace"))
	require.NotNil(t, mocks.findType("scaleway:network/privateNetwork:PrivateNetwork"))
	container := mocks.findType("scaleway:containers/container:Container")
	require.NotNil(t, container)
	assert.Equal(t, "nginx:latest", container.inputs[resource.PropertyKey("registryImage")].StringValue())
	assert.Equal(t, float64(80), container.inputs[resource.PropertyKey("port")].NumberValue())
	assert.Equal(t, "public", container.inputs[resource.PropertyKey("privacy")].StringValue())
	assert.False(t, container.inputs[resource.PropertyKey("privateNetworkId")].IsNull())
	env := container.inputs[resource.PropertyKey("environmentVariables")].ObjectValue()
	assert.Equal(t, "app", env[resource.PropertyKey("DEFANG_SERVICE")].StringValue())
	assert.Equal(t, "etag-1", env[resource.PropertyKey("DEFANG_ETAG")].StringValue())
}

func TestBuildProjectRejectsBuildOnlyServices(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := buildProject(ctx, "demo", ProjectInputs{
			Services: compose.Services{
				"app": {
					Build: &compose.BuildConfig{Context: pulumi.String(".")},
				},
			},
		})
		return err
	}, pulumi.WithMocks("proj", "stack", &projectRecordingMocks{}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires a pre-built image")
}
