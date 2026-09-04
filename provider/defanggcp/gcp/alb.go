package gcp

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/certificatemanager"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudrunv2"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/dns"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/redis"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/sql"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var (
	errHealthCheckPortMismatch = errors.New("health check port does not match the ingress target port")
	errUnsupportedProtocol     = errors.New("unsupported protocol")
	errNoTCPPort               = errors.New("at least one tcp port is needed for health check")
	errTooManyPorts            = errors.New("too many ports with protocol")
	errMultipleIngressPorts    = errors.New("multiple ingress ports are not supported for Compute Engine services")
	errDuplicateRoute          = errors.New("two services cannot share a load balancer route")
)

// LBServiceEntry holds the data needed to wire a service into the external load balancer.
// Exactly one of CloudRunJob, CloudRunService, InstanceGroup, PostgresInstance, or RedisInstance should be non-nil.
type LBServiceEntry struct {
	Name             string
	Config           compose.ServiceConfig
	CloudRunJob      *cloudrunv2.Job
	CloudRunService  *cloudrunv2.Service                 // non-nil for Cloud Run services
	InstanceGroup    *compute.RegionInstanceGroupManager // non-nil for Compute Engine services
	PostgresInstance *sql.DatabaseInstance
	RedisInstance    *redis.Instance
	PrivateFqdn      string
}

func CreateLoadBalancers(
	ctx *pulumi.Context,
	projectName string,
	services []LBServiceEntry,
	config *SharedInfra,
	opts ...pulumi.ResourceOption,
) error {
	if err := createInternalLoadBalancer(ctx, config, services, opts...); err != nil {
		return err
	}

	if err := createExternalLoadBalancers(ctx, projectName, config, services, opts...); err != nil {
		return err
	}

	return nil
}

// createExternalLoadBalancer creates a Global HTTPS Load Balancer for Cloud Run services with
// ingress ports. It mirrors the structure from the cd program's alb.go.
func createExternalLoadBalancers(
	ctx *pulumi.Context,
	projectName string,
	config *SharedInfra,
	entries []LBServiceEntry,
	opts ...pulumi.ResourceOption,
) error {
	// Filter to services that can be wired into the external LB. Managed services
	// (Postgres/Redis) and CloudRunJob entries have no HTTP backend and would
	// otherwise reach buildLBEntry's fall-through, which returns a zero-value
	// pulumi.IDOutput and panics when ToStringOutput is called on it.
	var ingressEntries []LBServiceEntry
	for _, e := range entries {
		if e.Config.HasIngressPorts() && (e.CloudRunService != nil || e.InstanceGroup != nil) {
			if e.InstanceGroup != nil && countIngressPorts(e.Config.Ports) > 1 {
				return fmt.Errorf(
					"service %s has multiple ingress ports; use at most one ingress port with any additional ports in host mode: %w",
					e.Name,
					errMultipleIngressPorts,
				)
			}
			ingressEntries = append(ingressEntries, e)
		}
	}
	if len(ingressEntries) == 0 {
		return nil
	}
	if err := validateExternalRoutes(ingressEntries, config.Domain); err != nil {
		return err
	}

	// Create public DNS A records for each ingress service when a domain is configured.
	// Mirrors the CD's getDomainAndCerts logic: one record for the main service domain
	// ({serviceName}.{domain}) and one per ingress port ({serviceName}--{port}.{domain}).
	if config.Domain != "" {
		ip := pulumi.StringArray{config.PublicIP.Address}
		for _, entry := range ingressEntries {
			for _, hostname := range delegateHostnames(entry, config.Domain) {
				if err := CreatePublicDNSRecord(ctx, config.PublicZoneId, hostname, "A", pulumi.Int(60), ip, opts...); err != nil {
					return err
				}
			}
		}
	}

	certMap, err := newCertMap(ctx, projectName, opts...)
	if err != nil {
		return err
	}

	if config.WildcardCertId != nil {
		// Logical name deliberately omits projectName, same as the VPC/firewalls in
		// gcp.go: Pulumi's default resource ID already prefixes it with
		// <pulumi-project>-<stack>, which includes projectName, so repeating it here
		// risked exceeding GCP's 63-char resource ID limit.
		if _, err := certificatemanager.NewCertificateMapEntry(ctx, "cert-map-entry",
			&certificatemanager.CertificateMapEntryArgs{
				Map:          certMap.Name,
				Certificates: pulumi.StringArray{config.WildcardCertId},
				Matcher:      pulumi.String("PRIMARY"),
			}, opts...); err != nil {
			return err
		}
	}

	urlMap, err := buildURLMap(ctx, ingressEntries, config, opts...)
	if err != nil {
		return err
	}

	if err := createHTTPSForwardingRule(ctx, config.PublicIP, urlMap, certMap, opts...); err != nil {
		return err
	}

	return createHTTPRedirectForwardingRule(ctx, config.PublicIP, opts...)
}

