package gcp

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const computeDatabaseURLKey = "DATABASE_URL"

// namedResourceMocks records the Pulumi logical name of every resource it
// creates, keyed by Pulumi type token. Physical naming (length, case,
// project/stack scoping) is Pulumi autonaming's job, configured in
// cd/config.go -- these resources deliberately don't override Name.
type namedResourceMocks struct {
	mu    sync.Mutex
	names map[string]string
}

func (m *namedResourceMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	outputs := args.Inputs.Copy()
	m.mu.Lock()
	if m.names == nil {
		m.names = map[string]string{}
	}
	m.names[args.TypeToken] = args.Name
	m.mu.Unlock()
	if _, ok := outputs["selfLink"]; !ok {
		outputs["selfLink"] = resource.NewStringProperty("https://compute.googleapis.com/" + args.Name)
	}
	return args.Name + "_id", outputs, nil
}

func (m *namedResourceMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

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

// classifyComputeSecretEnv routes bare ${VAR}, null "KEY:", and mixed values
// that interpolate config secrets to native secret refs. Mixed values receive
// their own derived secret, rather than being embedded in instance metadata.
func TestClassifyComputeSecretEnv(t *testing.T) {
	env := compose.Environment{
		"PLAIN":  pulumi.String("literal"),
		"BARE":   pulumi.String("${SECRET_ONE}"),
		"NULLED": nil,
		"MIXED":  pulumi.String("pre-${SECRET_TWO}-post"),
	}
	cp := &mockSecretConfigProvider{prefix: "Defang_proj_stack_"}

	spy := &secretManagerSpy{}
	var plan containerSecretPlan
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		var err error
		plan, err = classifyComputeSecretEnv(ctx, cp, "worker", env, nil)
		return err
	}, pulumi.WithMocks("proj", "stack", spy))
	require.NoError(t, err)

	// Secret refs are sorted by env key: BARE, MIXED, NULLED.
	require.Len(t, plan.secretRefs, 3)
	assert.Equal(t, "BARE", plan.secretRefs[0].envKey)
	assert.Equal(t, "Defang_proj_stack_SECRET_ONE", plan.secretRefs[0].identityKey)
	assert.Equal(t, "MIXED", plan.secretRefs[1].envKey)
	assert.Equal(t, "worker-env-MIXED", plan.secretRefs[1].identityKey)
	assert.NotNil(t, plan.secretRefs[1].dependency, "derived secret version must gate instance-template creation")
	assert.Equal(t, "NULLED", plan.secretRefs[2].envKey)
	assert.Equal(t, "Defang_proj_stack_NULLED", plan.secretRefs[2].identityKey)
	// Literals stay inline; the mixed value is held in a derived secret.
	assert.Contains(t, plan.inline, "PLAIN")
	assert.NotContains(t, plan.inline, "MIXED")
	assert.NotContains(t, plan.inline, "BARE")
	assert.NotContains(t, plan.inline, "NULLED")
	require.Len(t, spy.secrets, 1, "mixed interpolation needs one derived secret")
	require.Len(t, spy.versions, 1, "mixed interpolation needs one secret version")
	assert.Len(t, derivedSecretVersionDependencies(plan, nil), 1)

	// with no config provider, everything is inlined
	nilPlan, err := classifyComputeSecretEnv(nil, nil, "worker", env, nil)
	require.NoError(t, err)
	assert.Nil(t, nilPlan.secretRefs)
	assert.Equal(t, env, nilPlan.inline)
}

