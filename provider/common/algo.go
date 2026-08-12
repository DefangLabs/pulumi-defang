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

// NeedIngress returns true if the project needs a public load balancer: any
// non-managed service that is in a public network AND exposes an ingress port.
// Networks decide public vs private; the ingress port only selects load-balanced
// exposure. A service with ingress ports in a private/internal network is NOT
// public and does not need the public LB.
func NeedIngress(networks compose.Networks, services compose.Services) bool {
	for _, svc := range services {
		if svc.HasIngressPorts() && svc.Postgres == nil && svc.Redis == nil && InPublicNetwork(networks, svc) {
			return true
		}
	}
	return false
}

// NeedPrivateZone reports whether the project needs a private DNS zone. A private
// zone holds internal records for: a service in a private network (networks
// decide public/private), a host-mode service (transitional — kept so default-
// network host services keep their internal name, see pulumi-defang#253), or a
// managed Postgres/Redis. A project of only public ingress services needs none.
func NeedPrivateZone(networks compose.Networks, services compose.Services) bool {
	for _, svc := range services {
		if InPrivateNetwork(networks, svc) || svc.HasHostPorts() || IsManagedService(svc) {
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