//nolint:funlen,maintidx
func createInternalLoadBalancer(
	ctx *pulumi.Context,
	config *SharedInfra,
	services []LBServiceEntry,
	opts ...pulumi.ResourceOption,
) error {
	var internalAlbServices []string
	var firstPrivateBackendID pulumi.StringPtrInput
	var privateHostRules compute.RegionUrlMapHostRuleArray
	var privatePathMatchers compute.RegionUrlMapPathMatcherArray
	for _, service := range services {
		// Managed services: always create private DNS regardless of ports.
		// Logical names below deliberately omit projectName: Pulumi's default
		// resource ID already prefixes it with <pulumi-project>-<stack>, which
		// includes projectName, so repeating it here risked exceeding GCP's
		// 63-char resource ID limit. They also carry no role suffix: the type
		// token (dns:RecordSet) already says what they are, and every branch
		// below is mutually exclusive, so a service gets at most one private
		// record set here.
		switch {
		case service.Config.Postgres != nil:
			if _, err := dns.NewRecordSet(ctx, service.Name, &dns.RecordSetArgs{
				Name:        pulumi.String(internalServiceDns(service.Name)),
				Type:        pulumi.String("A"),
				Ttl:         pulumi.Int(60),
				ManagedZone: config.PrivateZone,
				Rrdatas:     pulumi.StringArray{service.PostgresInstance.PrivateIpAddress},
			}, opts...); err != nil {
				return err
			}
			continue
		case service.Config.Redis != nil:
			if _, err := dns.NewRecordSet(ctx, service.Name, &dns.RecordSetArgs{
				Name:        pulumi.String(internalServiceDns(service.Name)),
				Type:        pulumi.String("A"),
				Ttl:         pulumi.Int(60),
				ManagedZone: config.PrivateZone,
				Rrdatas:     pulumi.StringArray{service.RedisInstance.Host},
			}, opts...); err != nil {
				return err
			}
			continue
		}

		if len(service.Config.Ports) == 0 {
			continue
		}
		switch {
		case service.CloudRunService != nil && service.PrivateFqdn != "":
			serviceNeg, err := compute.NewRegionNetworkEndpointGroup(
				ctx,
				service.Name,
				&compute.RegionNetworkEndpointGroupArgs{
					NetworkEndpointType: pulumi.String("SERVERLESS"),
					Region:              pulumi.String(config.Region),
					CloudRun: &compute.RegionNetworkEndpointGroupCloudRunArgs{
						Service: pulumi.StringPtrInput(service.CloudRunService.Name),
					},
				},
				opts...,
			)
			if err != nil {
				return err
			}

			serviceBackend, err := compute.NewRegionBackendService(
				ctx,
				"private-lb-cloudrun-backend",
				&compute.RegionBackendServiceArgs{
					Region:              pulumi.String(config.Region),
					Protocol:            pulumi.String("HTTPS"),
					LoadBalancingScheme: pulumi.String("INTERNAL_MANAGED"),
					Backends: compute.RegionBackendServiceBackendArray{
						&compute.RegionBackendServiceBackendArgs{
							Group: serviceNeg.ID(),
						},
					},
				},
				opts...,
			)
			if err != nil {
				return err
			}
			if firstPrivateBackendID == nil {
				firstPrivateBackendID = serviceBackend.ID()
			}

			internalServiceName := common.ServiceLabel(service.Name)
			privateHostRules = append(privateHostRules, &compute.RegionUrlMapHostRuleArgs{
				Hosts:       pulumi.StringArray{pulumi.String(internalServiceName)},
				PathMatcher: pulumi.String(pathMatcherName(internalServiceName)),
			})

			privatePathMatchers = append(privatePathMatchers, &compute.RegionUrlMapPathMatcherArgs{
				Name:           pulumi.String(pathMatcherName(internalServiceName)),
				DefaultService: serviceBackend.ID(),
			})
			internalAlbServices = append(internalAlbServices, service.Name)
		case service.InstanceGroup != nil && service.PrivateFqdn != "":
			// When there is only one ingress port, use the same ALB as the cloud run services
			if len(service.Config.Ports) == 1 && service.Config.Ports[0].IsIngress() {
				port := service.Config.Ports[0]
				portTargetStr := strconv.FormatInt(int64(port.Target), 10)
				portProto := port.GetProtocol()
				healthCheckPath, healthCheckPort := compose.GetHealthCheckPathAndPort(service.Config.HealthCheck)
				if int(port.Target) != healthCheckPort {
					return fmt.Errorf(
						"health check port %d does not match the ingress target port %d: %w",
						healthCheckPort, port.Target, errHealthCheckPortMismatch)
				}

				firewall, err := compute.NewFirewall(ctx,
					service.Name,
					&compute.FirewallArgs{
						Network: config.VpcId,
						// Fixed health check IP ranges for internal passthrough NLB:
						// https://cloud.google.com/load-balancing/docs/health-checks#firewall_rules
						SourceRanges: pulumi.StringArray{
							pulumi.String("130.211.0.0/22"),
							pulumi.String("35.191.0.0/16"),
							pulumi.String("0.0.0.0/0"), // Allow traffic from LB backend.  TODO: Can this be stricter?
						},
						Allows: compute.FirewallAllowArray{&compute.FirewallAllowArgs{
							Protocol: pulumi.String(portProto),
							Ports:    pulumi.StringArray{pulumi.String(portTargetStr)},
						}},
						TargetTags: pulumi.StringArray{
							pulumi.String(service.Name), // Matching compute.go instance template tag
						},
						Direction: pulumi.String("INGRESS"),
					},
					opts...,
				)
				if err != nil {
					return err
				}

				healthCheck, err := compute.NewHealthCheck(ctx,
					service.Name,
					&compute.HealthCheckArgs{
						CheckIntervalSec: pulumi.Int(5),
						TimeoutSec:       pulumi.Int(5),
						HttpHealthCheck: &compute.HealthCheckHttpHealthCheckArgs{
							Port:        pulumi.Int(port.Target),
							RequestPath: pulumi.String(healthCheckPath),
						},
					},
					append(opts, pulumi.DependsOn([]pulumi.Resource{firewall}))...,
				)
				if err != nil {
					return err
				}
				serviceBackend, err := compute.NewRegionBackendService(ctx,
					service.Name,
					&compute.RegionBackendServiceArgs{
						Region:              pulumi.String(config.Region),
						Protocol:            pulumi.String("HTTP"),
						LoadBalancingScheme: pulumi.String("INTERNAL_MANAGED"),
						Backends: compute.RegionBackendServiceBackendArray{
							&compute.RegionBackendServiceBackendArgs{
								Group: service.InstanceGroup.InstanceGroup,
							},
						},
						HealthChecks: healthCheck.ID(),
						PortName:     pulumi.String(fmt.Sprintf("port-%v-%v", portProto, port.Target)), // Matching compute.go
					},
					opts...,
				)
				if err != nil {
					return err
				}
				if firstPrivateBackendID == nil {
					firstPrivateBackendID = serviceBackend.ID()
				}

				internalServiceName := common.ServiceLabel(service.Name)
				privateHostRules = append(privateHostRules, &compute.RegionUrlMapHostRuleArgs{
					Hosts:       pulumi.StringArray{pulumi.String(internalServiceName)},
					PathMatcher: pulumi.String(pathMatcherName(internalServiceName)),
				})

				privatePathMatchers = append(privatePathMatchers, &compute.RegionUrlMapPathMatcherArgs{
					Name:           pulumi.String(pathMatcherName(internalServiceName)),
					DefaultService: serviceBackend.ID(),
				})
				internalAlbServices = append(internalAlbServices, service.Name)
			} else { // Host mode
				if len(service.Config.Ports) == 0 {
					continue
				}
				// Create a private IP for the service
				internalNlbIP, err := compute.NewAddress(ctx,
					service.Name,
					&compute.AddressArgs{
						Subnetwork:  config.SubnetId,
						AddressType: pulumi.String("INTERNAL"),
						Region:      pulumi.String(config.Region),
						Purpose:     pulumi.String("SHARED_LOADBALANCER_VIP"),
					},
					opts...,
				)
				if err != nil {
					return err
				}

				var tcpHealthCheckPort *uint32
				var firewallAllows compute.FirewallAllowArray
				// Try minimize the number of forwarding rules by grouping the ports by protocol
				protocolPorts := make(map[compose.PortProtocol][]uint32)
				for _, port := range service.Config.Ports {
					proto := port.GetProtocol()
					if proto != compose.PortProtocolTCP && proto != compose.PortProtocolUDP {
						return fmt.Errorf("unsupported protocol %s: %w", proto, errUnsupportedProtocol)
					}
					portTarget := uint32(port.Target) //nolint:gosec // port numbers are always non-negative
					if tcpHealthCheckPort == nil && proto == compose.PortProtocolTCP {
						tcpHealthCheckPort = &portTarget
					}
					protocolPorts[proto] = append(protocolPorts[proto], portTarget)
					firewallAllows = append(firewallAllows, &compute.FirewallAllowArgs{
						Protocol: pulumi.String(proto),
						Ports:    pulumi.StringArray{pulumi.String(strconv.FormatUint(uint64(portTarget), 10))},
					})
				}
				if tcpHealthCheckPort == nil {
					return fmt.Errorf(
						"at least one tcp port is needed for health check for service %s: %w",
						service.Name, errNoTCPPort)
				}

				trafficFirewall, err := compute.NewFirewall(ctx,
					service.Name,
					&compute.FirewallArgs{
						Network:      config.VpcId,
						SourceRanges: pulumi.StringArray{pulumi.String("0.0.0.0/0")}, // TODO: Can this be stricter?
						Allows:       firewallAllows,
						TargetTags: pulumi.StringArray{
							pulumi.String(service.Name), // Matching compute.go instance template tag
						},
						Direction: pulumi.String("INGRESS"),
					},
					opts...,
				)
				if err != nil {
					return err
				}

				// "-hc" distinguishes this from trafficFirewall above: both are
				// gcp:compute/firewall:Firewall for the same service, so they can't
				// share trafficFirewall's bare service.Name -- Pulumi rejects that
				// as a duplicate URN (same type + name + parent).
				healthCheckFirewall, err := compute.NewFirewall(ctx,
					service.Name+"-hc",
					&compute.FirewallArgs{
						Network: config.VpcId,
						// Fixed health check IP ranges for internal passthrough NLB:
						// https://cloud.google.com/load-balancing/docs/health-checks#firewall_rules
						SourceRanges: pulumi.StringArray{
							pulumi.String("130.211.0.0/22"),
							pulumi.String("35.191.0.0/16"),
						},
						Allows: compute.FirewallAllowArray{&compute.FirewallAllowArgs{
							Protocol: pulumi.String(compose.PortProtocolTCP),
							Ports:    pulumi.StringArray{pulumi.String(strconv.FormatUint(uint64(*tcpHealthCheckPort), 10))},
						}},
						TargetTags: pulumi.StringArray{
							pulumi.String(service.Name), // Matching compute.go instance template tag
						},
						Direction: pulumi.String("INGRESS"),
					},
					opts...,
				)
				if err != nil {
					return err
				}

				hcPortStr := strconv.FormatUint(uint64(*tcpHealthCheckPort), 10)
				healthCheck, err := compute.NewHealthCheck(ctx,
					service.Name+hcPortStr,
					&compute.HealthCheckArgs{
						CheckIntervalSec:   pulumi.Int(30),
						TimeoutSec:         pulumi.Int(10),
						UnhealthyThreshold: pulumi.Int(3),
						HealthyThreshold:   pulumi.Int(2),
						TcpHealthCheck: &compute.HealthCheckTcpHealthCheckArgs{
							Port: pulumi.Int(*tcpHealthCheckPort),
						},
					},
					append(opts, pulumi.DependsOn([]pulumi.Resource{healthCheckFirewall}))...,
				)
				if err != nil {
					return err
				}

				for protocol, allPorts := range protocolPorts {
					if len(allPorts) > 100 { // Artificial limit to prevent too many forwarding rules being created
						return fmt.Errorf("too many ports with protocol %v for service %s: %w", protocol, service.Name, errTooManyPorts)
					}
					// Max 5 ports per forwarding rule:
					// https://cloud.google.com/load-balancing/docs/forwarding-rule-concepts#port_specifications
					for ports := range slices.Chunk(allPorts, 5) {
						portsName := strings.Trim(strings.ReplaceAll(fmt.Sprint(ports), " ", "-"), "[]")
						// No "-backend-service"/"-forwarding-rule" suffix (the type token
						// already says what each is); protocol is included since two
						// protocols can chunk to the same port set, which would otherwise
						// collide on a shared logical name (a live smoketest, defang-mvp#3181,
						// hit this bug's sibling missing-separator variant: "smokeworkerhost-
						// 6379-backend-service", also 68 chars -- one more reason to keep it
						// short instead of just re-adding a "-" before "host").
						name := fmt.Sprintf("%s-%s-%s", service.Name, strings.ToLower(string(protocol)), portsName)

						backendService, err := compute.NewRegionBackendService(ctx, name,
							&compute.RegionBackendServiceArgs{
								Region:              pulumi.String(config.Region),
								LoadBalancingScheme: pulumi.String("INTERNAL"),
								Backends: compute.RegionBackendServiceBackendArray{
									&compute.RegionBackendServiceBackendArgs{
										Group:         service.InstanceGroup.InstanceGroup,
										BalancingMode: pulumi.String("CONNECTION"),
									},
								},
								Protocol: pulumi.String(strings.ToUpper(string(protocol))),
								// Protocol: pulumi.String("UNSPECIFIED"), // For passthrough NLB, protocol specified in the forwarding rule
								ConnectionDrainingTimeoutSec: pulumi.Int(0), // Make configurable?
								HealthChecks:                 healthCheck.ID(),
							},
							opts...,
						)
						if err != nil {
							return err
						}

						portsInput := pulumi.StringArray{}
						for _, port := range ports {
							portsInput = append(portsInput, pulumi.String(strconv.FormatUint(uint64(port), 10)))
						}
						// Create a forwarding rule
						_, err = compute.NewForwardingRule(ctx, name,
							&compute.ForwardingRuleArgs{
								LoadBalancingScheme: pulumi.String("INTERNAL"),
								IpProtocol:          pulumi.String(strings.ToUpper(string(protocol))),
								Network:             config.VpcId,
								Subnetwork:          config.SubnetId,
								Region:              pulumi.String(config.Region),
								BackendService:      backendService.SelfLink,
								Ports:               portsInput,
								// Multiple forwarding rules share the same IP so internal DNS works.
								IpAddress: internalNlbIP.Address,
							},
							opts...,
						)
						if err != nil {
							return err
						}
					}
				}

				if _, err := dns.NewRecordSet(ctx, service.Name, &dns.RecordSetArgs{
					Name:        pulumi.String(internalServiceDns(service.Name)),
					Type:        pulumi.String("A"),
					Ttl:         pulumi.Int(60),
					ManagedZone: config.PrivateZone,
					Rrdatas:     pulumi.StringArray{internalNlbIP.Address},
				}, append(opts, pulumi.DependsOn([]pulumi.Resource{trafficFirewall}))...); err != nil {
					return err
				}
			}
		}
	}

	if len(internalAlbServices) > 0 {
		var regionalManagedProxySubnet *compute.Subnetwork
		var err error
		if config.ProxySubnetId != "" {
			regionalManagedProxySubnet, err = compute.GetSubnetwork(ctx,
				"managed-proxy-subnet",
				pulumi.ID(config.ProxySubnetId), nil, opts...,
			)
		} else {
			regionalManagedProxySubnet, err = compute.NewSubnetwork(ctx,
				"managed-proxy-subnet",
				&compute.SubnetworkArgs{
					Purpose:     pulumi.String("REGIONAL_MANAGED_PROXY"),
					IpCidrRange: pulumi.String("10.10.0.0/16"),
					Region:      pulumi.String(config.Region),
					Role:        pulumi.String("ACTIVE"),
					Network:     config.VpcId,
				},
				opts...,
			)
		}
		if err != nil {
			return err
		}

		privateUrlMap, err := compute.NewRegionUrlMap(ctx,
			"private-lb-urlmap",
			&compute.RegionUrlMapArgs{
				Region:         pulumi.String(config.Region),
				DefaultService: firstPrivateBackendID,
				HostRules:      privateHostRules,
				PathMatchers:   privatePathMatchers,
			},
			opts...,
		)
		if err != nil {
			return err
		}

		privateHttpProxy, err := compute.NewRegionTargetHttpProxy(ctx,
			"private-lb-http-proxy",
			&compute.RegionTargetHttpProxyArgs{
				Region: pulumi.String(config.Region),
				UrlMap: privateUrlMap.SelfLink,
			},
			append(opts, pulumi.DependsOn([]pulumi.Resource{regionalManagedProxySubnet}))...,
		)
		if err != nil {
			return err
		}
		// TODO: Currently only support HTTP traffic for internal ALB
		forwardingRule, err := compute.NewForwardingRule(ctx,
			"private-lb-forwarding-rule",
			&compute.ForwardingRuleArgs{
				Target:              privateHttpProxy.SelfLink,
				Region:              pulumi.String(config.Region),
				Network:             config.VpcId,
				Subnetwork:          config.SubnetId,
				PortRange:           pulumi.String("80"),
				LoadBalancingScheme: pulumi.String("INTERNAL_MANAGED"),
			},
			append(opts, pulumi.DependsOn([]pulumi.Resource{regionalManagedProxySubnet}))...,
		)
		if err != nil {
			return err
		}
		for _, serviceName := range internalAlbServices {
			if _, err := dns.NewRecordSet(ctx, serviceName, &dns.RecordSetArgs{
				Name:        pulumi.String(internalServiceDns(serviceName)),
				Type:        pulumi.String("A"),
				Ttl:         pulumi.Int(60),
				ManagedZone: config.PrivateZone,
				Rrdatas:     pulumi.StringArray{forwardingRule.IpAddress},
			}, opts...); err != nil {
				return err
			}
		}
	}
	return nil
}

