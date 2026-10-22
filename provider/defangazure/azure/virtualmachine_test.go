package azure

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/compute/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/network/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsVirtualMachineService(t *testing.T) {
	tests := []struct {
		name string
		svc  compose.ServiceConfig
		want bool
	}{
		{name: "no ports", svc: compose.ServiceConfig{}},
		{name: "default tcp", svc: compose.ServiceConfig{Ports: []compose.ServicePortConfig{{Target: 53}}}},
		{
			name: "explicit tcp",
			svc:  compose.ServiceConfig{Ports: []compose.ServicePortConfig{{Target: 53, Protocol: compose.PortProtocolTCP}}},
		},
		{
			name: "udp",
			svc:  compose.ServiceConfig{Ports: []compose.ServicePortConfig{{Target: 53, Protocol: compose.PortProtocolUDP}}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsVirtualMachineService(&tt.svc))
		})
	}
}

func TestVMDockerFlagsPreserveTCPAndUDPOnSamePort(t *testing.T) {
	svc := compose.ServiceConfig{Ports: []compose.ServicePortConfig{
		{Target: 53, Mode: compose.PortModeIngress, Protocol: compose.PortProtocolTCP},
		{Target: 53, Mode: compose.PortModeIngress, Protocol: compose.PortProtocolUDP},
	}}
	flags, _ := vmDockerFlags(svc)
	assert.Contains(t, flags, "53:53/tcp")
	assert.Contains(t, flags, "53:53/udp")
}

func TestSystemdArgEscapesExecExpansions(t *testing.T) {
	assert.Equal(t, `"a'b $$HOME 100%% c\\d\n"`, systemdArg("a'b $HOME 100% c\\d\n"))
}

func TestVMDockerFlagsUseSystemdQuoting(t *testing.T) {
	svc := compose.ServiceConfig{
		Entrypoint: []string{"/bin/sh", "can't $HOME"},
		Command:    []string{"echo", "100% ready"},
	}
	flags, command := vmDockerFlags(svc)
	assert.Contains(t, flags, `--entrypoint "/bin/sh"`)
	assert.Contains(t, command, `"can't $$HOME"`)
	assert.Contains(t, command, `"100%% ready"`)
	assert.NotContains(t, command, `'"'"'`)
}

func TestClassifyVMEnvironment(t *testing.T) {
	env := compose.Environment{
		"BARE":  pulumi.String("${API_KEY}"),
		"MIXED": pulumi.String("prefix-${API_KEY}"),
		"NULL":  nil,
		"PLAIN": pulumi.String("literal"),
	}
	plan, err := classifyVMEnvironment(nil, NewConfigProvider("https://vault.vault.azure.net"), env)
	require.NoError(t, err)

	require.Equal(t, []vmSecretEnv{
		{envKey: "BARE", secretURL: "https://vault.vault.azure.net/secrets/API-KEY"}, //nolint:gosec
		{envKey: "NULL", secretURL: "https://vault.vault.azure.net/secrets/NULL"},    //nolint:gosec
	}, plan.secretRefs)
	assert.Equal(t, pulumi.String("prefix-${API_KEY}"), plan.inline["MIXED"])
	assert.Equal(t, pulumi.String("literal"), plan.inline["PLAIN"])
	assert.NotContains(t, plan.inline, "BARE")
	assert.NotContains(t, plan.inline, "NULL")

	nilPlan, err := classifyVMEnvironment(nil, nil, env)
	require.NoError(t, err)
	assert.Equal(t, env, nilPlan.inline)
	assert.Empty(t, nilPlan.secretRefs)

	_, err = classifyVMEnvironment(nil, &compose.PulumiConfigProvider{}, env)
	assert.Error(t, err)
}

