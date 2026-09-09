package azure

import (
	"reflect"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/app/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	internalNet = map[compose.NetworkID]compose.ServiceNetworkConfig{"internal": {}}
	defaultNet  = map[compose.NetworkID]compose.ServiceNetworkConfig{compose.DefaultNetwork: {}}
	// topLevelNets is the shape a project like DefangLabs/station declares: a
	// public default network alongside a private one.
	topLevelNets = compose.Networks{compose.DefaultNetwork: {}, "internal": {}}
)

// TestBuildIngressHostPortGetsInternalIngress is the case that motivated
// pulumi-defang#555: on Container Apps a service is resolvable by name only if
// it has an ingress block, so a host-mode port must produce an ingress rather
// than none at all. Without it a sibling app addressing the service by name
// fails DNS resolution and the caller 502s.
//
// It must also be TCP-transport with an explicit ExposedPort matching the
// target: that's pulumi-defang#558, the follow-up bug that shipped with #555.
// An HTTP-transport ingress resolves the name but the sibling's `<name>:<port>`
// TCP dial never completes (station.defang.io's 2026-09-09 outage).
func TestBuildIngressHostPortGetsInternalIngress(t *testing.T) {
	ingress := buildIngress(compose.ServiceConfig{
		Ports:    []compose.ServicePortConfig{{Target: 8080, Mode: compose.PortModeHost}},
		Networks: internalNet,
	}, topLevelNets)

	require.NotNil(t, ingress, "a host port must still get an ingress, or the service has no resolvable name")
	assert.False(t, boolInputValue(t, ingress.External))
	assert.Equal(t, 8080, intInputValue(t, ingress.TargetPort))
	assert.Equal(t, "tcp", stringPtrInputValue(t, ingress.Transport),
		"host-only ingress must be TCP transport, or <name>:<port> dials from a sibling never complete")
	assert.Equal(t, 8080, intPtrInputValue(t, ingress.ExposedPort),
		"ExposedPort must be set explicitly so <name>:<port> addressing matches the compose port")
}

// TestBuildIngressHostPortStaysInternalInPublicNetwork pins the transitional
// boundary: host mode is treated as private even on the public default network,
// matching common.ServiceFQDN, so this does not hand a host service a public
// hostname (pulumi-defang#253).
func TestBuildIngressHostPortStaysInternalInPublicNetwork(t *testing.T) {
	ingress := buildIngress(compose.ServiceConfig{
		Ports:    []compose.ServicePortConfig{{Target: 5432, Mode: compose.PortModeHost}},
		Networks: defaultNet,
	}, topLevelNets)

	require.NotNil(t, ingress)
	assert.False(t, boolInputValue(t, ingress.External),
		"host mode must not be externally exposed until public host exposure exists")
}

// TestBuildIngressHostOnlyMultiplePorts covers a service with several host
// ports and no ingress port: the first becomes the main TCP ingress, the rest
// become internal AdditionalPortMappings — all reachable by name and port.
func TestBuildIngressHostOnlyMultiplePorts(t *testing.T) {
	ingress := buildIngress(compose.ServiceConfig{
		Ports: []compose.ServicePortConfig{
			{Target: 8080, Mode: compose.PortModeHost},
			{Target: 9090, Mode: compose.PortModeHost},
			{Target: 9091, Mode: compose.PortModeHost},
		},
		Networks: internalNet,
	}, topLevelNets)

	require.NotNil(t, ingress)
	assert.Equal(t, 8080, intInputValue(t, ingress.TargetPort))
	assert.Equal(t, "tcp", stringPtrInputValue(t, ingress.Transport))
	require.Len(t, ingress.AdditionalPortMappings, 2)
	mappings := ingress.AdditionalPortMappings.(app.IngressPortMappingArray)
	assert.Equal(t, 9090, intInputValue2(t, mappings[0].(app.IngressPortMappingArgs).TargetPort))
	assert.Equal(t, 9091, intInputValue2(t, mappings[1].(app.IngressPortMappingArgs).TargetPort))
}