func internalServiceDns(name string) string {
	// ServiceLabel so the record name matches the PrivateFqdn handle
	// (project.go/service.go) and the DEFANG_FQDN injected on the CE path.
	return common.ServiceLabel(name) + `.google.internal.`
}

func countIngressPorts(ports []compose.ServicePortConfig) int {
	var count int
	for _, port := range ports {
		if port.IsIngress() {
			count++
		}
	}
	return count
}

func newCertMap(
	ctx *pulumi.Context,
	projectName string,
	opts ...pulumi.ResourceOption,
) (*certificatemanager.CertificateMapResource, error) {
	args := &certificatemanager.CertificateMapResourceArgs{
		Description: pulumi.String(projectName + " public load balancer certificate map"),
	}
	// Logical name deliberately omits projectName: Pulumi's default resource ID
	// already prefixes it with <pulumi-project>-<stack>, which includes
	// projectName, so repeating it here risked exceeding GCP's 63-char resource
	// ID limit.
	return certificatemanager.NewCertificateMapResource(ctx, "cert-map", args, opts...)
}

// delegateHostnames returns the hostnames the provider publishes DNS records for
// on behalf of one ingress service: "<service>.<domain>", plus
// "<service>--<port>.<domain>" for each ingress port. Empty when no delegate
// domain is configured (e.g. the standalone Service path).
//
// Both callers read from this one list -- createExternalLoadBalancers creates a
// DNS A record per name, buildURLMap puts the same names in the service's host
// rule -- so a name that resolves always has a route.
func delegateHostnames(entry LBServiceEntry, domain string) []string {
	domain = common.NormalizeDNS(domain)
	if domain == "" {
		return nil
	}
	label := common.ServiceLabel(entry.Name)
	hostnames := []string{label + "." + domain}
	for _, port := range entry.Config.Ports {
		if !port.IsIngress() {
			continue
		}
		hostnames = append(hostnames, fmt.Sprintf("%s--%d.%s", label, port.Target, domain))
	}
	return hostnames
}

