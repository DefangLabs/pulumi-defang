package azure

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
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

// TestBuildIngressHostPortGetsInternalIngress is the case that motivated this
// change: on Container Apps a service is resolvable by name only if it has an
// ingress block, so a host-mode port must produce an internal ingress rather
// than none at all. Without it a sibling app addressing the service by name
// fails DNS resolution and the caller 502s.
func TestBuildIngressHostPortGetsInternalIngress(t *testing.T) {
	ingress := buildIngress(compose.ServiceConfig{
		Ports:    []compose.ServicePortConfig{{Target: 8080, Mode: compose.PortModeHost}},
		Networks: internalNet,
	}, topLevelNets)

	require.NotNil(t, ingress, "a host port must still get an ingress, or the service has no resolvable name")
	assert.False(t, boolInputValue(t, ingress.External))
	assert.Equal(t, 8080, intInputValue(t, ingress.TargetPort))
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
	assert.Nil(t, buildIngress(compose.ServiceConfig{Networks: internalNet}, topLevelNets))
}

// TestBuildIngressPrefersIngressPortOverHostPort: a service asking for both is
// asking to be load-balanced, and only an ingress port can be external.
func TestBuildIngressPrefersIngressPortOverHostPort(t *testing.T) {
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