func TestVMSecretFetchFile(t *testing.T) {
	const identityID = "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/kv"
	refs := []vmSecretEnv{
		{envKey: "API_KEY", secretURL: "https://vault.vault.azure.net/secrets/API-KEY"}, //nolint:gosec
	}
	var writeFile string
	var wg sync.WaitGroup
	wg.Add(1)
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		out := vmSecretFetchFile("dns", pulumi.String(identityID).ToStringPtrOutput(), refs)
		out.ApplyT(func(value string) string {
			defer wg.Done()
			writeFile = value
			return value
		})
		return nil
	}, pulumi.WithMocks("project", "stack", &recordVMMocks{resources: make(map[string][]resource.PropertyMap)}))
	require.NoError(t, err)
	wg.Wait()

	assert.Contains(t, writeFile, "path: /usr/local/sbin/defang-dns-secrets")
	assert.Contains(t, writeFile, `permissions: "0700"`)
	assert.Contains(t, writeFile, "umask 077")
	assert.Contains(t, writeFile, "resource=https://vault.azure.net")
	assert.Contains(t, writeFile, identityID)
	assert.Contains(t, writeFile, "https://vault.vault.azure.net/secrets/API-KEY?api-version=7.4")
	assert.Contains(t, writeFile, `printf '%s=%s\n' 'API_KEY' "$value"`)
	assert.Contains(t, writeFile, "} > \"$tmp\"")
	assert.Contains(t, writeFile, "mv \"$tmp\" /run/defang/dns.env")
}

func TestVirtualMachineSize(t *testing.T) {
	tests := []struct {
		name string
		svc  compose.ServiceConfig
		want string
	}{
		{name: "compose defaults", svc: compose.ServiceConfig{}, want: vmSizeB1ms},
		{name: "two CPUs", svc: serviceWithVMResources(2, "4G"), want: vmSizeB2s},
		{name: "memory selects D series", svc: serviceWithVMResources(2, "8G"), want: "Standard_D2s_v5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, virtualMachineSize(tt.svc))
		})
	}
}

func serviceWithVMResources(cpus float64, memory string) compose.ServiceConfig {
	return compose.ServiceConfig{Deploy: &compose.DeployConfig{Resources: &compose.Resources{
		Reservations: &compose.ResourceConfig{CPUs: &cpus, Memory: &memory},
	}}}
}

type recordVMMocks struct {
	mu        sync.Mutex
	resources map[string][]resource.PropertyMap
}

func (m *recordVMMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	m.mu.Lock()
	m.resources[args.TypeToken] = append(m.resources[args.TypeToken], args.Inputs)
	m.mu.Unlock()
	if strings.Contains(args.TypeToken, "randomPassword") {
		args.Inputs[resource.PropertyKey("result")] = resource.NewStringProperty("Test-password-123!")
	}
	if strings.HasSuffix(args.TypeToken, ":PublicIPAddress") {
		args.Inputs[resource.PropertyKey("ipAddress")] = resource.NewStringProperty("203.0.113.53")
	}
	return args.Name + "_id", args.Inputs, nil
}