// TestComputeMixedSecretEnvStaysOutOfInstanceMetadata drives the classification
// and cloud-init construction used by Compute Engine. The fully interpolated
// value must be stored in a Secret Manager version and referenced by the
// boot-time fetch script, never placed in the docker command stored as
// instance metadata.
func TestComputeMixedSecretEnvStaysOutOfInstanceMetadata(t *testing.T) {
	const password = "hunter2"
	spy := &secretManagerSpy{}
	var cloudInit string
	var wg sync.WaitGroup
	wg.Add(1)
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		svc := compose.ServiceConfig{
			Environment: compose.Environment{
				computeDatabaseURLKey: pulumi.String("postgresql://postgres:${PASSWORD}@db:5432/mydb"),
			},
		}
		plan, err := classifyComputeSecretEnv(
			ctx, &mockSecretConfigProvider{prefix: "Defang_proj_stack_"}, "worker", svc.Environment, nil)
		if err != nil {
			return err
		}
		out := getCloudInitConfig(
			"worker", pulumi.String("myapp:latest"), svc, "us-central1", "", "", "", "", "gcp-proj",
			false, nil, plan, nil)
		out.ApplyT(func(value string) string {
			defer wg.Done()
			cloudInit = value
			return value
		})
		return nil
	}, pulumi.WithMocks("proj", "stack", spy))
	require.NoError(t, err)
	wg.Wait()

	assert.NotContains(t, cloudInit, password, "secret value must not appear in instance metadata")
	assert.NotContains(t, cloudInit, computeDatabaseURLKey+"=postgresql",
		"composed value must not be an inline docker env flag")
	assert.Contains(t, cloudInit, "ExecStartPre=/run/defang/worker-secrets.sh")
	assert.Contains(t, cloudInit, "--env-file /run/defang/worker.env")
	assert.Contains(t, cloudInit, fmt.Sprintf(`printf '%%s=%%s\n' '%s' "$v"`, computeDatabaseURLKey))

	require.Len(t, spy.secrets, 1, "expected a derived Secret Manager secret")
	require.Len(t, spy.versions, 1, "expected a version containing the composed value")
	value := spy.versions[0].Inputs["secretData"]
	require.True(t, value.IsSecret(), "derived secret data must remain a Pulumi secret")
	assert.Equal(t, "postgresql://postgres:val-PASSWORD@db:5432/mydb", value.SecretValue().Element.StringValue())
}

func renderSecretFetchScript(t *testing.T, refs []computeSecretEnv) string {
	t.Helper()
	var rendered string
	var wg sync.WaitGroup
	wg.Add(1)
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		wf, _, _ := secretFetchScript("my-proj", "svc", refs)
		wf.ApplyT(func(value string) string {
			defer wg.Done()
			rendered = value
			return value
		})
		return nil
	}, pulumi.WithMocks("proj", "stack", testMocks{}))
	require.NoError(t, err)
	wg.Wait()
	return rendered
}

// secretFetchScript emits a COS-compatible boot fetch (metadata token + Secret
// Manager REST API) writing a tmpfs env-file, plus the ExecStartPre + env-file
// flag that consume it.
func TestSecretFetchScript(t *testing.T) {
	refs := []computeSecretEnv{
		{envKey: "DB", secretID: pulumi.String("Defang_p_s_DBPASS")},
		{envKey: "API", secretID: pulumi.String("Defang_p_s_APIKEY")},
	}
	_, pre, flag := secretFetchScript("my-proj", "svc", refs)
	rendered := renderSecretFetchScript(t, refs)

	assert.Equal(t, "ExecStartPre=/run/defang/svc-secrets.sh", pre)
	assert.Equal(t, "--env-file /run/defang/svc.env", flag)
	assert.Contains(t, rendered, "path: /run/defang/svc-secrets.sh")
	assert.Contains(t, rendered, `permissions: "0700"`)
	assert.Contains(t, rendered, "metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token")
	assert.Contains(t, rendered, "https://secretmanager.googleapis.com/v1/projects/my-proj/secrets/")
	assert.Contains(t, rendered,
		`v=$(sm 'Defang_p_s_DBPASS') || { echo "defang: failed to fetch secret for DB" >&2; exit 1; }`)
	assert.Contains(t, rendered, `printf '%s=%s\n' 'DB' "$v"`)
	assert.Contains(t, rendered,
		`v=$(sm 'Defang_p_s_APIKEY') || { echo "defang: failed to fetch secret for API" >&2; exit 1; }`)
	assert.Contains(t, rendered, `printf '%s=%s\n' 'API' "$v"`)
	assert.Contains(t, rendered, "} > /run/defang/svc.env")
	// fetch failures must fail ExecStartPre, never write an empty value
	assert.Contains(t, rendered, "set -euo pipefail")
	assert.Contains(t, rendered, `[ -n "$tok" ]`)
	assert.Contains(t, rendered, "curl -fsS")
	// Both requests share one curl_retry helper, so the per-attempt bound
	// (ExecStartPre runs in Type=oneshot units, which have no default systemd
	// start timeout -- an unbounded curl would stall the boot indefinitely)
	// appears once, in the helper's definition, not once per call site.
	assert.Equal(t, 1, strings.Count(rendered, "--connect-timeout 5 --max-time 30"))
	assert.Equal(t, 2, strings.Count(rendered, "curl_retry -H"))

	// no refs -> no output
	_, pre2, flag2 := secretFetchScript("p", "svc", nil)
	assert.Empty(t, renderSecretFetchScript(t, nil))
	assert.Empty(t, pre2)
	assert.Empty(t, flag2)
}

