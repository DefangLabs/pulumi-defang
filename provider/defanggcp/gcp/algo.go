package gcp

import (
	"regexp"
	"strconv"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
)

func IsManagedService(service *compose.ServiceConfig) bool {
	if service.Postgres != nil {
		return true
	}
	if service.Redis != nil {
		return true
	}
	return false
}

func IsNetworkInternal(networks compose.Networks, networkId compose.NetworkID) bool {
	if networks == nil {
		return false
	}
	return networks[networkId].Internal
}

func AllowEgress(networks compose.Networks, service *compose.ServiceConfig) bool {
	if len(networks) == 0 {
		// No explicit networks defined; services with no explicit network membership
		// have implicit egress through the "default" network (compose-spec normalization).
		return len(service.Networks) == 0
	}
	// Egress is allowed if the service is in at least one non-internal network
	for serviceNetwork := range service.Networks {
		if _, ok := networks[serviceNetwork]; ok {
			return !IsNetworkInternal(networks, serviceNetwork)
		}
	}
	return false
}

func HasPort(service *compose.ServiceConfig, mode string) bool {
	for _, port := range service.Ports {
		if port.Mode == mode {
			return true
		}
	}
	return false
}

func InPublicNetwork(networks compose.Networks, service *compose.ServiceConfig) bool {
	if len(networks) == 0 {
		// No explicit networks defined; services with no explicit network membership
		// are implicitly in the non-internal "default" network (compose-spec normalization).
		return len(service.Networks) == 0
	}
	_, inDefaultNetwork := service.Networks["default"]
	return inDefaultNetwork && !IsNetworkInternal(networks, "default")
}

func InPrivateNetwork(networks compose.Networks, service *compose.ServiceConfig) bool {
	switch len(service.Networks) {
	case 0:
		return false // not in any network
	case 1:
		return !InPublicNetwork(networks, service)
	default:
		return true
	}
}

func AcceptPublicTraffic(networks compose.Networks, service *compose.ServiceConfig) bool {
	// A service accepts traffic from the public internet if it's in the "default" network
	// and the default network is not internal and has a "host" port.
	// Services will have been added to the "default" network if they didn't have a "networks" section.
	return InPublicNetwork(networks, service) && HasPort(service, "host")
}

// Based on cd/aws/defang_service.ts
var healthcheckUrlRegex = regexp.MustCompile(
	`(?i)(?:http:\/\/)?(?:localhost|127\.0\.0\.1)(?::(\d{1,5}))?([?/](?:[?/a-z0-9._~!$&()*+,;=:@-]|%[a-f0-9]{2}){0,333})?`)

func getHealthCheckPathAndPort(hc *compose.HealthCheckConfig) (string, int) {
	path := "/"
	port := 80
	if len(hc.Test) < 1 || (hc.Test[0] != "CMD" && hc.Test[0] != "CMD-SHELL") {
		return path, port
	}
	for _, arg := range hc.Test[1:] {
		if match := healthcheckUrlRegex.FindStringSubmatch(arg); match != nil {
			if match[1] != "" {
				if n, err := strconv.Atoi(match[1]); err == nil {
					port = n
				}
			}
			if match[2] != "" {
				path = match[2]
			}
		}
	}
	return path, port
}

func IsProjectUsingLLM(project *compose.Project) bool {
	for _, service := range project.Services {
		if isTruthy(service.LLM) {
			return true
		}
	}
	return false
}

func NeedIngress(services compose.Services) bool {
	for _, service := range services {
		// static files are served by the CDN and redis/postgres don't use HTTP
		if HasPort(&service, "ingress") && !IsManagedService(&service) {
			return true
		}
	}
	return false
}

// NeedNATGateway determines if a NAT Gateway is needed for the stack
// TODO: could depend on deployment mode and the number of services
func NeedNATGateway(networks compose.Networks, services compose.Services) bool {
	for _, service := range services {
		if needNATGateway(networks, &service) {
			return true
		}
	}
	return false
}

func needNATGateway(networks compose.Networks, service *compose.ServiceConfig) bool {
	return !AcceptPublicTraffic(networks, service) && // services with public IPs do not need a NAT GW
		AllowEgress(networks, service) &&
		!IsManagedService(service) &&
		!IsCloudRunService(service) &&
		!isCloudRunJob(service)
}

func IsCloudRunService(service *compose.ServiceConfig) bool {
	return len(service.Ports) == 1 && service.Ports[0].Mode == "ingress"
}

func isCloudRunJob(service *compose.ServiceConfig) bool {
	return service.Restart == "no"
}

func isTruthy(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		result, _ := strconv.ParseBool(v)
		return result
	case int:
		return v != 0
	case float64:
		return v != 0
	case nil:
		return false
	default:
		return true
	}
}
