package azure

import (
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
		}}
		_, err = CreateVirtualMachineService(
			ctx,
			"dns",
			pulumi.String("cunnie/sslip.io-dns-server:latest"),
			svc,
			&SharedInfra{
				ResourceGroup: rg,
				Networking:    &NetworkingResult{VNet: vnet, ComputeSubnet: subnet},
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
	require.Len(t, mocks.byTypeSuffix(":VirtualMachineScaleSet"), 1)
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
