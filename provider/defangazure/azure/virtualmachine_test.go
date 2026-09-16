package azure

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
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
