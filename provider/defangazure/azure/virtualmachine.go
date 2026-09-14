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

func vmDockerFlags(svc compose.ServiceConfig) (string, string) {
	flags := make([]string, 0, 2*len(svc.Ports)+4)
	command := make([]string, 0, len(svc.Command)+len(svc.Entrypoint))
	if len(svc.Entrypoint) > 0 {
		flags = append(flags, "--entrypoint", shellQuote(svc.Entrypoint[0]))
		for _, arg := range svc.Entrypoint[1:] {
			command = append(command, shellQuote(arg))
		}
	}
	for _, arg := range svc.Command {
		command = append(command, shellQuote(arg))
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
	parts := []pulumi.StringInput{pulumi.String("--env " + shellQuote("DEFANG_SERVICE="+serviceName))}
	if etag != "" {
		parts = append(parts, pulumi.String("--env "+shellQuote("DEFANG_ETAG="+etag)))
	}
	if policyIdentity != nil {
		parts = append(parts, policyIdentity.ClientID.ApplyT(func(clientID string) string {
			return "--env " + shellQuote("AZURE_CLIENT_ID="+clientID)
		}).(pulumi.StringOutput))
	}
	for key, value := range common.Sorted(svc.Environment) {
		if static, ok := compose.StaticEnvValue(value); ok {
			resolved := ""
			if static != nil {
				resolved = *static
			}
			parts = append(parts, pulumi.String("--env "+shellQuote(key+"="+resolved)))
			continue
		}
		parts = append(parts, value.ToStringPtrOutput().ApplyT(func(resolved *string) string {
			if resolved == nil {
				return "--env " + shellQuote(key+"=")
			}
			return "--env " + shellQuote(key+"="+*resolved)
		}).(pulumi.StringOutput))
	}
	return pulumi.StringArray(parts).ToStringArrayOutput().ApplyT(func(values []string) string {
		return strings.Join(values, " ")
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
tenant=$(printf '%%s===' "$payload" | base64 -d 2>/dev/null | jq -er .tid)
refresh=$(curl -fsS -X POST -H 'Content-Type: application/x-www-form-urlencoded' \
  --data-urlencode grant_type=access_token \
  --data-urlencode "service=$registry" \
  --data-urlencode "tenant=$tenant" \
  --data-urlencode "access_token=$aad" \
  "https://$registry/oauth2/exchange" | jq -er .refresh_token)
printf '%%s' "$refresh" | docker login "$registry" --username 00000000-0000-0000-0000-000000000000 --password-stdin
`, shellQuote(registry), shellQuote(identityID))
		},
	).(pulumi.StringOutput)
}

//nolint:funlen // the cloud-init document is clearer when kept as one template
func virtualMachineCloudInit(
	serviceName string,
	image pulumi.StringInput,
	svc compose.ServiceConfig,
	infra *SharedInfra,
	policyIdentity *PolicyIdentity,
) pulumi.StringOutput {
	containerName := svc.GetContainerName(serviceName)
	dockerFlags, command := vmDockerFlags(svc)
	env := vmEnvironment(serviceName, svc, infra.Etag, policyIdentity)
	login := acrLoginScript(infra, svc)
	quotedImage := image.ToStringOutput().ApplyT(shellQuote).(pulumi.StringOutput)

	template := `#cloud-config
package_update: true
packages:
  - docker.io
  - curl
  - jq
write_files:
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
      ExecStartPre=-/usr/bin/docker rm -f %s
      ExecStart=/usr/bin/docker run --pull=always --rm --name=%s %s %s %s %s
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
	return pulumi.All(login, env, quotedImage).ApplyT(func(values []any) string {
		return fmt.Sprintf(
			template,
			indent(values[0].(string)),
			strconv.Quote(containerName),
			vmHealthPort,
			serviceName,
			serviceName,
			containerName,
			containerName,
			dockerFlags,
			values[1].(string),
			values[2].(string),
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

	cloudInit := virtualMachineCloudInit(serviceName, image, svc, infra, policyIdentity)
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
	var identity *compute.VirtualMachineScaleSetIdentityArgs
	if len(identities) > 0 {
		identity = &compute.VirtualMachineScaleSetIdentityArgs{
			Type:                   compute.ResourceIdentityTypeUserAssigned,
			UserAssignedIdentities: identities,
		}
	}

	vmOpts := append([]pulumi.ResourceOption{}, opts...)
	vmOpts = append(vmOpts, pulumi.DependsOn([]pulumi.Resource{lb}))
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
		UpgradePolicy: &compute.UpgradePolicyArgs{Mode: compute.UpgradeModeAutomatic},
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