// routeHostnames returns every hostname that must select this service's path
// matcher: the provider-generated delegate names plus every custom-domain name
// that the service asks Certificate Manager to cover. Keeping the BYOD half on
// common.ByodHostnames is important: PR #499 gives the service's domainname and
// its default-network aliases DNS/certificate treatment from that same list.
func routeHostnames(entry LBServiceEntry, domain string) []string {
	hosts := delegateHostnames(entry, domain)
	for _, hostname := range common.ByodHostnames(entry.Config) {
		hostname = common.NormalizeDNS(hostname)
		if hostname != "" && !slices.Contains(hosts, hostname) {
			hosts = append(hosts, hostname)
		}
	}

	// Unlike a classic Application Load Balancer, this global EXTERNAL_MANAGED
	// load balancer considers an explicit port part of the host match. Its HTTPS
	// forwarding rule listens on 443, so cover both forms clients send in Host
	// and HTTP/2 :authority ("example.com" and "example.com:443").
	withHTTPSPort := make([]string, 0, len(hosts)*2)
	for _, hostname := range hosts {
		withHTTPSPort = append(withHTTPSPort, hostname, hostname+":443")
	}
	return withHTTPSPort
}

// validateExternalRoutes rejects duplicate path matchers and host patterns. It
// runs before any external DNS, NEG, or backend resources are registered so
// callers receive this actionable error instead of a duplicate-URN or API error
// from a partially built load balancer.
func validateExternalRoutes(entries []LBServiceEntry, domain string) error {
	hostOwner := map[string]string{}    // host pattern -> service that serves it
	matcherOwner := map[string]string{} // path matcher name -> service it routes to
	for _, entry := range entries {
		matcher := pathMatcherName(entry.Name)
		if owner, taken := matcherOwner[matcher]; taken {
			return fmt.Errorf("services %q and %q both reduce to load balancer route %q: %w",
				owner, entry.Name, matcher, errDuplicateRoute)
		}
		matcherOwner[matcher] = entry.Name

		for _, host := range routeHostnames(entry, domain) {
			if owner, taken := hostOwner[host]; taken {
				return fmt.Errorf(
					"services %q and %q both claim hostname %q: %w", owner, entry.Name, host, errDuplicateRoute,
				)
			}
			hostOwner[host] = entry.Name
		}
	}
	return nil
}