func (*recordVMMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func (m *recordVMMocks) byTypeSuffix(suffix string) []resource.PropertyMap {
	m.mu.Lock()
	defer m.mu.Unlock()
	for token, resources := range m.resources {
		if strings.HasSuffix(token, suffix) {
			return resources
		}
	}
	return nil
}

func TestCreateVirtualMachineServiceRegistersDualProtocolLoadBalancer(t *testing.T) {
	t.Setenv("PULUMI_CONFIG", `{"defang-azure:min-healthy-percent": "50"}`)
	mocks := &recordVMMocks{resources: make(map[string][]resource.PropertyMap)}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		rg, err := resources.NewResourceGroup(ctx, "rg", nil)
		if err != nil {
			return err
		}
		vnet, err := network.NewVirtualNetwork(ctx, "network", &network.VirtualNetworkArgs{
			ResourceGroupName: rg.Name,
		})
		if err != nil {
			return err
		}
		subnet, err := network.NewSubnet(ctx, "compute", &network.SubnetArgs{
			ResourceGroupName:  rg.Name,
			VirtualNetworkName: vnet.Name,
			AddressPrefix:      pulumi.String("10.0.4.0/24"),
		})
		if err != nil {
			return err
		}
		svc := compose.ServiceConfig{Ports: []compose.ServicePortConfig{
			{Target: 53, Mode: compose.PortModeIngress, Protocol: compose.PortProtocolTCP},
			{Target: 53, Mode: compose.PortModeIngress, Protocol: compose.PortProtocolUDP},
		}, Environment: compose.Environment{
			"PLAIN":      pulumi.String("visible"),
			"SECRET_ENV": pulumi.String("${API_KEY}"),
		}}
		kvIdentityID := pulumi.String(
			"/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/kv",
		)
		_, err = CreateVirtualMachineService(
			ctx,
			"dns",
			pulumi.String("cunnie/sslip.io-dns-server:latest"),
			svc,
			&SharedInfra{
				ResourceGroup:      rg,
				Networking:         &NetworkingResult{VNet: vnet, ComputeSubnet: subnet},
				ConfigProvider:     NewConfigProvider("https://vault.vault.azure.net"),
				KeyVaultIdentityID: kvIdentityID.ToStringPtrOutput(),
			},
			nil,
		)
		return err
	}, pulumi.WithMocks("project", "stack", mocks))
	require.NoError(t, err)

	lbs := mocks.byTypeSuffix(":LoadBalancer")
	require.Len(t, lbs, 1)
	rules := lbs[0][resource.PropertyKey("loadBalancingRules")].ArrayValue()
	require.Len(t, rules, 2)
	protocols := map[string]int{}
	for _, rule := range rules {
		props := rule.ObjectValue()
		assert.InDelta(t, 53, props[resource.PropertyKey("frontendPort")].NumberValue(), 0)
		assert.InDelta(t, 53, props[resource.PropertyKey("backendPort")].NumberValue(), 0)
		protocols[props[resource.PropertyKey("protocol")].StringValue()]++
	}
	assert.Equal(t, map[string]int{azureProtocolTCP: 1, azureProtocolUDP: 1}, protocols)

	vmScaleSets := mocks.byTypeSuffix(":VirtualMachineScaleSet")
	require.Len(t, vmScaleSets, 1)
	upgradePolicy := vmScaleSets[0][resource.PropertyKey("upgradePolicy")].ObjectValue()
	assert.Equal(t, "Rolling", upgradePolicy[resource.PropertyKey("mode")].StringValue())
	rollingPolicy := upgradePolicy[resource.PropertyKey("rollingUpgradePolicy")].ObjectValue()
	assert.InDelta(t, 50, rollingPolicy[resource.PropertyKey("maxBatchInstancePercent")].NumberValue(), 0)
	assert.True(t, rollingPolicy[resource.PropertyKey("maxSurge")].BoolValue())
	assert.InDelta(t, 50, rollingPolicy[resource.PropertyKey("maxUnhealthyInstancePercent")].NumberValue(), 0)
	assert.InDelta(t, 0, rollingPolicy[resource.PropertyKey("maxUnhealthyUpgradedInstancePercent")].NumberValue(), 0)
	assert.False(t, rollingPolicy[resource.PropertyKey("rollbackFailedInstancesOnPolicyBreach")].BoolValue(),
		"Azure rejects rollbackFailedInstancesOnPolicyBreach when maxSurge is enabled")
	vmProfile := vmScaleSets[0][resource.PropertyKey("virtualMachineProfile")].ObjectValue()
	osProfile := vmProfile[resource.PropertyKey("osProfile")].ObjectValue()
	customData := osProfile[resource.PropertyKey("customData")].StringValue()
	cloudConfigBytes, err := base64.StdEncoding.DecodeString(customData)
	require.NoError(t, err)
	cloudConfig := string(cloudConfigBytes)
	assert.Contains(t, cloudConfig, "path: /etc/systemd/resolved.conf.d/defang.conf")
	assert.Contains(t, cloudConfig, "DNSStubListener=no")
	assert.Contains(t, cloudConfig, "ln -sf /run/systemd/resolve/resolv.conf /etc/resolv.conf")
	assert.Less(t,
		strings.Index(cloudConfig, "systemctl restart systemd-resolved"),
		strings.Index(cloudConfig, "systemctl restart docker"),
		"the DNS stub must release port 53 before Docker starts the service",
	)
	assert.Contains(t, cloudConfig, "ExecStartPre=/usr/local/sbin/defang-dns-secrets")
	assert.Contains(t, cloudConfig, "--env-file /run/defang/dns.env")
	assert.Contains(t, cloudConfig, "https://vault.vault.azure.net/secrets/API-KEY?api-version=7.4")
	assert.Contains(t, cloudConfig, `--env "PLAIN=visible"`)
	assert.NotContains(t, cloudConfig, "${API_KEY}")

	identity := vmScaleSets[0][resource.PropertyKey("identity")].ObjectValue()
	assigned := identity[resource.PropertyKey("userAssignedIdentities")].ArrayValue()
	require.Len(t, assigned, 1)
	assert.Equal(t, "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/kv",
		assigned[0].StringValue())
}