// Google's REST APIs pretty-print JSON by default (a space after every colon,
// e.g. `"data": "..."`), unlike the metadata server's own minified responses.
// The generated script's extraction must handle both, or every secret-backed
// Compute Engine boot fails fetching. This runs the actual generated sm()
// function against a realistic pretty-printed response instead of asserting
// on the script text.
func TestSecretFetchScriptExtractsPrettyPrintedJSON(t *testing.T) {
	wf := renderSecretFetchScript(t, []computeSecretEnv{{envKey: "DB", secretID: pulumi.String("my-secret")}})
	raw := rawScriptLines(t, wf)

	// Keep everything through the sm() definition. Drop the /run write (no
	// permission to create it in a test sandbox) and stop before the block
	// that would actually invoke it against the real envFile.
	var setup []string
	for _, line := range raw {
		if strings.Contains(line, "mkdir -p /run/defang") {
			continue
		}
		setup = append(setup, line)
		if strings.HasPrefix(line, "sm() {") {
			break
		}
	}
	require.Contains(t, setup[len(setup)-1], "sm() {")

	const secretValue = "s3cr3t"
	encoded := base64.StdEncoding.EncodeToString([]byte(secretValue))
	script := `
curl() {
  case "$*" in
  *metadata.google.internal*) echo '{
  "access_token": "test-token",
  "expires_in": 3599
}' ;;
  *secretmanager.googleapis.com*) echo '{
  "name": "projects/p/secrets/my-secret/versions/1",
  "payload": {
    "data": "` + encoded + `"
  }
}' ;;
  esac
}
export -f curl
` + strings.Join(setup, "\n") + "\nsm 'my-secret'\n"
	//nolint:gosec // G204: script is test-authored, not external input
	out, err := exec.CommandContext(t.Context(), "bash", "-c", script).Output()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		t.Fatalf("script failed: %v\nstderr: %s", err, ee.Stderr)
	}
	require.NoError(t, err)
	assert.Equal(t, secretValue, strings.TrimSpace(string(out)))
}

// rawScriptLines reconstructs the raw bash script from a write_files
// `content: |` block (each line indented 6 spaces further, see
// secretFetchScript), so tests can run the actual generated script instead of
// asserting on its text.
func rawScriptLines(t *testing.T, wf string) []string {
	t.Helper()
	var raw []string
	inContent := false
	for _, line := range strings.Split(wf, "\n") {
		if strings.Contains(line, "content: |") {
			inContent = true
			continue
		}
		if inContent {
			raw = append(raw, strings.TrimPrefix(line, "      "))
		}
	}
	require.NotEmpty(t, raw)
	return raw
}

// curlRetrySetup returns the script lines through the closing brace of the
// generated curl_retry() definition, dropping the /run write (no permission
// to create it in a test sandbox).
func curlRetrySetup(t *testing.T, wf string) []string {
	t.Helper()
	raw := rawScriptLines(t, wf)
	var setup []string
	inFunc := false
	for _, line := range raw {
		if strings.Contains(line, "mkdir -p /run/defang") {
			continue
		}
		setup = append(setup, line)
		if strings.Contains(line, "curl_retry() {") {
			inFunc = true
			continue
		}
		if inFunc && strings.TrimSpace(line) == "}" {
			return setup
		}
	}
	t.Fatal("curl_retry() definition not found in generated script")
	return nil
}

