package azure

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/compute/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/network/v3"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	vmHealthPort     = 65535
	vmSizeB1ms       = "Standard_B1ms"
	vmSizeB2s        = "Standard_B2s"
	azureProtocolTCP = "Tcp"
	azureProtocolUDP = "Udp"
)

var (
	errVirtualMachineNetworkingRequired = errors.New("azure VM services require project networking")
	errPrivateVMServiceUnsupported      = errors.New("private Azure VM services are not supported yet")
	errVMIngressRequired                = errors.New("azure VM load-balanced ports require mode: ingress")
	errArmVMServiceUnsupported          = errors.New("azure VM services do not support arm64 yet")
	errVMSecretsRequireKeyVault         = errors.New("azure VM secret references require a Key Vault identity")
	errEmptyVMSecretURL                 = errors.New("empty Key Vault secret URL")
)

// VirtualMachineServiceResult contains the stable endpoint and the VM scale set
// that runs a service which Azure Container Apps cannot host.
type VirtualMachineServiceResult struct {
	ScaleSet        *compute.VirtualMachineScaleSet
	PublicIPAddress *network.PublicIPAddress
}

// IsVirtualMachineService reports whether a service needs the Azure VM runtime.
// Container Apps supports HTTP and TCP ingress, but not UDP.
func IsVirtualMachineService(svc *compose.ServiceConfig) bool {
	for _, port := range svc.Ports {
		if port.GetProtocol() == compose.PortProtocolUDP {
			return true
		}
	}
	return false
}

type azureVMSize struct {
	name string
	cpu  float64
	mem  int
}

var azureVMSizes = []azureVMSize{
	{name: vmSizeB1ms, cpu: 1, mem: 2 * 1024},
	{name: vmSizeB2s, cpu: 2, mem: 4 * 1024},
	{name: "Standard_D2s_v5", cpu: 2, mem: 8 * 1024},
	{name: "Standard_D4s_v5", cpu: 4, mem: 16 * 1024},
	{name: "Standard_D8s_v5", cpu: 8, mem: 32 * 1024},
	{name: "Standard_D16s_v5", cpu: 16, mem: 64 * 1024},
}

func virtualMachineSize(svc compose.ServiceConfig) string {
	cpu := svc.GetCPUs()
	mem := svc.GetMemoryMiB()
	for _, size := range azureVMSizes {
		if size.cpu >= cpu && size.mem >= mem {
			return size.name
		}
	}
	return azureVMSizes[len(azureVMSizes)-1].name
}

func vmComputerNamePrefix(serviceName string) string {
	prefix := common.ServiceLabel(serviceName)
	if len(prefix) > 15 {
		prefix = prefix[:15]
	}
	return strings.TrimRight(prefix, "-")
}