func buildURLMap(
	ctx *pulumi.Context,
	entries []LBServiceEntry,
	config *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*compute.URLMap, error) {
	var firstBackendID pulumi.StringPtrInput
	var hostRules compute.URLMapHostRuleArray
	var pathMatchers compute.URLMapPathMatcherArray

	for _, entry := range entries {
		backendID, matcher, err := buildLBEntry(ctx, entry, config.Region, opts...)
		if err != nil {
			return nil, err
		}
		// Skip entries with no applicable backend: appending a nil matcher panics in
		// the SDK, and ToStringOutput on a zero-value IDOutput panics on a nil receiver.
		if matcher == nil {
			continue
		}
		name := pathMatcherName(entry.Name)
		pathMatchers = append(pathMatchers, matcher)

		// A path matcher is only reachable through a host rule that names it, so
		// every hostname pointed at this LB needs to appear in one. Without this,
		// only BYOD names got a rule and every delegate name fell through to
		// DefaultService -- i.e. to the first ingress service. See #373.
		hosts := routeHostnames(entry, config.Domain)
		if len(hosts) > 0 {
			hostRules = append(hostRules, &compute.URLMapHostRuleArgs{
				Hosts:       pulumi.ToStringArray(hosts),
				PathMatcher: pulumi.String(name),
			})
		}

		// The first ingress service in TopologicalSort order becomes the URL map's
		// DefaultService: where an unmatched Host header lands, i.e. the bare
		// project domain or any host pointed at this IP from outside the project.
		// Same as the legacy CD (gcpcd/alb.go, firstPublicBackendID).
		// TODO(#373): decide which service should own the bare project domain.
		if firstBackendID == nil {
			firstBackendID = backendID.ToStringOutput()
		}
	}

	// Logical name deliberately omits projectName: Pulumi's default resource ID
	// already prefixes it with <pulumi-project>-<stack>, which includes
	// projectName, so repeating it here risked exceeding GCP's 63-char resource
	// ID limit.
	return compute.NewURLMap(ctx, "urlmap", &compute.URLMapArgs{
		DefaultService: firstBackendID,
		HostRules:      hostRules,
		PathMatchers:   pathMatchers,
	}, opts...)
}

