package common

import (
	"regexp"
	"strconv"
)

// Based on https://www.ietf.org/rfc/rfc3986.txt, using the pattern for query
// (which is a superset of path's `pchar`) but removing the single quote.
var healthcheckURLRegex = regexp.MustCompile(`(?i)(?:http://)?(?:localhost|127\.0\.0\.1)(?::(\d{1,5}))?([?/](?:[?/a-z0-9._~!$&()*+,;=:@-]|%[a-f0-9]{2}){0,333})?`)

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

// IsManagedService returns true if the service is a managed backing service (e.g., Postgres).
func IsManagedService(svc ServiceConfig) bool {
	return svc.Postgres != nil
}

// HasPort returns true if any port has the given mode.
func HasPort(svc ServiceConfig, mode string) bool {
	for _, p := range svc.Ports {
		if p.Mode == mode {
			return true
		}
	}
	return false
}

// NeedIngress returns true if any non-managed service in the map has ingress ports.
func NeedIngress(services map[string]ServiceConfig) bool {
	for _, svc := range services {
		if HasPort(svc, "ingress") && !IsManagedService(svc) {
			return true
		}
	}
	return false
}