func TestVirtualMachineUpgradePolicyFollowsMinHealthyPercent(t *testing.T) {
	tests := []struct {
		name                string
		minHealthyPercent   int
		mode                compute.UpgradeMode
		maxUnhealthyPercent int
	}{
		{name: "affordable permits teardown", minHealthyPercent: 0, mode: compute.UpgradeModeAutomatic},
		{
			name: "balanced rolls at fifty percent", minHealthyPercent: 50,
			mode: compute.UpgradeModeRolling, maxUnhealthyPercent: 50,
		},
		{
			name: "high availability uses Azure floor", minHealthyPercent: 100,
			mode: compute.UpgradeModeRolling, maxUnhealthyPercent: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := virtualMachineUpgradePolicy(tt.minHealthyPercent)
			assert.Equal(t, tt.mode, policy.Mode)
			if tt.mode == compute.UpgradeModeAutomatic {
				assert.Nil(t, policy.RollingUpgradePolicy)
				return
			}
			require.NotNil(t, policy.RollingUpgradePolicy)
			rollingPolicy, ok := policy.RollingUpgradePolicy.(*compute.RollingUpgradePolicyArgs)
			require.True(t, ok, "expected RollingUpgradePolicyArgs, got %T", policy.RollingUpgradePolicy)
			assert.Equal(t, tt.maxUnhealthyPercent,
				intPtrInputValue(t, rollingPolicy.MaxUnhealthyInstancePercent))
		})
	}
}

// TestVMHealthStartPeriodSecondsClamps covers vmHealthStartPeriodSeconds in
// isolation: no healthcheck, a declared start_period, and a pathological
// value past the provider's ceiling.
func TestVMHealthStartPeriodSecondsClamps(t *testing.T) {
	tests := []struct {
		name        string
		healthCheck *compose.HealthCheckConfig
		want        int
	}{
		{name: "no healthcheck means no grace", healthCheck: nil, want: 0},
		{name: "start_period is used directly", healthCheck: &compose.HealthCheckConfig{StartPeriodSeconds: 60}, want: 60},
		{
			name:        "a pathological start_period is clamped, not left unbounded",
			healthCheck: &compose.HealthCheckConfig{StartPeriodSeconds: 100000},
			want:        maxVMHealthStartPeriodSeconds,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				svc := compose.ServiceConfig{HealthCheck: tt.healthCheck}
				assert.Equal(t, tt.want, vmHealthStartPeriodSeconds(ctx, "worker", svc))
				return nil
			}, pulumi.WithMocks("project", "stack", azureNoopMocks{}))
			require.NoError(t, err)
		})
	}
}

