package common

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
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

// NeedPublicIngress returns true if the project needs a public load balancer: any
// non-managed service that is in a public network AND exposes an ingress port.
// Networks decide public vs private; the ingress port only selects load-balanced
// exposure. A service with ingress ports in a private/internal network is NOT
// public and does not need the public LB.
func NeedPublicIngress(networks compose.Networks, services compose.Services) bool {
	for _, svc := range services {
		if svc.HasIngressPorts() && svc.Postgres == nil && svc.Redis == nil && InPublicNetwork(networks, svc) {
			return true
		}
	}
	return false
}

// DefangAppSubdomainKey is the recipe key that turns the platform-provided
// public subdomain (the defang.app delegate domain) off. Each provider declares
// it in its own recipe namespace; the name is shared so the three clouds cannot
// drift apart on the spelling.
const DefangAppSubdomainKey = "use-defang-app-subdomain"

// publicServicesWithoutDomain returns the sorted names of the services that would
// be published under the platform-provided public domain — the same predicate as
// NeedPublicIngress (an ingress port in a public network, and not a managed
// Postgres/Redis) — but that have no domainname of their own. Sorted so a warning
// built from it is deterministic across Go's randomized map iteration order.
func publicServicesWithoutDomain(networks compose.Networks, services compose.Services) []string {
	var degraded []string
	for name, svc := range services {
		if svc.HasIngressPorts() && svc.Postgres == nil && svc.Redis == nil &&
			InPublicNetwork(networks, svc) && svc.DomainName == "" {
			degraded = append(degraded, name)
		}
	}
	slices.Sort(degraded)
	return degraded
}

// ProjectPublicDomain returns the platform-provided public domain to publish this
// project's services under, or "" when there is none to use.
//
// It is the single gate all three providers share, and it deliberately returns a
// domain rather than a boolean: every downstream site already branches on
// `domain != ""` (GCP's wildcard cert and delegate zone, AWS's wildcard ACM cert
// and public A records, Azure's delegate zone and per-service CNAME), so an empty
// return switches the whole project onto the no-public-domain path each cloud
// already implements, with no new branching.
//
// When the recipe opts out, a service that would have been published under the
// domain and has no domainname of its own is NOT an error: it keeps the hostname
// its cloud assigns anyway — the ALB's DNS name on AWS, the Cloud Run URL on GCP,
// the azurecontainerapps.io name on Azure — and that hostname is what the provider
// already reports as the service's endpoint. So this warns and degrades. Failing
// the deploy instead would reject a project that has a working public endpoint,
// and an error raised from inside a Pulumi program surfaces as a stack failure
// with a partial apply, which is a punishing way to report a cosmetic difference.
//
// nativeHostname describes, for the warning, what the services fall back to.
func ProjectPublicDomain(
	ctx *pulumi.Context,
	useDefangAppSubdomain bool,
	domain string,
	networks compose.Networks,
	services compose.Services,
	nativeHostname string,
) string {
	if useDefangAppSubdomain {
		return domain
	}
	if degraded := publicServicesWithoutDomain(networks, services); len(degraded) > 0 {
		_ = ctx.Log.Warn(fmt.Sprintf(
			"%q is disabled, so no defang.app subdomain is created: %s reachable only at %s. "+
				"Set `domainname:` on a service to publish it under a domain you control.",
			DefangAppSubdomainKey, describeServices(degraded), nativeHostname), nil)
	}
	return ""
}

// describeServices renders a service-name list for a log line: `service "a"` for
// one, `services "a", "b"` for several, each quoted so an empty or odd name is
// still visible.
func describeServices(names []string) string {
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = strconv.Quote(n)
	}
	if len(names) == 1 {
		return "service " + quoted[0] + " is"
	}
	return "services " + strings.Join(quoted, ", ") + " are"
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
	// A service with no `networks:` section is implicitly in the "default" network
	// (compose-spec normalization), whether or not the project declares any
	// networks of its own. Applying that rule only when the project declared none
	// made an otherwise-public service read as non-public purely because some
	// unrelated network existed, which since this PR also gates its public
	// load-balancer attachment and FQDN, not just its DNS zone.
	inDefaultNetwork = inDefaultNetwork || len(service.Networks) == 0
	// A nil networks map reads Internal as false, so this also covers the
	// no-declared-networks case: the default network is public unless the project
	// declared it internal.
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