// buildLBEntry creates backend resources for one LB service entry and returns the
// backend ID and its path matcher. Returns a nil matcher if the entry has no
// applicable ports. Host rules are built by the caller, which owns the routing.
func buildLBEntry(
	ctx *pulumi.Context,
	entry LBServiceEntry,
	region string,
	opts ...pulumi.ResourceOption,
) (pulumi.IDOutput, *compute.URLMapPathMatcherArgs, error) {
	if entry.CloudRunService != nil {
		return buildCloudRunLBEntry(ctx, entry, region, opts...)
	}
	if entry.InstanceGroup != nil {
		return buildMIGLBEntry(ctx, entry, opts...)
	}
	return pulumi.IDOutput{}, nil, nil
}

func buildCloudRunLBEntry(
	ctx *pulumi.Context,
	entry LBServiceEntry,
	region string,
	opts ...pulumi.ResourceOption,
) (pulumi.IDOutput, *compute.URLMapPathMatcherArgs, error) {
	neg, err := compute.NewRegionNetworkEndpointGroup(ctx, entry.Name+"-neg",
		&compute.RegionNetworkEndpointGroupArgs{
			NetworkEndpointType: pulumi.String("SERVERLESS"),
			Region:              pulumi.String(region),
			CloudRun: &compute.RegionNetworkEndpointGroupCloudRunArgs{
				Service: entry.CloudRunService.Name,
			},
		}, opts...)
	if err != nil {
		return pulumi.IDOutput{}, nil, err
	}

	backend, err := compute.NewBackendService(ctx, entry.Name+"-backend",
		&compute.BackendServiceArgs{
			Protocol:            pulumi.String("HTTPS"),
			LoadBalancingScheme: pulumi.String("EXTERNAL_MANAGED"),
			Backends: compute.BackendServiceBackendArray{
				&compute.BackendServiceBackendArgs{Group: neg.ID()},
			},
		}, opts...)
	if err != nil {
		return pulumi.IDOutput{}, nil, err
	}

	return backend.ID(), &compute.URLMapPathMatcherArgs{
		Name: pulumi.String(pathMatcherName(entry.Name)), DefaultService: backend.ID(),
	}, nil
}

