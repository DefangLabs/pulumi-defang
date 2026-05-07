package scaleway

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/containers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerSizing(t *testing.T) {
	assert.Equal(t, 140, containerCPULimit(0))
	assert.Equal(t, 250, containerCPULimit(0.25))
	assert.Equal(t, 1000, containerCPULimit(1))
	assert.Equal(t, 256, containerMemoryLimit(0))
	assert.Equal(t, 512, containerMemoryLimit(512))
}

func TestContainerProtocol(t *testing.T) {
	assert.Equal(t, "http1", containerProtocol(compose.ServiceConfig{}))
	assert.Equal(t, "http1", containerProtocol(compose.ServiceConfig{Ports: []compose.ServicePortConfig{{Target: 80, AppProtocol: compose.PortAppProtocolHTTP}}}))
	assert.Equal(t, "h2c", containerProtocol(compose.ServiceConfig{Ports: []compose.ServicePortConfig{{Target: 80, AppProtocol: compose.PortAppProtocolHTTP2}}}))
	assert.Equal(t, "h2c", containerProtocol(compose.ServiceConfig{Ports: []compose.ServicePortConfig{{Target: 80, AppProtocol: compose.PortAppProtocolGRPC}}}))
}

func TestCreateContainerServiceMapsInputs(t *testing.T) {
	mocks := &recordingMocks{}
	secretRef := "${API_KEY}"
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		namespace, err := containers.NewNamespace(ctx, "ns", &containers.NamespaceArgs{})
		if err != nil {
			return err
		}
		_, err = CreateContainerService(ctx, &mockConfigProvider{values: map[string]string{"API_KEY": "secret-value"}}, "app", pulumi.String("nginx:latest"), compose.ServiceConfig{
			Ports: []compose.ServicePortConfig{{Target: 8080, AppProtocol: compose.PortAppProtocolGRPC}},
			Environment: map[string]*string{
				"API_KEY": &secretRef,
				"MODE":    ptr("prod"),
			},
		}, &SharedInfra{Namespace: namespace, Region: "fr-par", Etag: "deploy-123"})
		return err
	}, pulumi.WithMocks("proj", "stack", mocks))

	require.NoError(t, err)
	container := mocks.findType("scaleway:containers/container:Container")
	require.NotNil(t, container)
	assert.Equal(t, "nginx:latest", container.inputs[resource.PropertyKey("registryImage")].StringValue())
	assert.Equal(t, float64(8080), container.inputs[resource.PropertyKey("port")].NumberValue())
	assert.Equal(t, "h2c", container.inputs[resource.PropertyKey("protocol")].StringValue())
	assert.Equal(t, "public", container.inputs[resource.PropertyKey("privacy")].StringValue())
	env := container.inputs[resource.PropertyKey("environmentVariables")].ObjectValue()
	assert.Equal(t, "app", env[resource.PropertyKey("DEFANG_SERVICE")].StringValue())
	assert.Equal(t, "deploy-123", env[resource.PropertyKey("DEFANG_ETAG")].StringValue())
	assert.Equal(t, "prod", env[resource.PropertyKey("MODE")].StringValue())
	secretValue := container.inputs[resource.PropertyKey("secretEnvironmentVariables")]
	require.True(t, secretValue.IsSecret())
	secrets := secretValue.SecretValue().Element.ObjectValue()
	assert.Equal(t, "secret-value", secrets[resource.PropertyKey("API_KEY")].StringValue())
}
