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
func ParseHealthCheckPathPort(test []string) (path string, port int) {
	path = "/"
	if len(test) < 1 || (test[0] != "CMD" && test[0] != "CMD-SHELL") {
		return
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
			return
		}
	}
	return
}

// NeedIngress returns true if any non-managed service in the map has ingress ports.
func NeedIngress(services map[string]compose.ServiceConfig) bool {
	for _, svc := range services {
		if svc.HasIngressPorts() && svc.Postgres == nil && svc.Redis == nil {
			return true
		}
	}
	return false
}