// buildMIGLBEntry creates an LB backend for the first ingress port of a Compute Engine MIG.
func buildMIGLBEntry(
	ctx *pulumi.Context,
	entry LBServiceEntry,
	opts ...pulumi.ResourceOption,
) (pulumi.IDOutput, *compute.URLMapPathMatcherArgs, error) {
	for _, port := range entry.Config.Ports {
		// Use IsIngress() so a default (empty) Mode is treated as ingress, matching
		// HasIngressPorts() — otherwise the filter in createExternalLoadBalancers admits
		// the entry but we fall through to the zero-value return, which panics later.
		if !port.IsIngress() {
			continue
		}
		portStr := strconv.Itoa(int(port.Target))
		hc, err := compute.NewHealthCheck(ctx, entry.Name+"-"+portStr+"-public-lb-hc",
			&compute.HealthCheckArgs{
				CheckIntervalSec: pulumi.Int(5),
				TimeoutSec:       pulumi.Int(5),
				TcpHealthCheck: &compute.HealthCheckTcpHealthCheckArgs{
					Port: pulumi.Int(port.Target),
				},
			}, opts...)
		if err != nil {
			return pulumi.IDOutput{}, nil, err
		}

		backend, err := compute.NewBackendService(ctx, entry.Name+"-"+portStr+"-gce-backend",
			&compute.BackendServiceArgs{
				Protocol:            pulumi.String("HTTP"),
				LoadBalancingScheme: pulumi.String("EXTERNAL_MANAGED"),
				Backends: compute.BackendServiceBackendArray{
					migBackend(entry),
				},
				HealthChecks: hc.ID(),
				PortName:     pulumi.String(fmt.Sprintf("port-%v-%v", port.GetProtocol(), port.Target)),
			}, append(opts, pulumi.DependsOn([]pulumi.Resource{entry.InstanceGroup}))...)
		if err != nil {
			return pulumi.IDOutput{}, nil, err
		}

		return backend.ID(), &compute.URLMapPathMatcherArgs{
			Name: pulumi.String(pathMatcherName(entry.Name)), DefaultService: backend.ID(),
		}, nil
	}
	return pulumi.IDOutput{}, nil, nil
}