// curl_retry must retry a failing request with backoff before giving up. This
// runs the actual generated retry loop against a curl stub that fails twice
// then succeeds, and a no-op sleep stub so the test doesn't block on the real
// ~30s of backoff -- it asserts on attempt count and behavior, not timing.
func TestSecretFetchScriptRetriesOnTransientFailure(t *testing.T) {
	wf := renderSecretFetchScript(t, []computeSecretEnv{{envKey: "DB", secretID: pulumi.String("my-secret")}})
	setup := curlRetrySetup(t, wf)

	script := `
n=0
sleeps=0
curl() {
  n=$((n+1))
  [ "$n" -lt 3 ] && return 22
  echo ok
}
export -f curl
sleep() { sleeps=$((sleeps+1)); }
export -f sleep
` + strings.Join(setup, "\n") + `
curl_retry -H test http://example.invalid || rc=$?
echo "rc=${rc:-0} n=$n sleeps=$sleeps"
`
	//nolint:gosec // G204: script is test-authored, not external input
	out, err := exec.CommandContext(t.Context(), "bash", "-c", script).Output()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		t.Fatalf("script failed: %v\nstderr: %s", err, ee.Stderr)
	}
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	// Succeeds on the 3rd attempt, after 2 backoff sleeps. curl_retry's own
	// stdout (the stubbed "ok" from the successful attempt) precedes this.
	assert.Equal(t, "rc=0 n=3 sleeps=2", lines[len(lines)-1])
}

// Once curl_retry's attempts are exhausted it must give up rather than retry
// forever, returning failure so the caller's existing fail-fast handling
// (non-zero exit, stderr message, ExecStartPre failure) still applies.
func TestSecretFetchScriptGivesUpAfterMaxAttempts(t *testing.T) {
	wf := renderSecretFetchScript(t, []computeSecretEnv{{envKey: "DB", secretID: pulumi.String("my-secret")}})
	setup := curlRetrySetup(t, wf)

	script := `
n=0
curl() {
  n=$((n+1))
  return 22
}
export -f curl
sleep() { :; }
export -f sleep
` + strings.Join(setup, "\n") + `
curl_retry -H test http://example.invalid || rc=$?
echo "rc=${rc:-0} n=$n"
`
	//nolint:gosec // G204: script is test-authored, not external input
	out, err := exec.CommandContext(t.Context(), "bash", "-c", script).Output()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		t.Fatalf("script failed: %v\nstderr: %s", err, ee.Stderr)
	}
	require.NoError(t, err)
	// 5 attempts total, then gives up -- never retries forever.
	assert.Equal(t, "rc=1 n=5", strings.TrimSpace(string(out)))
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
			{
				envKey: "SECRET_ENV", secretID: pulumi.String("Defang_proj_stack_SECRET_ENV"),
			},
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

	assert.Contains(t, cloudInit, "ExecStartPre=/run/defang/api-secrets.sh")
	assert.Contains(t, cloudInit, "--env-file /run/defang/api.env")
	assert.Contains(t, cloudInit, "https://secretmanager.googleapis.com/v1/projects/gcp-proj/secrets/")
	assert.Contains(t, cloudInit, `v=$(sm 'Defang_proj_stack_SECRET_ENV')`)
	assert.Contains(t, cloudInit, `printf '%s=%s\n' 'SECRET_ENV' "$v"`)
	// plain value inlined, secret value NOT inlined
	assert.Contains(t, cloudInit, `-e "PLAIN=x"`)
	assert.NotContains(t, cloudInit, `-e "SECRET_ENV`)
	// no leftover format artifacts
	assert.NotContains(t, cloudInit, "%!")
}

