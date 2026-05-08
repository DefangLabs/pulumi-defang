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

func TestContainerPrivacy(t *testing.T) {
	// Explicit ingress is public
	assert.Equal(t, "public", containerPrivacy(compose.ServiceConfig{
		Ports: []compose.ServicePortConfig{{Target: 80, Mode: compose.PortModeIngress}},
	}))
	// Empty mode defaults to ingress per Compose spec, so also public
	assert.Equal(t, "public", containerPrivacy(compose.ServiceConfig{
		Ports: []compose.ServicePortConfig{{Target: 80, Mode: ""}},
	}))
	// No ports means private (background worker, consumer, etc.)
	assert.Equal(t, "private", containerPrivacy(compose.ServiceConfig{}))
}

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

func TestValidateContainerServiceRejectsUnsupportedInputs(t *testing.T) {
	t.Run("host port", func(t *testing.T) {
		err := validateContainerService(compose.ServiceConfig{Ports: []compose.ServicePortConfig{{Mode: compose.PortModeHost, Target: 8080}}})
		require.ErrorIs(t, err, ErrContainerUnsupported)
		assert.Contains(t, err.Error(), "host-mode ports")
	})

	t.Run("multiple ingress ports", func(t *testing.T) {
		err := validateContainerService(compose.ServiceConfig{Ports: []compose.ServicePortConfig{
			{Mode: compose.PortModeIngress, Target: 8080},
			{Mode: compose.PortModeIngress, Target: 9092},
		}})
		require.ErrorIs(t, err, ErrContainerUnsupported)
		assert.Contains(t, err.Error(), "exactly one public port")
	})

	t.Run("reserved port", func(t *testing.T) {
		err := validateContainerService(compose.ServiceConfig{Ports: []compose.ServicePortConfig{{Target: 8008}}})
		require.ErrorIs(t, err, ErrContainerUnsupported)
		assert.Contains(t, err.Error(), "reserved")
	})

	t.Run("reserved env", func(t *testing.T) {
		err := validateContainerService(compose.ServiceConfig{Environment: map[string]*string{"SCW_REGION": ptr("fr-par")}})
		require.ErrorIs(t, err, ErrContainerUnsupported)
		assert.Contains(t, err.Error(), "reserved")
	})

	t.Run("unsupported llm", func(t *testing.T) {
		err := validateContainerService(compose.ServiceConfig{LLM: &compose.LlmConfig{}})
		require.ErrorIs(t, err, ErrContainerUnsupported)
		assert.Contains(t, err.Error(), "LLM")
	})

	t.Run("unsupported platform", func(t *testing.T) {
		err := validateContainerService(compose.ServiceConfig{Platform: ptr("linux/arm64")})
		require.ErrorIs(t, err, ErrContainerUnsupported)
		assert.Contains(t, err.Error(), "linux/amd64")
	})
}

func TestValidateContainerServiceRejectsInvalidResourceLimits(t *testing.T) {
	cpus := 8.0
	err := validateContainerService(compose.ServiceConfig{
		Deploy: &compose.DeployConfig{Resources: &compose.Resources{Reservations: &compose.ResourceConfig{CPUs: &cpus}}},
	})
	require.ErrorIs(t, err, ErrContainerUnsupported)
	assert.Contains(t, err.Error(), "CPU limit")

	memory := "13g"
	err = validateContainerService(compose.ServiceConfig{
		Deploy: &compose.DeployConfig{Resources: &compose.Resources{Reservations: &compose.ResourceConfig{Memory: &memory}}},
	})
	require.ErrorIs(t, err, ErrContainerUnsupported)
	assert.Contains(t, err.Error(), "memory limit")
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
			HealthCheck: &compose.HealthCheckConfig{
				Test:            []string{"CMD", "curl", "-f", "http://localhost:8080/"},
				IntervalSeconds: 5,
				Retries:         3,
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
	healthChecks := container.inputs[resource.PropertyKey("healthChecks")].ArrayValue()
	require.Len(t, healthChecks, 1)
	healthCheck := healthChecks[0].ObjectValue()
	assert.Equal(t, float64(3), healthCheck[resource.PropertyKey("failureThreshold")].NumberValue())
	assert.Equal(t, "5s", healthCheck[resource.PropertyKey("interval")].StringValue())
}

func TestCreateContainerServicePrivateService(t *testing.T) {
	mocks := &recordingMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		namespace, err := containers.NewNamespace(ctx, "ns", &containers.NamespaceArgs{})
		if err != nil {
			return err
		}
		// A private service has no ports (background worker, consumer, etc.)
		_, err = CreateContainerService(ctx, &mockConfigProvider{}, "worker", pulumi.String("myapp:latest"), compose.ServiceConfig{}, &SharedInfra{Namespace: namespace, Region: "fr-par"})
		return err
	}, pulumi.WithMocks("proj", "stack", mocks))

	require.NoError(t, err)
	container := mocks.findType("scaleway:containers/container:Container")
	require.NotNil(t, container)
	assert.Equal(t, "private", container.inputs[resource.PropertyKey("privacy")].StringValue())
}
