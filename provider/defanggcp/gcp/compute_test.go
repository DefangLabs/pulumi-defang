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
				out := getCloudInitConfig(
					"api", pulumi.String("img:latest"), svc, "us-central1", tt.etag, "", "", tt.fqdn, "",
					false, nil, containerSecretPlan{inline: svc.Environment}, nil)
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

// getCloudInitConfig must stamp the defang-* LogEntry labels into the COS
// fluent-bit config so the Defang CLI's (and Fabric's) Cloud Logging tail
// queries match Compute Engine logs. Values are SafeLabelValue-normalized;
// empty etag/project/stack are omitted.
func TestGetCloudInitConfigLogLabels(t *testing.T) {
	svc := compose.ServiceConfig{
		Ports: []compose.ServicePortConfig{{Target: 8080, Mode: compose.PortModeHost}},
	}

	tests := []struct {
		name                     string
		etag, projectName, stack string
		want                     string
	}{
		{
			name:        "all set, normalized",
			etag:        "Etag123",
			projectName: "My Project",
			stack:       "beta",
			want: `echo "    labels defang-etag=etag123,defang-project=my-project,defang-service=api,defang-stack=beta"` +
				` >> /etc/fluent-bit/fluent-bit.conf`,
		},
		{
			name: "empty etag/project/stack omitted",
			want: `echo "    labels defang-service=api" >> /etc/fluent-bit/fluent-bit.conf`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cloudInit string
			var wg sync.WaitGroup
			wg.Add(1)
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				out := getCloudInitConfig(
					"api", pulumi.String("img:latest"), svc, "us-central1", tt.etag, tt.projectName, tt.stack, "", "",
					false, nil, containerSecretPlan{inline: svc.Environment}, nil)
				out.ApplyT(func(s string) string {
					defer wg.Done()
					cloudInit = s
					return s
				})
				return nil
			}, pulumi.WithMocks("proj", "stack", testMocks{}))
			require.NoError(t, err)
			wg.Wait()

			assert.Contains(t, cloudInit, tt.want)
			assert.Contains(t, cloudInit, "systemctl restart fluent-bit")
		})
	}
}