// TestBuildIngressExternalFollowsNetworks covers the ingress-mode matrix: the
// port mode asks for load-balanced exposure, the networks decide whether that
// exposure is public.
func TestBuildIngressExternalFollowsNetworks(t *testing.T) {
	tests := []struct {
		name         string
		networks     compose.Networks
		svcNetworks  map[compose.NetworkID]compose.ServiceNetworkConfig
		wantExternal bool
	}{
		{"default network is public", topLevelNets, defaultNet, true},
		{"private network is not public", topLevelNets, internalNet, false},
		{
			"internal default network is not public",
			compose.Networks{compose.DefaultNetwork: {Internal: true}}, defaultNet, false,
		},
		{"no networks declared anywhere is public", nil, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ingress := buildIngress(compose.ServiceConfig{
				Ports:    []compose.ServicePortConfig{{Target: 80, Mode: compose.PortModeIngress}},
				Networks: tt.svcNetworks,
			}, tt.networks)

			require.NotNil(t, ingress)
			assert.Equal(t, tt.wantExternal, boolInputValue(t, ingress.External))
		})
	}
}

// TestBuildIngressNoPortsGetsNoIngress keeps the one case that must stay nil: a
// worker with no published port is not reachable by anything.
func TestBuildIngressNoPortsGetsNoIngress(t *testing.T) {
	ingress := buildIngress(compose.ServiceConfig{Networks: internalNet}, topLevelNets)
	assert.Nil(t, ingress)
}

// TestBuildIngressKeepsIngressPortAndMapsHostPort: a service with both an
// ingress port and a host port keeps the ingress port as its (possibly
// external) main ingress, but the host port must still get an internal
// AdditionalPortMapping rather than being silently dropped — pulumi-defang#558.
func TestBuildIngressKeepsIngressPortAndMapsHostPort(t *testing.T) {
	ingress := buildIngress(compose.ServiceConfig{
		Ports: []compose.ServicePortConfig{
			{Target: 5432, Mode: compose.PortModeHost},
			{Target: 80, Mode: compose.PortModeIngress},
		},
		Networks: defaultNet,
	}, topLevelNets)

	require.NotNil(t, ingress)
	assert.Equal(t, 80, intInputValue(t, ingress.TargetPort))
	assert.True(t, boolInputValue(t, ingress.External))
	require.Len(t, ingress.AdditionalPortMappings, 1, "the host port must still be reachable, not dropped")
	mapping := ingress.AdditionalPortMappings.(app.IngressPortMappingArray)[0].(app.IngressPortMappingArgs)
	assert.Equal(t, 5432, intInputValue2(t, mapping.TargetPort))
	assert.False(t, boolInputValue2(t, mapping.External))
}

// buildIngress builds its args from plain literals, so the inputs are the
// concrete pulumi.Bool / pulumi.Int types rather than Outputs and can be read
// back directly.
func boolInputValue(t *testing.T, input pulumi.BoolPtrInput) bool {
	t.Helper()
	b, ok := input.(pulumi.Bool)
	require.True(t, ok, "expected a pulumi.Bool, got %T", input)
	return bool(b)
}

func intInputValue(t *testing.T, input pulumi.IntPtrInput) int {
	t.Helper()
	i, ok := input.(pulumi.Int)
	require.True(t, ok, "expected a pulumi.Int, got %T", input)
	return int(i)
}

// intPtrInputValue reads back the value pulumi.IntPtr(v) wraps. pulumi.IntPtr's
// concrete type is package-private, so this dereferences via reflection instead
// of a type assertion (the same approach stringPtrInputValue uses below).
func intPtrInputValue(t *testing.T, input pulumi.IntPtrInput) int {
	t.Helper()
	require.NotNil(t, input)

	value := reflect.ValueOf(input)
	require.Equal(t, reflect.Pointer, value.Kind())
	require.False(t, value.IsNil())
	require.Equal(t, reflect.Int, value.Elem().Kind())
	return int(value.Elem().Int())
}

// intInputValue2/boolInputValue2 read the non-pointer Int/Bool inputs used by
// IngressPortMappingArgs (TargetPort, External), as opposed to the *PtrInput
// fields on IngressArgs itself.
func intInputValue2(t *testing.T, input pulumi.IntInput) int {
	t.Helper()
	i, ok := input.(pulumi.Int)
	require.True(t, ok, "expected a pulumi.Int, got %T", input)
	return int(i)
}

func boolInputValue2(t *testing.T, input pulumi.BoolInput) bool {
	t.Helper()
	b, ok := input.(pulumi.Bool)
	require.True(t, ok, "expected a pulumi.Bool, got %T", input)
	return bool(b)
}