func migBackend(entry LBServiceEntry) *compute.BackendServiceBackendArgs {
	backend := &compute.BackendServiceBackendArgs{Group: entry.InstanceGroup.InstanceGroup}
	if entry.Config.HasHostPorts() {
		backend.BalancingMode = pulumi.String("RATE")
		backend.MaxRatePerInstance = pulumi.Float64(10000)
	}
	return backend
}

// Logical names in createHTTPSForwardingRule and createHTTPRedirectForwardingRule
// deliberately omit projectName: Pulumi's default resource ID already prefixes it
// with <pulumi-project>-<stack>, which includes projectName, so repeating it here
// risked exceeding GCP's 63-char resource ID limit.
func createHTTPSForwardingRule(
	ctx *pulumi.Context,
	publicIP *compute.GlobalAddress,
	urlMap *compute.URLMap,
	certMap *certificatemanager.CertificateMapResource,
	opts ...pulumi.ResourceOption,
) error {
	certMapRef := certMap.ID().ApplyT(func(id string) (string, error) {
		return fmt.Sprintf("//certificatemanager.googleapis.com/%v", id), nil
	}).(pulumi.StringOutput)

	httpsProxy, err := compute.NewTargetHttpsProxy(ctx, "https-proxy",
		&compute.TargetHttpsProxyArgs{
			UrlMap:         urlMap.SelfLink,
			CertificateMap: certMapRef,
		}, opts...)
	if err != nil {
		return err
	}

	_, err = compute.NewGlobalForwardingRule(ctx, "https-rule",
		&compute.GlobalForwardingRuleArgs{
			Target:              httpsProxy.SelfLink,
			IpAddress:           publicIP.Address,
			PortRange:           pulumi.String("443"),
			LoadBalancingScheme: pulumi.String("EXTERNAL_MANAGED"),
		}, opts...)
	return err
}

func createHTTPRedirectForwardingRule(
	ctx *pulumi.Context,
	publicIP *compute.GlobalAddress,
	opts ...pulumi.ResourceOption,
) error {
	redirectMap, err := compute.NewURLMap(ctx, "http-urlmap",
		&compute.URLMapArgs{
			DefaultUrlRedirect: &compute.URLMapDefaultUrlRedirectArgs{
				HttpsRedirect:        pulumi.Bool(true),
				RedirectResponseCode: pulumi.String("MOVED_PERMANENTLY_DEFAULT"),
				StripQuery:           pulumi.Bool(false),
			},
		}, opts...)
	if err != nil {
		return err
	}

	httpProxy, err := compute.NewTargetHttpProxy(ctx, "http-proxy",
		&compute.TargetHttpProxyArgs{UrlMap: redirectMap.ID()}, opts...)
	if err != nil {
		return err
	}

	_, err = compute.NewGlobalForwardingRule(ctx, "http-rule",
		&compute.GlobalForwardingRuleArgs{
			IpAddress:           publicIP.Address,
			IpProtocol:          pulumi.String("TCP"),
			PortRange:           pulumi.String("80"),
			Target:              httpProxy.ID(),
			LoadBalancingScheme: pulumi.String("EXTERNAL_MANAGED"),
		}, opts...)
	return err
}

var pathMatcherNameRegex = regexp.MustCompile(`[^a-z0-9-]`)

// RegionUrlMap: field 'resource.pathMatchers[0].name' must be a match of regex '(?:[a-z](?:[-a-z0-9]{0,61}[a-z0-9])?)'
// Which is lowercase alphanumeric, with optional dashes, and must start with a letter and end with a letter or number.
func pathMatcherName(name string) string {
	name = strings.ToLower(name)
	name = pathMatcherNameRegex.ReplaceAllLiteralString(name, "-") // Replace non-alphanumeric and non-dash with dash
	for len(name) > 0 && (name[0] < 'a' || name[0] > 'z') {        // Must start with a letter
		name = name[1:] // Remove leading non-letter characters
	}
	for len(name) > 0 && name[len(name)-1] == '-' { // Must end with a letter or number
		name = name[:len(name)-1] // Remove trailing dashes
	}
	return name
}

// healthcheck path/port parsing moved to provider/compose.GetHealthCheckPathAndPort
// so the Azure Container Apps provider can share the same logic.
