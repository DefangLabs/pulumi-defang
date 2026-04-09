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
	return len(service.Ports) == 1 && service.Ports[0].Mode == "ingress"
}

func isCloudRunJob(service *compose.ServiceConfig) bool {
	return service.Restart == "no"
}
