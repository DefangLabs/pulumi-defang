package compose

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestBuildConfigUnmarshalYAML(t *testing.T) {
	input := `
context: ./src
dockerfile: Dockerfile.prod
args:
  GO_VERSION: "1.22"
shm_size: "256m"
target: builder
`
	var bc BuildConfig
	require.NoError(t, yaml.Unmarshal([]byte(input), &bc))

	// Context should be a pulumi.String, not a raw Go string
	assert.Implements(t, (*pulumi.StringInput)(nil), bc.Context)
	assert.Equal(t, "Dockerfile.prod", *bc.Dockerfile)
	assert.Equal(t, map[string]string{"GO_VERSION": "1.22"}, bc.Args)
	assert.Equal(t, "256m", *bc.ShmSize)
	assert.Equal(t, "builder", *bc.Target)
}

func TestStringInputCannotUnmarshalYAML(t *testing.T) {
	// Demonstrates that yaml.Unmarshal into a pulumi.StringInput field
	// panics — the yaml decoder tries to assign a string to the interface
	// and reflect.Set rejects it. This is why BuildConfig needs a custom
	// UnmarshalYAML method.
	type raw struct {
		Value pulumi.StringInput `yaml:"value"`
	}
	var r raw
	assert.Panics(t, func() {
		_ = yaml.Unmarshal([]byte(`value: hello`), &r)
	}, "pulumi.StringInput should panic when unmarshaled from yaml")
}

func TestProjectUnmarshalYAML(t *testing.T) {
	input := `name: my-project
services:
  web:
    image: nginx:latest
    ports:
      - target: 80
        mode: ingress
    build:
      context: ./app
      dockerfile: Dockerfile
    environment:
      PORT: "8080"
      CONFIG:
    deploy:
      replicas: 2
      resources:
        reservations:
          cpus: 0.5
          memory: "512Mi"
  db:
    image: postgres:16
    x-defang-postgres:
      allow-downtime: true
networks:
  backend:
    internal: true
`
	var p Project
	require.NoError(t, yaml.Unmarshal([]byte(input), &p))

	require.Len(t, p.Services, 2)
	require.Contains(t, p.Services, "web")
	require.Contains(t, p.Services, "db")

	web := p.Services["web"]
	assert.Equal(t, "nginx:latest", *web.Image)
	require.Len(t, web.Ports, 1)
	assert.Equal(t, int32(80), web.Ports[0].Target)
	assert.EqualValues(t, "ingress", web.Ports[0].Mode)
	require.NotNil(t, web.Build)
	assert.Implements(t, (*pulumi.StringInput)(nil), web.Build.Context)
	assert.Equal(t, "Dockerfile", *web.Build.Dockerfile)
	port := "8080"
	assert.Equal(t, map[string]*string{"PORT": &port, "CONFIG": nil}, web.Environment)
	require.NotNil(t, web.Deploy)
	assert.Equal(t, int32(2), *web.Deploy.Replicas)
	require.NotNil(t, web.Deploy.Resources)
	require.NotNil(t, web.Deploy.Resources.Reservations)
	assert.InDelta(t, 0.5, *web.Deploy.Resources.Reservations.CPUs, 0.001)
	assert.Equal(t, "512Mi", *web.Deploy.Resources.Reservations.Memory)

	db := p.Services["db"]
	assert.Equal(t, "postgres:16", *db.Image)
	require.NotNil(t, db.Postgres)
	assert.True(t, *db.Postgres.AllowDowntime)

	require.Contains(t, p.Networks, NetworkID("backend"))
	assert.True(t, p.Networks[NetworkID("backend")].Internal)
}
