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
	assert.Equal(t, "http1", containerProtocol(compose.ServiceConfig{
		Ports: []compose.ServicePortConfig{{Target: 80, AppProtocol: compose.PortAppProtocolHTTP}},
	}))
	assert.Equal(t, "h2c", containerProtocol(compose.ServiceConfig{
		Ports: []compose.ServicePortConfig{{Target: 80, AppProtocol: compose.PortAppProtocolHTTP2}},
	}))
	assert.Equal(t, "h2c", containerProtocol(compose.ServiceConfig{
		Ports: []compose.ServicePortConfig{{Target: 80, AppProtocol: compose.PortAppProtocolGRPC}},
	}))
}

func TestValidateContainerServiceRejectsUnsupportedInputs(t *testing.T) {
	t.Run("host port", func(t *testing.T) {
		err := validateContainerService(compose.ServiceConfig{
			Ports: []compose.ServicePortConfig{{Mode: compose.PortModeHost, Target: 8080}},
		})
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
	secretRef := "${" + "CONFIG_VALUE}"
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		namespace, err := containers.NewNamespace(ctx, "ns", &containers.NamespaceArgs{})
		if err != nil {
			return err
		}
		configProvider := &mockConfigProvider{
			values: map[string]string{"CONFIG_VALUE": "secret-value"},
		}
		_, err = CreateContainerService(ctx, configProvider, "app", pulumi.String("nginx:latest"), compose.ServiceConfig{
			Ports: []compose.ServicePortConfig{{Target: 8080, AppProtocol: compose.PortAppProtocolGRPC}},
			Environment: map[string]*string{
				"CONFIG_VALUE": &secretRef,
				"MODE":         ptr("prod"),
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
	assert.InDelta(t, 8080, container.inputs[resource.PropertyKey("port")].NumberValue(), 0)
	assert.Equal(t, "h2c", container.inputs[resource.PropertyKey("protocol")].StringValue())
	assert.Equal(t, "public", container.inputs[resource.PropertyKey("privacy")].StringValue())
	env := container.inputs[resource.PropertyKey("environmentVariables")].ObjectValue()
	assert.Equal(t, "app", env[resource.PropertyKey("DEFANG_SERVICE")].StringValue())
	assert.Equal(t, "deploy-123", env[resource.PropertyKey("DEFANG_ETAG")].StringValue())
	assert.Equal(t, "prod", env[resource.PropertyKey("MODE")].StringValue())
	secretValue := container.inputs[resource.PropertyKey("secretEnvironmentVariables")]
	require.True(t, secretValue.IsSecret())
	secrets := secretValue.SecretValue().Element.ObjectValue()
	assert.Equal(t, "secret-value", secrets[resource.PropertyKey("CONFIG_VALUE")].StringValue())
	healthChecks := container.inputs[resource.PropertyKey("healthChecks")].ArrayValue()
	require.Len(t, healthChecks, 1)
	healthCheck := healthChecks[0].ObjectValue()
	assert.InDelta(t, 3, healthCheck[resource.PropertyKey("failureThreshold")].NumberValue(), 0)
	assert.Equal(t, "5s", healthCheck[resource.PropertyKey("interval")].StringValue())
}

func TestFindManagedHostInURL(t *testing.T) {
	hosts := map[string]pulumi.StringOutput{
		"postgres": {},
		"redis":    {},
	}

	tests := []struct {
		name    string
		raw     string
		wantSvc string
		wantOK  bool
	}{
		{"postgres URL with userinfo", "postgres://user:pass@postgres:5432/db", "postgres", true},
		{"redis URL", "redis://redis:6379", "redis", true},
		{"rediss URL with path", "rediss://default:secret@redis:6379/0", "redis", true},
		{"no scheme", "postgres:5432", "", false},
		{"plain value", "postgres", "", false},
		{"unknown host", "postgres://user:pass@unknown:5432/db", "", false},
		{"http URL", "http://postgres/health", "postgres", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, ok := findManagedHostInURL(tt.raw, hosts)
			assert.Equal(t, tt.wantSvc, svc)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}

func TestCreateContainerServicePrivateService(t *testing.T) {
	mocks := &recordingMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		namespace, err := containers.NewNamespace(ctx, "ns", &containers.NamespaceArgs{})
		if err != nil {
			return err
		}
		// A private service has no ports (background worker, consumer, etc.)
		_, err = CreateContainerService(
			ctx,
			&mockConfigProvider{},
			"worker",
			pulumi.String("myapp:latest"),
			compose.ServiceConfig{},
			&SharedInfra{Namespace: namespace, Region: "fr-par"},
		)
		return err
	}, pulumi.WithMocks("proj", "stack", mocks))

	require.NoError(t, err)
	container := mocks.findType("scaleway:containers/container:Container")
	require.NotNil(t, container)
	assert.Equal(t, "private", container.inputs[resource.PropertyKey("privacy")].StringValue())
}

func TestHealthShim(t *testing.T) {
	t.Run("no ports and command triggers shim", func(t *testing.T) {
		svc := compose.ServiceConfig{
			Command: []string{"npm", "run", "worker"},
		}
		assert.True(t, needsHealthShim(svc))
		script := healthShimScript(svc)
		assert.NotEmpty(t, script)
		assert.Contains(t, script, "exec npm run worker")
		assert.Contains(t, script, "node -e")
		assert.Contains(t, script, "python3 -c")
	})

	t.Run("no ports and entrypoint+command", func(t *testing.T) {
		svc := compose.ServiceConfig{
			Entrypoint: []string{"/usr/bin/env"},
			Command:    []string{"node", "worker.js"},
		}
		script := healthShimScript(svc)
		assert.Contains(t, script, "exec /usr/bin/env node worker.js")
	})

	t.Run("no ports and no command returns empty", func(t *testing.T) {
		svc := compose.ServiceConfig{}
		assert.True(t, needsHealthShim(svc))
		assert.Empty(t, healthShimScript(svc))
	})

	t.Run("has ports does not need shim", func(t *testing.T) {
		svc := compose.ServiceConfig{
			Ports:   []compose.ServicePortConfig{{Target: 3000, Mode: compose.PortModeIngress}},
			Command: []string{"npm", "start"},
		}
		assert.False(t, needsHealthShim(svc))
	})
}

func TestHealthShimContainsFallbacksForCommonBaseImages(t *testing.T) {
	script := healthShimScript(compose.ServiceConfig{Command: []string{"npm", "run", "worker"}})

	tests := []struct {
		name string
		want string
	}{
		{name: "node base images", want: "node -e"},
		{name: "python base images", want: "python3 -c"},
		{name: "legacy python base images", want: "python -c"},
		{name: "alpine busybox images", want: "nc -l -p"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, script, tt.want)
		})
	}
}

func TestHealthShimInjectedInContainer(t *testing.T) {
	mocks := &recordingMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		namespace, err := containers.NewNamespace(ctx, "ns", &containers.NamespaceArgs{})
		if err != nil {
			return err
		}
		_, err = CreateContainerService(
			ctx,
			&mockConfigProvider{},
			"worker",
			pulumi.String("myapp:latest"),
			compose.ServiceConfig{Command: []string{"npm", "run", "worker"}},
			&SharedInfra{Namespace: namespace, Region: "fr-par"},
		)
		return err
	}, pulumi.WithMocks("proj", "stack", mocks))

	require.NoError(t, err)
	container := mocks.findType("scaleway:containers/container:Container")
	require.NotNil(t, container)

	// Port should be set to default health shim port
	assert.InDelta(t, defaultHealthShimPort, container.inputs[resource.PropertyKey("port")].NumberValue(), 0)

	// Min scale must be 1 for portless workers (no HTTP traffic to wake them)
	assert.InDelta(t, 1, container.inputs[resource.PropertyKey("minScale")].NumberValue(), 0)

	// Commands should be ["/bin/sh", "-c"]
	cmds := container.inputs[resource.PropertyKey("commands")].ArrayValue()
	require.Len(t, cmds, 2)
	assert.Equal(t, "/bin/sh", cmds[0].StringValue())
	assert.Equal(t, "-c", cmds[1].StringValue())

	// Args should contain the shim script wrapping the original command
	args := container.inputs[resource.PropertyKey("args")].ArrayValue()
	require.Len(t, args, 1)
	assert.Contains(t, args[0].StringValue(), "exec npm run worker")
	assert.Contains(t, args[0].StringValue(), "node -e")
}

func TestContainerMinScaleWorkerAlwaysOne(t *testing.T) {
	// Portless worker must have min_scale=1 since Scaleway only wakes
	// containers on HTTP requests; queue consumers get no HTTP traffic.
	worker := compose.ServiceConfig{Command: []string{"npm", "run", "worker"}}
	assert.Equal(t, pulumi.IntPtr(1), containerMinScale(worker))

	// Service with ports can scale to zero (HTTP traffic wakes it)
	web := compose.ServiceConfig{
		Ports:   []compose.ServicePortConfig{{Target: 3000, Mode: compose.PortModeIngress}},
		Command: []string{"npm", "start"},
	}
	assert.Equal(t, pulumi.IntPtr(0), containerMinScale(web))
}

func TestShellJoin(t *testing.T) {
	assert.Equal(t, "npm run worker", shellJoin([]string{"npm", "run", "worker"}))
	assert.Equal(t, "echo 'hello world'", shellJoin([]string{"echo", "hello world"}))
	assert.Equal(t, "cmd 'it'\"'\"'s'", shellJoin([]string{"cmd", "it's"}))
}