// pollHealthStatus polls defang-health at 127.0.0.1:port until it answers
// (it needs a moment to start listening), returning the HTTP status code of
// the first response it gets.
func pollHealthStatus(t *testing.T, port int) int {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	for range 50 {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			defer func() { _ = resp.Body.Close() }()
			return resp.StatusCode
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("defang-health never started listening")
	return 0
}

// TestDefangHealthScriptHonorsStartPeriod runs the actual generated
// defang-health script (the VM's HTTP health endpoint) against a real
// python3, with "docker" stubbed to report the container as not running.
//
// Reproduces a DefangLabs/station production incident (2026-10-22): Azure's
// load balancer HTTP probe marks an instance up or down on the very first
// explicit response it gets, regardless of the probe's configured threshold
// (confirmed against Microsoft's own docs during PR review) — so a
// start_period a compose author declares has to change what this script
// reports, not the probe's settings, or a new VMSS instance that's merely
// slow, not broken, can still abort a rolling upgrade.
func TestDefangHealthScriptHonorsStartPeriod(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}

	const port = 18123
	script := renderDefangHealthScript(5, "worker", port) // 5s start_period
	dir := t.TempDir()
	scriptPath := dir + "/defang-health"
	// 0600: python3 only needs to read this file, not execute it.
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0o600))

	// Fake "docker" always reports the container as NOT running, so every
	// response for the life of this test reflects the start_period logic,
	// not real container state.
	fakeDocker := dir + "/docker"
	//nolint:gosec // G306: must be executable to work as a PATH-resolved "docker"
	require.NoError(t, os.WriteFile(fakeDocker, []byte("#!/bin/sh\necho false\n"), 0o700))

	//nolint:gosec // G204: scriptPath is t.TempDir()-derived, not external input
	cmd := exec.CommandContext(t.Context(), "python3", scriptPath)
	cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"))
	require.NoError(t, cmd.Start())
	defer func() { _ = cmd.Process.Kill() }()

	get := func() int { return pollHealthStatus(t, port) }

	assert.Equal(t, 200, get(), "unhealthy container during start_period must not fail the probe")
	time.Sleep(6 * time.Second) // past the 5s start_period
	assert.Equal(t, 503, get(), "unhealthy container past start_period must fail the probe")
}

// TestDefangHealthScriptResetsGraceOnContainerRestart covers the gap
// CodeRabbit flagged in PR review on the first version of this fix: the
// grace window has to be measured from the CONTAINER's own start time
// (docker inspect's State.StartedAt), not from when the health endpoint
// itself started -- Restart=always restarts the container independently of
// this script, and a restart after a later crash deserves its own
// start_period, not zero grace just because the health endpoint has been
// up for a while.
func TestDefangHealthScriptResetsGraceOnContainerRestart(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}

	const port = 18124
	const startPeriod = 3
	script := renderDefangHealthScript(startPeriod, "worker", port)
	dir := t.TempDir()
	scriptPath := dir + "/defang-health"
	// 0600: python3 only needs to read this file, not execute it.
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0o600))

	// statePath holds "<running>\n<startedAt>\n", mirroring the two fields
	// defang-health actually reads off `docker inspect`. Its absence
	// mirrors `docker inspect` failing outright: the container doesn't
	// exist yet (still pulling the image, before the first `docker run`).
	statePath := dir + "/state"
	fakeDocker := dir + "/docker"
	//nolint:gosec // G306: must be executable to work as a PATH-resolved "docker"
	require.NoError(t, os.WriteFile(fakeDocker, []byte(`#!/bin/sh
state="`+statePath+`"
[ -f "$state" ] || exit 1
running=$(sed -n 1p "$state")
started=$(sed -n 2p "$state")
printf '%s|%s\n' "$running" "$started"
`), 0o700))

	//nolint:gosec // G204: scriptPath is t.TempDir()-derived, not external input
	cmd := exec.CommandContext(t.Context(), "python3", scriptPath)
	cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"))
	require.NoError(t, cmd.Start())
	defer func() { _ = cmd.Process.Kill() }()

	get := func() int { return pollHealthStatus(t, port) }
	setState := func(running bool, startedAt time.Time) {
		runningStr := "false"
		if running {
			runningStr = "true"
		}
		content := runningStr + "\n" + startedAt.UTC().Format("2006-01-02T15:04:05Z") + "\n"
		require.NoError(t, os.WriteFile(statePath, []byte(content), 0o600))
	}

	// Phase 1: container doesn't exist yet at all -- falls back to this
	// process's own (just-started) uptime, within start_period.
	assert.Equal(t, 200, get(), "no container yet, still within this endpoint's own startup grace")

	// Phase 2: container is up and actually running -- healthy regardless
	// of age.
	setState(true, time.Now().Add(-1*time.Hour))
	assert.Equal(t, 200, get(), "a genuinely running container is healthy no matter its age")

	// Phase 3: container crashed, but its own StartedAt is recent -- still
	// within ITS start_period, not the health endpoint's uptime.
	setState(false, time.Now())
	assert.Equal(t, 200, get(), "a crashed container within its own start_period must not fail the probe")

	// Phase 4: wait past that container's start_period -- now unhealthy.
	time.Sleep(time.Duration(startPeriod+1) * time.Second)
	assert.Equal(t, 503, get(), "a crashed container past its own start_period must fail the probe")

	// Phase 5: the container restarts (a newer StartedAt), still not yet
	// reporting Running -- grace must reset for the new start, even though
	// the health endpoint itself has been up this whole time.
	setState(false, time.Now())
	assert.Equal(t, 200, get(),
		"a container restart must reset the grace window, not reuse the health endpoint's own uptime")
}

