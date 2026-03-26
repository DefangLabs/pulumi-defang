package defanggcp

import "github.com/DefangLabs/pulumi-defang/provider/compose"

const defaultPostgresPort = 5432
const defaultRedisPort = 6379

// firstIngressPort returns the first configured port target, or the provided default port.
func firstIngressPort(ports []compose.ServicePortConfig, defaultPort int32) int32 {
	if len(ports) > 0 && ports[0].Target > 0 {
		return ports[0].Target
	}
	return defaultPort
}
