package common

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
)

var (
	ingressPort = compose.ServicePortConfig{Target: 80, Mode: compose.PortModeIngress}
	hostPort    = compose.ServicePortConfig{Target: 5432, Mode: compose.PortModeHost}
	// backendNet puts a service in a non-default network, which is private.
	backendNet = map[compose.NetworkID]compose.ServiceNetworkConfig{"backend": {}}
	// defaultNet puts a service explicitly in the default network.
	defaultNet = map[compose.NetworkID]compose.ServiceNetworkConfig{compose.DefaultNetwork: {}}
	// internalDefault marks the top-level default network as internal (private).
	internalDefault = compose.Networks{compose.DefaultNetwork: {Internal: true}}
)

func TestNeedPrivateZone(t *testing.T) {
	tests := []struct {
		name     string
		networks compose.Networks
		services compose.Services
		want     bool
	}{
		{"no services", nil, compose.Services{}, false},
		{
			"ingress-only service in the default network needs no private zone",
			nil,
			compose.Services{"web": {Ports: []compose.ServicePortConfig{ingressPort}}},
			false,
		},
		{"portless worker needs no private zone", nil, compose.Services{"worker": {}}, false},
		{
			"host-mode service needs a private zone (transitional)",
			nil,
			compose.Services{"db": {Ports: []compose.ServicePortConfig{hostPort}}},
			true,
		},
		{"managed postgres needs a private zone", nil, compose.Services{"pg": {Postgres: &compose.PostgresConfig{}}}, true},
		{"managed redis needs a private zone", nil, compose.Services{"cache": {Redis: &compose.RedisConfig{}}}, true},
		{
			"ingress service in a non-default (private) network needs a private zone",
			nil,
			compose.Services{"api": {Ports: []compose.ServicePortConfig{ingressPort}, Networks: backendNet}},
			true,
		},
		{
			"ingress service in an internal default network needs a private zone",
			internalDefault,
			compose.Services{"web": {Ports: []compose.ServicePortConfig{ingressPort}, Networks: defaultNet}},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NeedPrivateZone(tt.networks, tt.services); got != tt.want {
				t.Errorf("NeedPrivateZone() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNeedIngress(t *testing.T) {
	tests := []struct {
		name     string
		networks compose.Networks
		services compose.Services
		want     bool
	}{
		{
			"ingress service in the default network is public",
			nil,
			compose.Services{"web": {Ports: []compose.ServicePortConfig{ingressPort}}},
			true,
		},
		{
			"host-only service has no ingress",
			nil,
			compose.Services{"db": {Ports: []compose.ServicePortConfig{hostPort}}},
			false,
		},
		{
			"ingress service in a private network is NOT public",
			nil,
			compose.Services{"api": {Ports: []compose.ServicePortConfig{ingressPort}, Networks: backendNet}},
			false,
		},
		{
			"ingress service in an internal default network is NOT public",
			internalDefault,
			compose.Services{"web": {Ports: []compose.ServicePortConfig{ingressPort}, Networks: defaultNet}},
			false,
		},
		{
			"managed service with ingress ports is excluded",
			nil,
			compose.Services{"pg": {Ports: []compose.ServicePortConfig{ingressPort}, Postgres: &compose.PostgresConfig{}}},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NeedIngress(tt.networks, tt.services); got != tt.want {
				t.Errorf("NeedIngress() = %v, want %v", got, tt.want)
			}
		})
	}
}