// TestAcrLoginScriptSurvivesUnpaddedJWTBase64 runs the actual generated ACR
// login script (the VM's dns.service ExecStartPre) against a real bash, with
// curl/docker stubbed out. It reproduces the production incident this fixed:
// a JWT payload's base64 segment is essentially never a multiple of 4 chars
// (valid padding is 0-2 '=', never 3), so blindly appending '===' left
// trailing bytes that made GNU base64 exit 1 -- despite decoding the tenant
// JSON to stdout correctly first. Under 'set -e -o pipefail' that killed the
// script before the ACR token exchange ran, so the DNS container never
// started, on every single VM boot. This never surfaced as a Pulumi error:
// the script only runs later, as a systemd ExecStartPre inside the VM.
func TestAcrLoginScriptSurvivesUnpaddedJWTBase64(t *testing.T) {
	tenantID := "11111111-2222-3333-4444-555555555555"
	claims, err := json.Marshal(map[string]string{"tid": tenantID})
	require.NoError(t, err)
	// A real AAD access token has three dot-separated parts; only the
	// payload (claims) segment matters here. RawURLEncoding matches AAD's
	// own encoding: no padding, '-'/'_' instead of '+'/'/'.
	fakeJWT := "header." + base64.RawURLEncoding.EncodeToString(claims) + ".sig"

	captureFile := t.TempDir() + "/exchange-args"
	script := renderAcrLoginScript("myregistry.azurecr.io", "/subscriptions/sub/.../identity")

	harness := `
curl() {
  case "$*" in
  *169.254.169.254*) echo '{"access_token":"` + fakeJWT + `"}' ;;
  *oauth2/exchange*) printf '%s' "$*" > '` + captureFile + `'; echo '{"refresh_token":"fake-refresh-token"}' ;;
  *) echo "unexpected curl invocation: $*" >&2; return 1 ;;
  esac
}
export -f curl
docker() {
  case "$1 $2" in
  "login myregistry.azurecr.io") cat >/dev/null; echo "Login Succeeded" ;;
  *) echo "unexpected docker invocation: $*" >&2; return 1 ;;
  esac
}
export -f docker
` + script

	//nolint:gosec // G204: script is test-authored, not external input
	out, err := exec.CommandContext(t.Context(), "bash", "-c", harness).CombinedOutput()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		t.Fatalf("acr-login script failed: %v\noutput: %s", err, out)
	}
	require.NoError(t, err)
	assert.Contains(t, string(out), "Login Succeeded")

	//nolint:gosec // G304: captureFile is t.TempDir()-derived, not external input
	exchangeArgsBytes, err := os.ReadFile(captureFile)
	exchangeArgs := string(exchangeArgsBytes)
	require.NoError(t, err, "the ACR token exchange must run -- it never did before this fix")
	assert.Contains(t, exchangeArgs, "tenant="+tenantID,
		"the tenant extracted from the JWT payload must reach the exchange request")
}

func TestVMComputerNamePrefix(t *testing.T) {
	tests := map[string]string{
		"dns":                      "dns",
		"DNS_Server":               "dns-server",
		"a-very-long-service-name": "a-very-long-ser",
		"exactly-fifteen":          "exactly-fifteen",
	}
	for input, want := range tests {
		assert.Equal(t, want, vmComputerNamePrefix(input))
	}
}
