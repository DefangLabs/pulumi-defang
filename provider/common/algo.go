package common

import (
	"regexp"
	"strconv"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
)

// Based on https://www.ietf.org/rfc/rfc3986.txt, using the pattern for query
// (which is a superset of path's `pchar`) but removing the single quote.
var healthcheckURLRegex = regexp.MustCompile(
	`(?i)(?:http://)?(?:localhost|127\.0\.0\.1)(?::(\d{1,5}))?([?/](?:[?/a-z0-9._~!$&()*+,;=:@-]|%[a-f0-9]{2}){0,333})?`,
)

// ParseHealthCheckPathPort parses the health check path and port from a CMD/CMD-SHELL test command.
// Returns path (default "/") and port (0 if not specified).
func ParseHealthCheckPathPort(test []string) (string, int) {
	path := "/"
	port := 0
	if len(test) < 1 || (test[0] != "CMD" && test[0] != "CMD-SHELL") {
		return path, port
	}
	for _, arg := range test[1:] {
		if match := healthcheckURLRegex.FindStringSubmatch(arg); match != nil {
			if match[1] != "" {
				if n, err := strconv.Atoi(match[1]); err == nil {
					port = n
				}
			}
			if match[2] != "" {
				path = match[2]
			}
			return path, port
		}
	}
	return path, port
}

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
	if len(networks) == 0 {
		// No explicit networks defined; services with no explicit network membership
		// are implicitly in the non-internal "default" network (compose-spec normalization).
		return len(service.Networks) == 0
	}
	_, inDefaultNetwork := service.Networks[compose.DefaultNetwork]
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
