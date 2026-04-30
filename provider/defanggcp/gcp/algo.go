package gcp

import (
	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
)

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
	return !common.AcceptPublicTraffic(networks, *service) && // services with public IPs do not need a NAT GW
		common.AllowEgress(networks, *service) &&
		!common.IsManagedService(*service) &&
		!IsCloudRunService(service) &&
		!isCloudRunJob(service)
}

func IsCloudRunService(service *compose.ServiceConfig) bool {
	// Mode defaults to "ingress" when unset (matches compose.ServicePortConfig.IsIngress);
	// without this, a compose YAML omitting `mode:` falls through to Compute Engine and
	// the project Endpoints output returns a raw load-balancer IP instead of a Cloud Run URL.
	return len(service.Ports) == 1 && service.Ports[0].IsIngress()
}

func isCloudRunJob(service *compose.ServiceConfig) bool {
	return service.Restart == "no"
}