// A run-once sidecar (restart: "no") must become a oneshot unit started before
// the main service, with the main container mounting its volumes via
// --volumes-from; '%' in env values must survive the pulumi.Sprintf pass.
// The sidecar image is an Output (e.g. a digest resolved at apply time) to
// cover dynamic sidecar images.
func TestGetCloudInitConfigSidecars(t *testing.T) {
	handlerImageURI := "region-docker.pkg.dev/proj/repo/handler@sha256:0123456789abcdef"
	handlerImage := pulumi.String(handlerImageURI).ToStringOutput() // dynamic, StaticImage() == nil
	percentVal := "100%"
	svc := compose.ServiceConfig{
		Entrypoint:  []string{"/handler/handler"},
		VolumesFrom: []string{"handler"},
		DependsOn:   compose.DependsOnConfig{"handler": {}},
		Environment: compose.Environment{"RATIO": pulumi.String(percentVal)},
	}
	sidecars := map[string]compose.ServiceConfig{
		"handler": {
			Image:      handlerImage,
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
		out := getCloudInitConfig("cd", pulumi.String("img:latest"), svc, "us-central1", "", "", "", "", "",
			true, sidecars, containerSecretPlan{inline: svc.Environment},
			map[string]containerSecretPlan{"handler": {inline: sidecars["handler"].Environment}})
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
	assert.Contains(t, cloudInit, handlerImageURI)
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

// mockSecretConfigProvider resolves bare ${VAR} / null env refs to a
// deterministic Secret Manager ID so the classify/cloud-init paths can be
// tested without a live backend.
type mockSecretConfigProvider struct{ prefix string }

func (m *mockSecretConfigProvider) GetConfigValue(
	_ *pulumi.Context, key string, _ ...pulumi.InvokeOption,
) pulumi.StringOutput {
	return pulumi.String("val-" + key).ToStringOutput()
}

func (m *mockSecretConfigProvider) GetSecretRef(
	_ *pulumi.Context, key string, _ ...pulumi.InvokeOption,
) (string, error) {
	return m.prefix + key, nil
}

// classifyComputeSecretEnv routes bare ${VAR} and null "KEY:" values to native
// secret refs, and leaves literals and interpolated (mixed) values inline.
func TestClassifyComputeSecretEnv(t *testing.T) {
	env := compose.Environment{
		"PLAIN":  pulumi.String("literal"),
		"BARE":   pulumi.String("${SECRET_ONE}"),
		"NULLED": nil,
		"MIXED":  pulumi.String("pre-${SECRET_TWO}-post"),
	}
	cp := &mockSecretConfigProvider{prefix: "Defang_proj_stack_"}

	plan := classifyComputeSecretEnv(nil, cp, env)

	// secret refs are sorted by env key: BARE, NULLED
	require.Len(t, plan.secretRefs, 2)
	assert.Equal(t, computeSecretEnv{envKey: "BARE", secretID: "Defang_proj_stack_SECRET_ONE"}, plan.secretRefs[0])
	assert.Equal(t, computeSecretEnv{envKey: "NULLED", secretID: "Defang_proj_stack_NULLED"}, plan.secretRefs[1])
	// literals and mixed interpolation stay inline
	assert.Contains(t, plan.inline, "PLAIN")
	assert.Contains(t, plan.inline, "MIXED")
	assert.NotContains(t, plan.inline, "BARE")
	assert.NotContains(t, plan.inline, "NULLED")

	// with no config provider, everything is inlined
	nilPlan := classifyComputeSecretEnv(nil, nil, env)
	assert.Nil(t, nilPlan.secretRefs)
	assert.Equal(t, env, nilPlan.inline)
}

// secretFetchScript emits a COS-compatible boot fetch (metadata token + Secret
// Manager REST API) writing a tmpfs env-file, plus the ExecStartPre + env-file
// flag that consume it.
func TestSecretFetchScript(t *testing.T) {
	refs := []computeSecretEnv{
		{envKey: "DB", secretID: "Defang_p_s_DBPASS"},
		{envKey: "API", secretID: "Defang_p_s_APIKEY"},
	}
	wf, pre, flag := secretFetchScript("my-proj", "svc", refs)

	assert.Equal(t, "ExecStartPre=/opt/defang/svc-secrets.sh", pre)
	assert.Equal(t, "--env-file /run/defang/svc.env", flag)
	assert.Contains(t, wf, "path: /opt/defang/svc-secrets.sh")
	assert.Contains(t, wf, `permissions: "0700"`)
	assert.Contains(t, wf, "metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token")
	assert.Contains(t, wf, "https://secretmanager.googleapis.com/v1/projects/my-proj/secrets/")
	assert.Contains(t, wf, `printf '%s=%s\n' 'DB' "$(sm 'Defang_p_s_DBPASS')"`)
	assert.Contains(t, wf, `printf '%s=%s\n' 'API' "$(sm 'Defang_p_s_APIKEY')"`)
	assert.Contains(t, wf, "} > /run/defang/svc.env")

	// no refs -> no output
	wf2, pre2, flag2 := secretFetchScript("p", "svc", nil)
	assert.Empty(t, wf2)
	assert.Empty(t, pre2)
	assert.Empty(t, flag2)
}

// getCloudInitConfig must boot-fetch secret env (not inline it) while still
// inlining plain values, and wire up the ExecStartPre + --env-file.
func TestGetCloudInitConfigSecrets(t *testing.T) {
	svc := compose.ServiceConfig{
		Ports: []compose.ServicePortConfig{{Target: 8080, Mode: compose.PortModeHost}},
	}
	plan := containerSecretPlan{
		inline: compose.Environment{"PLAIN": pulumi.String("x")},
		secretRefs: []computeSecretEnv{
			{envKey: "SECRET_ENV", secretID: "Defang_proj_stack_SECRET_ENV"},
		},
	}

	var cloudInit string
	var wg sync.WaitGroup
	wg.Add(1)
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		out := getCloudInitConfig(
			"api", pulumi.String("img:latest"), svc, "us-central1", "", "", "", "", "gcp-proj",
			false, nil, plan, nil)
		out.ApplyT(func(s string) string {
			defer wg.Done()
			cloudInit = s
			return s
		})
		return nil
	}, pulumi.WithMocks("proj", "stack", testMocks{}))
	require.NoError(t, err)
	wg.Wait()

	assert.Contains(t, cloudInit, "ExecStartPre=/opt/defang/api-secrets.sh")
	assert.Contains(t, cloudInit, "--env-file /run/defang/api.env")
	assert.Contains(t, cloudInit, "https://secretmanager.googleapis.com/v1/projects/gcp-proj/secrets/")
	assert.Contains(t, cloudInit, `printf '%s=%s\n' 'SECRET_ENV' "$(sm 'Defang_proj_stack_SECRET_ENV')"`)
	// plain value inlined, secret value NOT inlined
	assert.Contains(t, cloudInit, `-e "PLAIN=x"`)
	assert.NotContains(t, cloudInit, `-e "SECRET_ENV`)
	// no leftover format artifacts
	assert.NotContains(t, cloudInit, "%!")
}