func azureProtocol(port compose.ServicePortConfig) string {
	if port.GetProtocol() == compose.PortProtocolUDP {
		return azureProtocolUDP
	}
	return azureProtocolTCP
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// systemdArg quotes one argument in an Exec line. systemd applies $ variable
// and % specifier expansion after parsing quotes, so both must be doubled even
// inside a quoted argument. strconv.Quote emits the C-style escapes accepted
// by systemd.syntax(7).
func systemdArg(s string) string {
	s = strings.ReplaceAll(s, "$", "$$")
	s = strings.ReplaceAll(s, "%", "%%")
	return strconv.Quote(s)
}

func vmDockerFlags(svc compose.ServiceConfig) (string, string) {
	flags := make([]string, 0, 2*len(svc.Ports)+4)
	command := make([]string, 0, len(svc.Command)+len(svc.Entrypoint))
	if len(svc.Entrypoint) > 0 {
		flags = append(flags, "--entrypoint", systemdArg(svc.Entrypoint[0]))
		for _, arg := range svc.Entrypoint[1:] {
			command = append(command, systemdArg(arg))
		}
	}
	for _, arg := range svc.Command {
		command = append(command, systemdArg(arg))
	}
	for _, port := range svc.Ports {
		flags = append(flags, "--publish", fmt.Sprintf("%d:%d/%s", port.Target, port.Target, port.GetProtocol()))
	}
	return strings.Join(flags, " "), strings.Join(command, " ")
}

func vmEnvironment(
	serviceName string,
	svc compose.ServiceConfig,
	etag string,
	policyIdentity *PolicyIdentity,
) pulumi.StringOutput {
	parts := []pulumi.StringInput{pulumi.String("--env " + systemdArg("DEFANG_SERVICE="+serviceName))}
	if etag != "" {
		parts = append(parts, pulumi.String("--env "+systemdArg("DEFANG_ETAG="+etag)))
	}
	if policyIdentity != nil {
		parts = append(parts, policyIdentity.ClientID.ApplyT(func(clientID string) string {
			return "--env " + systemdArg("AZURE_CLIENT_ID="+clientID)
		}).(pulumi.StringOutput))
	}
	for key, value := range common.Sorted(svc.Environment) {
		if static, ok := compose.StaticEnvValue(value); ok {
			resolved := ""
			if static != nil {
				resolved = *static
			}
			parts = append(parts, pulumi.String("--env "+systemdArg(key+"="+resolved)))
			continue
		}
		parts = append(parts, value.ToStringPtrOutput().ApplyT(func(resolved *string) string {
			if resolved == nil {
				return "--env " + systemdArg(key+"=")
			}
			return "--env " + systemdArg(key+"="+*resolved)
		}).(pulumi.StringOutput))
	}
	return pulumi.StringArray(parts).ToStringArrayOutput().ApplyT(func(values []string) string {
		return strings.Join(values, " ")
	}).(pulumi.StringOutput)
}

type vmSecretEnv struct {
	envKey    string
	secretURL string
}

type vmEnvironmentPlan struct {
	inline     compose.Environment
	secretRefs []vmSecretEnv
}

// classifyVMEnvironment keeps ordinary values inline but turns bare config
// references (FOO=${SECRET}, or a null FOO:) into Key Vault references that
// the VM resolves at container start. Secret values therefore never enter
// VMSS customData or Docker's command line.
func classifyVMEnvironment(
	ctx *pulumi.Context, cp compose.ConfigProvider, env compose.Environment,
) (vmEnvironmentPlan, error) {
	if cp == nil {
		return vmEnvironmentPlan{inline: env}, nil
	}
	plan := vmEnvironmentPlan{inline: make(compose.Environment, len(env))}
	for key, value := range common.Sorted(env) {
		if configKey := compose.GetConfigName2(key, value); configKey != "" {
			secretURL, err := cp.GetSecretRef(ctx, configKey)
			if err != nil {
				return vmEnvironmentPlan{}, fmt.Errorf("resolving Key Vault reference for %s: %w", key, err)
			}
			if secretURL == "" {
				return vmEnvironmentPlan{}, fmt.Errorf("resolving Key Vault reference for %s: %w", key, errEmptyVMSecretURL)
			}
			plan.secretRefs = append(plan.secretRefs, vmSecretEnv{envKey: key, secretURL: secretURL})
			continue
		}
		plan.inline[key] = value
	}
	return plan, nil
}

// vmSecretFetchFile emits a root-only boot script that authenticates with the
// VM's user-assigned identity, fetches Key Vault values, and writes Docker's
// env file under /run (tmpfs). The secret values never appear in customData.
// Docker env files are line-oriented, so multiline secrets need file mounts
// rather than environment variables.
func vmSecretFetchFile(serviceName string, identityID pulumi.StringPtrInput, refs []vmSecretEnv) pulumi.StringOutput {
	if len(refs) == 0 {
		return pulumi.String("").ToStringOutput()
	}
	unit := common.ServiceLabel(serviceName)
	scriptPath := "/usr/local/sbin/defang-" + unit + "-secrets"
	envFile := "/run/defang/" + unit + ".env"
	return identityID.ToStringPtrOutput().ApplyT(func(identityID *string) string {
		var script strings.Builder
		script.WriteString("#!/bin/bash\nset -euo pipefail\numask 077\nmkdir -p -m 0700 /run/defang\n")
		script.WriteString("curl_retry() {\n" +
			"  local attempt=1 max=5 delay=2\n" +
			"  while ! curl -fsS --connect-timeout 5 --max-time 30 \"$@\"; do\n" +
			"    [ \"$attempt\" -ge \"$max\" ] && return 1\n" +
			"    sleep \"$delay\"\n" +
			"    delay=$((delay * 2))\n" +
			"    attempt=$((attempt + 1))\n" +
			"  done\n" +
			"}\n")
		id := ""
		if identityID != nil {
			id = *identityID
		}
		fmt.Fprintf(&script, "identity_id=%s\n", shellQuote(id))
		script.WriteString(`token=$(curl_retry --get -H Metadata:true \
  --data-urlencode api-version=2018-02-01 \
  --data-urlencode resource=https://vault.azure.net \
  --data-urlencode mi_res_id="$identity_id" \
  http://169.254.169.254/metadata/identity/oauth2/token | jq -er .access_token)
`)
		script.WriteString(`[ -n "$token" ] || { echo "defang: empty metadata token" >&2; exit 1; }
`)
		fmt.Fprintf(&script, "tmp=$(mktemp %s.XXXXXX)\n", envFile)
		script.WriteString(`trap 'rm -f "$tmp"' EXIT
{
`)
		for _, ref := range refs {
			fmt.Fprintf(&script,
				"  value=$(curl_retry -H \"Authorization: Bearer $token\" %s | jq -er .value)\n",
				shellQuote(ref.secretURL+"?api-version=7.4"))
			fmt.Fprintf(&script, "  printf '%%s=%%s\\n' %s \"$value\"\n", shellQuote(ref.envKey))
		}
		fmt.Fprintf(&script, "} > \"$tmp\"\nmv \"$tmp\" %s\ntrap - EXIT\n", envFile)

		var writeFile strings.Builder
		fmt.Fprintf(&writeFile,
			"  - path: %s\n    permissions: \"0700\"\n    owner: root\n    content: |\n", scriptPath)
		for _, line := range strings.Split(strings.TrimRight(script.String(), "\n"), "\n") {
			writeFile.WriteString("      ")
			writeFile.WriteString(line)
			writeFile.WriteByte('\n')
		}
		return writeFile.String()
	}).(pulumi.StringOutput)
}

func acrLoginScript(infra *SharedInfra, svc compose.ServiceConfig) pulumi.StringOutput {
	if svc.Build == nil || infra.BuildInfra == nil {
		return pulumi.String("#!/bin/bash\nexit 0\n").ToStringOutput()
	}
	return pulumi.All(infra.BuildInfra.LoginServer(), infra.BuildInfra.ManagedIdentityID()).ApplyT(
		func(values []any) string {
			registry := values[0].(string)
			identityID := values[1].(string)
			return renderAcrLoginScript(registry, identityID)
		},
	).(pulumi.StringOutput)
}

// renderAcrLoginScript is a plain function (no Pulumi Outputs) so it's unit-testable:
// it's the ExecStartPre of the VM's systemd unit, so a broken script never surfaces as
// a Pulumi error, only as a container that silently never starts.
func renderAcrLoginScript(registry, identityID string) string {
	return fmt.Sprintf(`#!/bin/bash
set -euo pipefail
registry=%s
identity_id=%s
aad_json=$(curl -fsS --get -H Metadata:true \
  --data-urlencode api-version=2018-02-01 \
  --data-urlencode resource=https://management.azure.com/ \
  --data-urlencode mi_res_id="$identity_id" \
  http://169.254.169.254/metadata/identity/oauth2/token)
aad=$(printf '%%s' "$aad_json" | jq -er .access_token)
payload=$(printf '%%s' "$aad" | cut -d. -f2 | tr '_-' '/+')
# A JWT payload segment's base64 length is essentially never a multiple of 4
# (valid padding is 0-2 '=' chars, never 3), so blindly appending '===' always
# leaves trailing bytes GNU base64 rejects with "invalid input" (exit 1) even
# though it decodes the real payload to stdout correctly beforehand. Under
# 'set -e -o pipefail' that exit code, not any actual decoding failure, was
# killing this script on every single run before it ever reached the ACR
# token exchange below.
tenant=$(printf '%%s===' "$payload" | { base64 -d 2>/dev/null || true; } | jq -er .tid)
refresh=$(curl -fsS -X POST -H 'Content-Type: application/x-www-form-urlencoded' \
  --data-urlencode grant_type=access_token \
  --data-urlencode "service=$registry" \
  --data-urlencode "tenant=$tenant" \
  --data-urlencode "access_token=$aad" \
  "https://$registry/oauth2/exchange" | jq -er .refresh_token)
printf '%%s' "$refresh" | docker login "$registry" --username 00000000-0000-0000-0000-000000000000 --password-stdin
`, shellQuote(registry), shellQuote(identityID))
}

//nolint:funlen // the cloud-init document is clearer when kept as one template
func virtualMachineCloudInit(
	serviceName string,
	image pulumi.StringInput,
	svc compose.ServiceConfig,
	infra *SharedInfra,
	policyIdentity *PolicyIdentity,
	envPlan vmEnvironmentPlan,
) pulumi.StringOutput {
	containerName := svc.GetContainerName(serviceName)
	dockerFlags, command := vmDockerFlags(svc)
	inlineSvc := svc
	inlineSvc.Environment = envPlan.inline
	env := vmEnvironment(serviceName, inlineSvc, infra.Etag, policyIdentity)
	login := acrLoginScript(infra, svc)
	quotedImage := image.ToStringOutput().ApplyT(systemdArg).(pulumi.StringOutput)
	secretWriteFile := vmSecretFetchFile(serviceName, infra.KeyVaultIdentityID, envPlan.secretRefs)
	secretExecPre := ""
	secretEnvFlag := ""
	if len(envPlan.secretRefs) > 0 {
		unit := common.ServiceLabel(serviceName)
		secretExecPre = "      ExecStartPre=/usr/local/sbin/defang-" + unit + "-secrets\n"
		secretEnvFlag = "--env-file /run/defang/" + unit + ".env"
	}

	template := `#cloud-config
package_update: true
packages:
  - docker.io
  - curl
  - jq
write_files:
%s
  - path: /etc/docker/daemon.json
    permissions: "0644"
    content: |
      {"userland-proxy": false}
  - path: /etc/systemd/resolved.conf.d/defang.conf
    permissions: "0644"
    content: |
      [Resolve]
      DNSStubListener=no
  - path: /usr/local/sbin/defang-acr-login
    permissions: "0700"
    content: |
%s
  - path: /usr/local/sbin/defang-health
    permissions: "0755"
    content: |
      #!/usr/bin/python3
      import http.server
      import subprocess

      class Handler(http.server.BaseHTTPRequestHandler):
          def do_GET(self):
              result = subprocess.run(
                  ["docker", "inspect", "--format={{.State.Running}}", %s],
                  capture_output=True,
                  check=False,
                  text=True,
              )
              healthy = result.returncode == 0 and result.stdout.strip() == "true"
              body = b"OK" if healthy else b"FAIL"
              self.send_response(200 if healthy else 503)
              self.send_header("Content-Length", str(len(body)))
              self.end_headers()
              self.wfile.write(body)

          def log_message(self, format, *args):
              return

      http.server.ThreadingHTTPServer(("0.0.0.0", %d), Handler).serve_forever()
  - path: /etc/systemd/system/defang-health.service
    permissions: "0644"
    content: |
      [Unit]
      Description=Defang container health endpoint
      After=docker.service
      Requires=docker.service

      [Service]
      ExecStart=/usr/local/sbin/defang-health
      Restart=always
      RestartSec=5

      [Install]
      WantedBy=multi-user.target
  - path: /etc/systemd/system/%s.service
    permissions: "0644"
    content: |
      [Unit]
      Description=Defang service %s
      Wants=network-online.target
      After=network-online.target docker.service
      Requires=docker.service

      [Service]
      Restart=always
      RestartSec=10
      ExecStartPre=/usr/local/sbin/defang-acr-login
%s
      ExecStartPre=-/usr/bin/docker rm -f %s
      ExecStart=/usr/bin/docker run --pull=always --rm --name=%s %s %s %s %s %s
      ExecStop=/usr/bin/docker stop -t 30 %s
      StandardOutput=journal+console
      StandardError=journal+console

      [Install]
      WantedBy=multi-user.target
runcmd:
  - ln -sf /run/systemd/resolve/resolv.conf /etc/resolv.conf
  - systemctl restart systemd-resolved
  - systemctl restart docker
  - systemctl daemon-reload
  - systemctl enable --now defang-health.service
  - systemctl enable --now %s.service
`
	indent := func(script string) string {
		lines := strings.Split(strings.TrimRight(script, "\n"), "\n")
		for i := range lines {
			lines[i] = "      " + lines[i]
		}
		return strings.Join(lines, "\n")
	}
	return pulumi.All(secretWriteFile, login, env, quotedImage).ApplyT(func(values []any) string {
		return fmt.Sprintf(
			template,
			values[0].(string),
			indent(values[1].(string)),
			strconv.Quote(containerName),
			vmHealthPort,
			serviceName,
			serviceName,
			secretExecPre,
			containerName,
			containerName,
			dockerFlags,
			values[2].(string),
			secretEnvFlag,
			values[3].(string),
			command,
			containerName,
			serviceName,
		)
	}).(pulumi.StringOutput)
}

// CreateVirtualMachineService runs a UDP service on an Azure VM scale set and
// fronts it with a Standard Load Balancer whose public IP is a separate,
// statically allocated resource. TCP and UDP rules can share the same port.
//
//nolint:funlen,maintidx // sequential load balancer and VMSS setup is clearer as one function
func CreateVirtualMachineService(
	ctx *pulumi.Context,
	serviceName string,
	image pulumi.StringInput,
	svc compose.ServiceConfig,
	infra *SharedInfra,
	policyIdentity *PolicyIdentity,
	opts ...pulumi.ResourceOption,
) (*VirtualMachineServiceResult, error) {
	if infra.Networking == nil || infra.Networking.ComputeSubnet == nil {
		return nil, errVirtualMachineNetworkingRequired
	}
	if !common.InPublicNetwork(infra.Networks, svc) {
		return nil, fmt.Errorf("service %s: %w", serviceName, errPrivateVMServiceUnsupported)
	}
	if svc.HasHostPorts() {
		return nil, fmt.Errorf("service %s: %w", serviceName, errVMIngressRequired)
	}
	if strings.Contains(svc.GetPlatform(), "arm64") {
		return nil, fmt.Errorf("service %s: %w", serviceName, errArmVMServiceUnsupported)
	}
	envPlan, err := classifyVMEnvironment(ctx, infra.ConfigProvider, svc.Environment)
	if err != nil {
		return nil, fmt.Errorf("service %s environment: %w", serviceName, err)
	}
	if len(envPlan.secretRefs) > 0 && infra.KeyVaultIdentityID == nil {
		return nil, fmt.Errorf("service %s: %w", serviceName, errVMSecretsRequireKeyVault)
	}

	publicIP, err := network.NewPublicIPAddress(ctx, serviceName, &network.PublicIPAddressArgs{
		ResourceGroupName:        infra.ResourceGroup.Name,
		PublicIPAddressVersion:   pulumi.String("IPv4"),
		PublicIPAllocationMethod: pulumi.String("Static"),
		Sku: &network.PublicIPAddressSkuArgs{
			Name: pulumi.String("Standard"),
			Tier: pulumi.String("Regional"),
		},
		Tags: ServiceTags(serviceName),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating public IP for %s: %w", serviceName, err)
	}

	const (
		frontendName = "public"
		backendName  = "instances"
		probeName    = "health"
	)
	lbResourceID := pulumi.Sprintf(
		"%s/providers/Microsoft.Network/loadBalancers/%s", infra.ResourceGroup.ID(), serviceName)
	frontendID := pulumi.Sprintf("%s/frontendIPConfigurations/%s", lbResourceID, frontendName)
	backendID := pulumi.Sprintf("%s/backendAddressPools/%s", lbResourceID, backendName)
	probeID := pulumi.Sprintf("%s/probes/%s", lbResourceID, probeName)
	frontendRef := &network.SubResourceArgs{Id: frontendID.ToStringPtrOutput()}
	backendRef := &network.SubResourceArgs{Id: backendID.ToStringPtrOutput()}
	probeRef := &network.SubResourceArgs{Id: probeID.ToStringPtrOutput()}

	securityRules := network.SecurityRuleTypeArray{
		&network.SecurityRuleTypeArgs{
			Name:                     pulumi.String("health"),
			Priority:                 pulumi.Int(100),
			Direction:                pulumi.String("Inbound"),
			Access:                   pulumi.String("Allow"),
			Protocol:                 pulumi.String("Tcp"),
			SourceAddressPrefix:      pulumi.String("AzureLoadBalancer"),
			SourcePortRange:          pulumi.String("*"),
			DestinationAddressPrefix: pulumi.String("*"),
			DestinationPortRange:     pulumi.String(strconv.Itoa(vmHealthPort)),
		},
	}
	loadBalancingRules := make(network.LoadBalancingRuleArray, 0, len(svc.Ports))
	for i, port := range svc.Ports {
		protocol := azureProtocol(port)
		name := strings.ToLower(protocol) + "-" + strconv.Itoa(int(port.Target))
		securityRules = append(securityRules, &network.SecurityRuleTypeArgs{
			Name:                     pulumi.String(name),
			Priority:                 pulumi.Int(200 + i),
			Direction:                pulumi.String("Inbound"),
			Access:                   pulumi.String("Allow"),
			Protocol:                 pulumi.String(protocol),
			SourceAddressPrefix:      pulumi.String("Internet"),
			SourcePortRange:          pulumi.String("*"),
			DestinationAddressPrefix: pulumi.String("*"),
			DestinationPortRange:     pulumi.String(strconv.Itoa(int(port.Target))),
		})
		rule := &network.LoadBalancingRuleArgs{
			Name:                    pulumi.String(name),
			Protocol:                pulumi.String(protocol),
			FrontendIPConfiguration: frontendRef,
			BackendAddressPool:      backendRef,
			Probe:                   probeRef,
			FrontendPort:            pulumi.Int(port.Target),
			BackendPort:             pulumi.IntPtr(int(port.Target)),
			DisableOutboundSnat:     pulumi.Bool(true),
			EnableFloatingIP:        pulumi.Bool(false),
		}
		if protocol == azureProtocolTCP {
			rule.EnableTcpReset = pulumi.Bool(true)
		}
		loadBalancingRules = append(loadBalancingRules, rule)
	}

	nsg, err := network.NewNetworkSecurityGroup(ctx, serviceName, &network.NetworkSecurityGroupArgs{
		ResourceGroupName:        infra.ResourceGroup.Name,
		NetworkSecurityGroupName: pulumi.StringPtr(serviceName),
		SecurityRules:            securityRules,
		Tags:                     ServiceTags(serviceName),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating network security group for %s: %w", serviceName, err)
	}

	lb, err := network.NewLoadBalancer(ctx, serviceName, &network.LoadBalancerArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		LoadBalancerName:  pulumi.StringPtr(serviceName),
		Sku: &network.LoadBalancerSkuArgs{
			Name: pulumi.String("Standard"),
			Tier: pulumi.String("Regional"),
		},
		FrontendIPConfigurations: network.FrontendIPConfigurationArray{
			&network.FrontendIPConfigurationArgs{
				Name: pulumi.String(frontendName),
				PublicIPAddress: &network.PublicIPAddressTypeArgs{
					Id: publicIP.ID().ToStringOutput(),
				},
			},
		},
		BackendAddressPools: network.BackendAddressPoolArray{
			&network.BackendAddressPoolArgs{Name: pulumi.String(backendName)},
		},
		Probes: network.ProbeArray{
			&network.ProbeArgs{
				Name:              pulumi.String(probeName),
				Protocol:          pulumi.String("Http"),
				Port:              pulumi.Int(vmHealthPort),
				RequestPath:       pulumi.StringPtr("/"),
				IntervalInSeconds: pulumi.IntPtr(10),
				NumberOfProbes:    pulumi.IntPtr(3),
			},
		},
		LoadBalancingRules: loadBalancingRules,
		OutboundRules: network.OutboundRuleArray{
			&network.OutboundRuleArgs{
				Name:                     pulumi.String("egress"),
				Protocol:                 pulumi.String("All"),
				BackendAddressPool:       backendRef,
				FrontendIPConfigurations: network.SubResourceArray{frontendRef},
				IdleTimeoutInMinutes:     pulumi.IntPtr(4),
			},
		},
		Tags: ServiceTags(serviceName),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating load balancer for %s: %w", serviceName, err)
	}

	password, err := random.NewRandomPassword(ctx, serviceName+"-vm", &random.RandomPasswordArgs{
		Length:  pulumi.Int(32),
		Special: pulumi.Bool(true),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating VM credential for %s: %w", serviceName, err)
	}

	cloudInit := virtualMachineCloudInit(serviceName, image, svc, infra, policyIdentity, envPlan)
	customData := cloudInit.ApplyT(func(value string) string {
		return base64.StdEncoding.EncodeToString([]byte(value))
	}).(pulumi.StringOutput)

	identities := pulumi.StringArray{}
	if svc.Build != nil && infra.BuildInfra != nil {
		identities = append(identities, infra.BuildInfra.ManagedIdentityID())
	}
	if policyIdentity != nil {
		identities = append(identities, policyIdentity.ID)
	}
	if len(envPlan.secretRefs) > 0 {
		identities = append(identities, infra.KeyVaultIdentityID.ToStringPtrOutput().Elem())
	}
	var identity *compute.VirtualMachineScaleSetIdentityArgs
	if len(identities) > 0 {
		identity = &compute.VirtualMachineScaleSetIdentityArgs{
			Type:                   compute.ResourceIdentityTypeUserAssigned,
			UserAssignedIdentities: identities,
		}
	}

	vmOpts := append([]pulumi.ResourceOption{}, opts...)
	vmOpts = append(vmOpts,
		pulumi.DependsOn([]pulumi.Resource{lb}),
	)
	scaleSet, err := compute.NewVirtualMachineScaleSet(ctx, serviceName, &compute.VirtualMachineScaleSetArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		VmScaleSetName:    pulumi.StringPtr(serviceName),
		OrchestrationMode: pulumi.StringPtr("Uniform"),
		Overprovision:     pulumi.BoolPtr(false),
		Identity:          identity,
		Sku: &compute.SkuArgs{
			Name:     pulumi.StringPtr(virtualMachineSize(svc)),
			Tier:     pulumi.StringPtr("Standard"),
			Capacity: pulumi.Float64Ptr(float64(svc.GetReplicas())),
		},
		// customData changes must reimage VMSS instances for cloud-init to run.
		// A rolling max-surge upgrade creates a healthy replacement before it
		// deletes the old instance, keeping the load-balanced endpoint available.
		UpgradePolicy: &compute.UpgradePolicyArgs{
			Mode: compute.UpgradeModeRolling,
			RollingUpgradePolicy: &compute.RollingUpgradePolicyArgs{
				MaxBatchInstancePercent:               pulumi.IntPtr(50),
				MaxSurge:                              pulumi.BoolPtr(true),
				MaxUnhealthyInstancePercent:           pulumi.IntPtr(0),
				MaxUnhealthyUpgradedInstancePercent:   pulumi.IntPtr(0),
				RollbackFailedInstancesOnPolicyBreach: pulumi.BoolPtr(true),
			},
		},
		VirtualMachineProfile: &compute.VirtualMachineScaleSetVMProfileArgs{
			OsProfile: &compute.VirtualMachineScaleSetOSProfileArgs{
				AdminUsername:      pulumi.StringPtr("defang"),
				ComputerNamePrefix: pulumi.StringPtr(vmComputerNamePrefix(serviceName)),
				AdminPassword:      password.Result,
				CustomData:         customData.ToStringPtrOutput(),
				LinuxConfiguration: &compute.LinuxConfigurationArgs{
					DisablePasswordAuthentication: pulumi.BoolPtr(false),
					ProvisionVMAgent:              pulumi.BoolPtr(true),
				},
			},
			StorageProfile: &compute.VirtualMachineScaleSetStorageProfileArgs{
				ImageReference: &compute.ImageReferenceArgs{
					Publisher: pulumi.StringPtr("Canonical"),
					Offer:     pulumi.StringPtr("ubuntu-24_04-lts"),
					Sku:       pulumi.StringPtr("server"),
					Version:   pulumi.StringPtr("latest"),
				},
				OsDisk: &compute.VirtualMachineScaleSetOSDiskArgs{
					CreateOption: pulumi.String("FromImage"),
					Caching:      compute.CachingTypesReadOnly,
					ManagedDisk: &compute.VirtualMachineScaleSetManagedDiskParametersArgs{
						StorageAccountType: pulumi.StringPtr("StandardSSD_LRS"),
					},
				},
			},
			NetworkProfile: &compute.VirtualMachineScaleSetNetworkProfileArgs{
				HealthProbe: &compute.ApiEntityReferenceArgs{Id: probeID.ToStringPtrOutput()},
				NetworkInterfaceConfigurations: compute.VirtualMachineScaleSetNetworkConfigurationArray{
					&compute.VirtualMachineScaleSetNetworkConfigurationArgs{
						Name:    pulumi.String("primary"),
						Primary: pulumi.BoolPtr(true),
						NetworkSecurityGroup: &compute.SubResourceArgs{
							Id: nsg.ID().ToStringPtrOutput(),
						},
						IpConfigurations: compute.VirtualMachineScaleSetIPConfigurationArray{
							&compute.VirtualMachineScaleSetIPConfigurationArgs{
								Name:    pulumi.String("primary"),
								Primary: pulumi.BoolPtr(true),
								Subnet: &compute.ApiEntityReferenceArgs{
									Id: infra.Networking.ComputeSubnet.ID().ToStringPtrOutput(),
								},
								LoadBalancerBackendAddressPools: compute.SubResourceArray{
									&compute.SubResourceArgs{Id: backendID.ToStringPtrOutput()},
								},
							},
						},
					},
				},
			},
		},
		Tags: ServiceTags(serviceName),
	}, vmOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating VM scale set for %s: %w", serviceName, err)
	}

	return &VirtualMachineServiceResult{ScaleSet: scaleSet, PublicIPAddress: publicIP}, nil
}
