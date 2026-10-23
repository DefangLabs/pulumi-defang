package common

import (
	"github.com/DefangLabs/pulumi-defang/provider/compose"
)

// NeedIngress returns true if any non-managed service in the map has ingress ports.
func NeedIngress(services compose.Services) bool {
	for _, svc := range services {
		if svc.HasIngressPorts() && svc.Postgres == nil && svc.Redis == nil {
			return true
		}
	}
	return false
}

func AcceptPublicTraffic(networks compose.Networks, service compose.ServiceConfig) bool {
	// A service accepts traffic from the public internet if it's in the "default" network
	// and the default network is not internal and has a "host" port.
	_, inDefaultNetwork := service.Networks[compose.DefaultNetwork]
	// Services will have been added to the "default" network if they didn't have a "networks" section.
	inDefaultNetwork = inDefaultNetwork || len(service.Networks) == 0
	return inDefaultNetwork && !IsNetworkInternal(networks, compose.DefaultNetwork) && service.HasHostPorts()
}

func IsManagedService(service compose.ServiceConfig) bool {
	return service.Postgres != nil || service.Redis != nil
}

func IsNetworkInternal(networks compose.Networks, networkId compose.NetworkID) bool {
	return networks[networkId].Internal
}

func InPublicNetwork(networks compose.Networks, service compose.ServiceConfig) bool {
	_, inDefaultNetwork := service.Networks[compose.DefaultNetwork]
	if len(networks) == 0 {
		// No explicit networks defined; services with no explicit network membership
		// are implicitly in the non-internal "default" network (compose-spec normalization).
		return inDefaultNetwork || len(service.Networks) == 0
	}
	return inDefaultNetwork && !IsNetworkInternal(networks, compose.DefaultNetwork)
}

func InPrivateNetwork(networks compose.Networks, service compose.ServiceConfig) bool {
	switch len(service.Networks) {
	case 0:
		return false
	case 1:
		return !InPublicNetwork(networks, service)
	default:
		return true
	}
}

func AllowEgress(networks compose.Networks, service compose.ServiceConfig) bool {
	// Egress is allowed if the service is in at least one non-internal network
	for n := range service.Networks {
		if !IsNetworkInternal(networks, n) {
			return true
		}
	}
	return len(service.Networks) == 0 // if no networks specified, assume default non-internal network
}

func IsProjectUsingLLM(services compose.Services) bool {
	for _, svc := range services {
		if svc.LLM != nil {
			return true
		}
	}
	return false
}