// A long compose service name combined with a configured autonaming pattern
// used to push the health check and firewall physical names over GCP's
// 63-char RFC1035 limit (Pulumi's autoname appends <project>-<stack>-<name>
// plus a random suffix). gcpComputeName caps them explicitly instead.
// The MIG health check's firewall shares its logical name rather than
// appending a "-fw" suffix: the GCP console's own type column already
// identifies it as a firewall rule, so a type-naming suffix is redundant.
// Physical naming (length, case, project/stack scoping) is Pulumi
// autonaming's job, configured via the per-resource-type overrides in
// cd/config.go -- not asserted here.
func TestCreateMIGAutoHealingFirewallSharesHealthCheckName(t *testing.T) {
	mocks := &namedResourceMocks{}
	port := 6379

	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := createMIGAutoHealing(
			ctx, "smokeworker", "smokeworker", &port, false, pulumi.String("projects/p/global/networks/vpc"), nil)
		return err
	}, pulumi.WithMocks("pulumi-project", "dev", mocks))
	require.NoError(t, err)

	hcName := mocks.names["gcp:compute/healthCheck:HealthCheck"]
	fwName := mocks.names["gcp:compute/firewall:Firewall"]
	assert.Equal(t, "smokeworker-6379", hcName)
	assert.Equal(t, hcName, fwName, "firewall should share the health check's logical name, not append -fw")
}

// TestCreateMIGAutoHealingHonorsStartPeriod covers the three
// svcHealthCheck.StartPeriodSeconds cases: absent (no grace period — this
// provider declares no VM-boot allowance that isn't in the compose spec,
// matching Azure VM/VMSS, which has never had one), present (initialDelaySec
// is exactly the declared start_period, not a hidden baseline plus it), and
// a value past GCP's documented initialDelaySec ceiling (clamp rather than
// send an invalid request).
func TestCreateMIGAutoHealingHonorsStartPeriod(t *testing.T) {
	port := 6379
	tests := []struct {
		name        string
		healthCheck *compose.HealthCheckConfig
		want        int
	}{
		{
			name: "no healthcheck means no grace period", healthCheck: nil,
			want: 0,
		},
		{
			name:        "start_period sets the grace period directly",
			healthCheck: &compose.HealthCheckConfig{StartPeriodSeconds: 45},
			want:        45,
		},
		{
			name:        "a start_period past GCP's ceiling is clamped",
			healthCheck: &compose.HealthCheckConfig{StartPeriodSeconds: 100000},
			want:        maxMIGAutoHealingInitialDelaySeconds,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var autoHealing compute.RegionInstanceGroupManagerAutoHealingPoliciesPtrInput
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				var err error
				autoHealing, err = createMIGAutoHealing(
					ctx, "smokeworker", "smokeworker", &port, false,
					pulumi.String("projects/p/global/networks/vpc"), tt.healthCheck)
				return err
			}, pulumi.WithMocks("pulumi-project", "dev", &namedResourceMocks{}))
			require.NoError(t, err)

			policy, ok := autoHealing.(compute.RegionInstanceGroupManagerAutoHealingPoliciesArgs)
			require.True(t, ok, "expected RegionInstanceGroupManagerAutoHealingPoliciesArgs, got %T", autoHealing)
			assert.Equal(t, pulumi.Int(tt.want), policy.InitialDelaySec)
		})
	}
}

// A live smoketest (defang-mvp#3181) hit GCP's 63-char compute resource name
// limit: autonaming's pattern for InstanceTemplate is
// "${project}-${stack}-${name}-${hex(7)}" (cd/config.go), and with the
// logical name "smokeworker-instance-template" plus a realistic
// project/stack ("html-css-js"/"newprovidergcp"), the physical name came to
// 64 chars -- one over the limit. Assert the bare service name instead of
// re-deriving the exact character budget here (that belongs to autonaming's
// own pattern, not this package).
func TestCreateInstanceTemplateUsesBareServiceName(t *testing.T) {
	mocks := &namedResourceMocks{}

	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := createInstanceTemplate(
			ctx, "smokeworker", "smokeworker", "e2-small", "debian-cloud/debian-12",
			pulumi.String("#!/bin/sh\n"),
			&ServiceIdentity{Email: pulumi.String("test@example.com")},
			nil,
			testInfra(ctx),
			pulumi.ResourceArrayOutput{},
		)
		return err
	}, pulumi.WithMocks("pulumi-project", "dev", mocks))
	require.NoError(t, err)

	tmplName := mocks.names["gcp:compute/instanceTemplate:InstanceTemplate"]
	assert.Equal(t, "smokeworker", tmplName)
}
