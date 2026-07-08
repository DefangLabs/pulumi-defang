package gcp

import (
	"strings"
	"sync"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getCloudInitConfig must inject the Defang runtime env vars into the
// `docker run` command, mirroring the Cloud Run path. DEFANG_SERVICE is always
// present; DEFANG_ETAG/DEFANG_FQDN only when non-empty.
func TestGetCloudInitConfigDefangEnv(t *testing.T) {
	svc := compose.ServiceConfig{
		Ports: []compose.ServicePortConfig{{Target: 8080, Mode: compose.PortModeHost}},
	}

	tests := []struct {
		name    string
		etag    string
		fqdn    string
		present []string
		absent  []string
	}{
		{
			name:    "all set",
			etag:    "etag123",
			fqdn:    "api.google.internal",
			present: []string{`-e "DEFANG_SERVICE=api"`, `-e "DEFANG_ETAG=etag123"`, `-e "DEFANG_FQDN=api.google.internal"`},
		},
		{
			name:    "no etag or fqdn",
			present: []string{`-e "DEFANG_SERVICE=api"`},
			absent:  []string{"DEFANG_ETAG", "DEFANG_FQDN"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cloudInit string
			var wg sync.WaitGroup
			wg.Add(1)
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				out := getCloudInitConfig("api", pulumi.String("img:latest"), svc, "us-central1", tt.etag, tt.fqdn, false, nil)
				out.ApplyT(func(s string) string {
					defer wg.Done()
					cloudInit = s
					return s
				})
				return nil
			}, pulumi.WithMocks("proj", "stack", testMocks{}))
			require.NoError(t, err)
			wg.Wait()

			for _, want := range tt.present {
				assert.Contains(t, cloudInit, want)
			}
			for _, notWant := range tt.absent {
				assert.NotContains(t, cloudInit, notWant)
			}
		})
	}
}

// A run-once sidecar (restart: "no") must become a oneshot unit started before
// the main service, with the main container mounting its volumes via
// --volumes-from; '%' in env values must survive the pulumi.Sprintf pass.
func TestGetCloudInitConfigSidecars(t *testing.T) {
	handlerImage := "region-docker.pkg.dev/proj/repo/handler:1"
	percentVal := "100%"
	svc := compose.ServiceConfig{
		Entrypoint:  []string{"/handler/handler"},
		VolumesFrom: []string{"handler"},
		DependsOn:   compose.DependsOnConfig{"handler": {}},
		Environment: map[string]*string{"RATIO": &percentVal},
	}
	sidecars := map[string]compose.ServiceConfig{
		"handler": {
			Image:      &handlerImage,
			Entrypoint: []string{"true"},
			Restart:    "no",
			Volumes: []compose.ServiceVolumeConfig{
				{Source: "handler", Target: "/handler", ReadOnly: true},
				{Source: "pulumi-plugins", Target: "/root/.pulumi/plugins"},
			},
		},
	}

	var cloudInit string
	var wg sync.WaitGroup
	wg.Add(1)
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		out := getCloudInitConfig("cd", pulumi.String("img:latest"), svc, "us-central1", "", "", true, sidecars)
		out.ApplyT(func(s string) string {
			defer wg.Done()
			cloudInit = s
			return s
		})
		return nil
	}, pulumi.WithMocks("proj", "stack", testMocks{}))
	require.NoError(t, err)
	wg.Wait()

	// sidecar unit: oneshot, no --rm, named container, volume mounts
	assert.Contains(t, cloudInit, "/etc/systemd/system/cd-handler.service")
	assert.Contains(t, cloudInit, "Type=oneshot")
	assert.Contains(t, cloudInit, "RemainAfterExit=yes")
	assert.Contains(t, cloudInit,
		"--name=handler --entrypoint true -v handler:/handler:ro -v pulumi-plugins:/root/.pulumi/plugins")
	assert.Contains(t, cloudInit, handlerImage)
	// main unit: ordered after the sidecar, volumes-from it
	assert.Contains(t, cloudInit, "Requires=cd-handler.service")
	assert.Contains(t, cloudInit, "After=cd-handler.service")
	assert.Contains(t, cloudInit, "--volumes-from handler")
	// sidecar started before the main service
	handlerStart := strings.Index(cloudInit, "systemctl start cd-handler.service")
	mainStart := strings.Index(cloudInit, "systemctl start cd.service")
	require.Positive(t, handlerStart)
	require.Positive(t, mainStart)
	assert.Less(t, handlerStart, mainStart)
	// '%' escaping: env value intact, image substituted
	assert.Contains(t, cloudInit, `-e "RATIO=100%"`)
	assert.Contains(t, cloudInit, "img:latest")
	assert.NotContains(t, cloudInit, "%!")
}
