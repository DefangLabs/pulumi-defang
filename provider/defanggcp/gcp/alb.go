package gcp

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/certificatemanager"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudrunv2"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/dns"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/redis"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/sql"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const portModeIngress = "ingress"
const protoTCP = "tcp"

var (
	errHealthCheckPortMismatch = errors.New("health check port does not match the ingress target port")
	errUnsupportedProtocol     = errors.New("unsupported protocol")
	errNoTCPPort               = errors.New("at least one tcp port is needed for health check")
	errTooManyPorts            = errors.New("too many ports with protocol")
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
) error {
	if err := createInternalLoadBalancer(ctx, projectName, config, services); err != nil {
		return err
	}

	if err := createExternalLoadBalancers(ctx, projectName, config, services); err != nil {
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
	// Filter to services with ingress ports
	var ingressEntries []LBServiceEntry
	for _, e := range entries {
		if e.Config.HasIngressPorts() {
			ingressEntries = append(ingressEntries, e)
		}
	}
	if len(ingressEntries) == 0 {
		return nil
	}

	// Create public DNS A records for each ingress service when a domain is configured.
	// Mirrors the CD's getDomainAndCerts logic: one record for the main service domain
	// ({serviceName}.{domain}) and one per ingress port ({serviceName}--{port}.{domain}).
	if config.Domain != "" {
		ip := pulumi.StringArray{config.PublicIP.Address}
		for _, entry := range ingressEntries {
			svcDomain := entry.Name + "." + config.Domain
			if err := CreatePublicDNSRecord(ctx, config.PublicZoneId, svcDomain, "A", pulumi.Int(60), ip, opts...); err != nil {
				return err
			}
			for _, port := range entry.Config.Ports {
				if port.Mode != portModeIngress {
					continue
				}
				portDomain := fmt.Sprintf("%s--%d.%s", entry.Name, port.Target, config.Domain)
				if err := CreatePublicDNSRecord(
					ctx, config.PublicZoneId, portDomain, "A", pulumi.Int(60), ip, opts...,
				); err != nil {
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
		if _, err := certificatemanager.NewCertificateMapEntry(ctx, projectName+"-cert-map-entry",
			&certificatemanager.CertificateMapEntryArgs{
				Map:          certMap.Name,
				Certificates: pulumi.StringArray{config.WildcardCertId},
				Matcher:      pulumi.String("PRIMARY"),
			}, opts...); err != nil {
			return err
		}
	}

	urlMap, err := buildURLMap(ctx, projectName, ingressEntries, config.Region, opts...)
	if err != nil {
		return err
	}

	if err := createHTTPSForwardingRule(ctx, projectName, config.PublicIP, urlMap, certMap, opts...); err != nil {
		return err
	}

	return createHTTPRedirectForwardingRule(ctx, projectName, config.PublicIP, opts...)
}

//nolint:funlen,maintidx
func createInternalLoadBalancer(
	ctx *pulumi.Context,
	projectName string,
	config *SharedInfra,
	services []LBServiceEntry,
) error {
	var internalAlbServices []string
	var firstPrivateBackendID pulumi.StringPtrInput
	var privateHostRules compute.RegionUrlMapHostRuleArray
	var privatePathMatchers compute.RegionUrlMapPathMatcherArray
	for _, service := range services {
		// Managed services: always create private DNS regardless of ports.
		switch {
		case service.Config.Postgres != nil:
			if _, err := dns.NewRecordSet(ctx, projectName+"-"+service.Name+"-private-db-dns", &dns.RecordSetArgs{
				Name:        pulumi.String(internalServiceDns(service.Name)),
				Type:        pulumi.String("A"),
				Ttl:         pulumi.Int(60),
				ManagedZone: config.PrivateZone,
				Rrdatas:     pulumi.StringArray{service.PostgresInstance.PrivateIpAddress},
			}); err != nil {
				return err
			}
			continue
		case service.Config.Redis != nil:
			if _, err := dns.NewRecordSet(ctx, projectName+"-"+service.Name+"-private-redis-dns", &dns.RecordSetArgs{
				Name:        pulumi.String(internalServiceDns(service.Name)),
				Type:        pulumi.String("A"),
				Ttl:         pulumi.Int(60),
				ManagedZone: config.PrivateZone,
				Rrdatas:     pulumi.StringArray{service.RedisInstance.Host},
			}); err != nil {
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
			)
			if err != nil {
				return err
			}
			if firstPrivateBackendID == nil {
				firstPrivateBackendID = serviceBackend.ID()
			}

			internalServiceName := service.Name
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
			if len(service.Config.Ports) == 1 && service.Config.Ports[0].Mode == "ingress" {
				port := service.Config.Ports[0]
				portTargetStr := strconv.FormatInt(int64(port.Target), 10)
				portProto := port.GetProtocol()
				healthCheckPath, healthCheckPort := getHealthCheckPathAndPort(service.Config.HealthCheck)
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
					pulumi.DependsOn([]pulumi.Resource{firewall}),
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
				)
				if err != nil {
					return err
				}
				if firstPrivateBackendID == nil {
					firstPrivateBackendID = serviceBackend.ID()
				}

				internalServiceName := service.Name
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
				)
				if err != nil {
					return err
				}

				var tcpHealthCheckPort *uint32
				var firewallAllows compute.FirewallAllowArray
				// Try minimize the number of forwarding rules by grouping the ports by protocol
				protocolPorts := make(map[string][]uint32)
				for _, port := range service.Config.Ports {
					proto := port.GetProtocol()
					if proto != protoTCP && proto != "udp" {
						return fmt.Errorf("unsupported protocol %s: %w", proto, errUnsupportedProtocol)
					}
					portTarget := uint32(port.Target) //nolint:gosec // port numbers are always non-negative
					if tcpHealthCheckPort == nil && proto == protoTCP {
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
				)
				if err != nil {
					return err
				}

				healthCheckFirewall, err := compute.NewFirewall(ctx,
					service.Name,
					&compute.FirewallArgs{
						Network: config.VpcId,
						// Fixed health check IP ranges for internal passthrough NLB:
						// https://cloud.google.com/load-balancing/docs/health-checks#firewall_rules
						SourceRanges: pulumi.StringArray{
							pulumi.String("130.211.0.0/22"),
							pulumi.String("35.191.0.0/16"),
						},
						Allows: compute.FirewallAllowArray{&compute.FirewallAllowArgs{
							Protocol: pulumi.String(protoTCP),
							Ports:    pulumi.StringArray{pulumi.String(strconv.FormatUint(uint64(*tcpHealthCheckPort), 10))},
						}},
						TargetTags: pulumi.StringArray{
							pulumi.String(service.Name), // Matching compute.go instance template tag
						},
						Direction: pulumi.String("INGRESS"),
					},
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
					pulumi.DependsOn([]pulumi.Resource{healthCheckFirewall}),
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

						backendService, err := compute.NewRegionBackendService(ctx,
							service.Name+fmt.Sprintf("host-%v-backend-service", portsName),
							&compute.RegionBackendServiceArgs{
								Region:              pulumi.String(config.Region),
								LoadBalancingScheme: pulumi.String("INTERNAL"),
								Backends: compute.RegionBackendServiceBackendArray{
									&compute.RegionBackendServiceBackendArgs{
										Group:         service.InstanceGroup.InstanceGroup,
										BalancingMode: pulumi.String("CONNECTION"),
									},
								},
								Protocol: pulumi.String(strings.ToUpper(protocol)),
								// Protocol: pulumi.String("UNSPECIFIED"), // For passthrough NLB, protocol specified in the forwarding rule
								ConnectionDrainingTimeoutSec: pulumi.Int(0), // Make configurable?
								HealthChecks:                 healthCheck.ID(),
							},
						)
						if err != nil {
							return err
						}

						portsInput := pulumi.StringArray{}
						for _, port := range ports {
							portsInput = append(portsInput, pulumi.String(strconv.FormatUint(uint64(port), 10)))
						}
						// Create a forwarding rule
						_, err = compute.NewForwardingRule(ctx,
							service.Name+fmt.Sprintf("host-%v-forwarding-rule", portsName),
							&compute.ForwardingRuleArgs{
								LoadBalancingScheme: pulumi.String("INTERNAL"),
								IpProtocol:          pulumi.String(strings.ToUpper(protocol)),
								Network:             config.VpcId,
								Subnetwork:          config.SubnetId,
								Region:              pulumi.String(config.Region),
								BackendService:      backendService.SelfLink,
								Ports:               portsInput,
								// Multiple forwarding rules share the same IP so internal DNS works.
								IpAddress: internalNlbIP.Address,
							},
						)
						if err != nil {
							return err
						}
					}
				}

				if _, err := dns.NewRecordSet(ctx, projectName+"-"+service.Name+"-private-lb-dns", &dns.RecordSetArgs{
					Name:        pulumi.String(internalServiceDns(service.Name)),
					Type:        pulumi.String("A"),
					Ttl:         pulumi.Int(60),
					ManagedZone: config.PrivateZone,
					Rrdatas:     pulumi.StringArray{internalNlbIP.Address},
				}, pulumi.DependsOn([]pulumi.Resource{trafficFirewall})); err != nil {
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
				pulumi.ID(config.ProxySubnetId), nil,
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
			pulumi.DependsOn([]pulumi.Resource{regionalManagedProxySubnet}),
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
			pulumi.DependsOn([]pulumi.Resource{regionalManagedProxySubnet}),
		)
		if err != nil {
			return err
		}
		for _, serviceName := range internalAlbServices {
			if _, err := dns.NewRecordSet(ctx, projectName+"-"+serviceName+"-private-lb-dns", &dns.RecordSetArgs{
				Name:        pulumi.String(internalServiceDns(serviceName)),
				Type:        pulumi.String("A"),
				Ttl:         pulumi.Int(60),
				ManagedZone: config.PrivateZone,
				Rrdatas:     pulumi.StringArray{forwardingRule.IpAddress},
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func internalServiceDns(name string) string {
	return name + `.google.internal.`
}

func newCertMap(
	ctx *pulumi.Context,
	projectName string,
	opts ...pulumi.ResourceOption,
) (*certificatemanager.CertificateMapResource, error) {
	args := &certificatemanager.CertificateMapResourceArgs{
		Description: pulumi.String(projectName + " public load balancer certificate map"),
	}
	return certificatemanager.NewCertificateMapResource(ctx, projectName+"-cert-map", args, opts...)
}

func buildURLMap(
	ctx *pulumi.Context,
	projectName string,
	entries []LBServiceEntry,
	region string,
	opts ...pulumi.ResourceOption,
) (*compute.URLMap, error) {
	var firstBackendID pulumi.StringPtrInput
	var hostRules compute.URLMapHostRuleArray
	var pathMatchers compute.URLMapPathMatcherArray

	for _, entry := range entries {
		backendID, matcher, hostRule, err := buildLBEntry(ctx, entry, region, opts...)
		if err != nil {
			return nil, err
		}
		pathMatchers = append(pathMatchers, matcher)
		if hostRule != nil {
			hostRules = append(hostRules, hostRule)
		}
		if firstBackendID == nil {
			firstBackendID = backendID.ToStringOutput()
		}
	}

	return compute.NewURLMap(ctx, projectName+"-urlmap", &compute.URLMapArgs{
		DefaultService: firstBackendID,
		HostRules:      hostRules,
		PathMatchers:   pathMatchers,
	}, opts...)
}

// buildLBEntry creates backend resources for one LB service entry and returns the
// backend ID, path matcher, and optional host rule. Returns nil backendID if the
// entry has no applicable ports.
func buildLBEntry(
	ctx *pulumi.Context,
	entry LBServiceEntry,
	region string,
	opts ...pulumi.ResourceOption,
) (pulumi.IDOutput, *compute.URLMapPathMatcherArgs, *compute.URLMapHostRuleArgs, error) {
	if entry.CloudRunService != nil {
		return buildCloudRunLBEntry(ctx, entry, region, opts...)
	}
	if entry.InstanceGroup != nil {
		return buildMIGLBEntry(ctx, entry, opts...)
	}
	return pulumi.IDOutput{}, nil, nil, nil
}

func buildCloudRunLBEntry(
	ctx *pulumi.Context,
	entry LBServiceEntry,
	region string,
	opts ...pulumi.ResourceOption,
) (pulumi.IDOutput, *compute.URLMapPathMatcherArgs, *compute.URLMapHostRuleArgs, error) {
	neg, err := compute.NewRegionNetworkEndpointGroup(ctx, entry.Name+"-neg",
		&compute.RegionNetworkEndpointGroupArgs{
			NetworkEndpointType: pulumi.String("SERVERLESS"),
			Region:              pulumi.String(region),
			CloudRun: &compute.RegionNetworkEndpointGroupCloudRunArgs{
				Service: entry.CloudRunService.Name,
			},
		}, opts...)
	if err != nil {
		return pulumi.IDOutput{}, nil, nil, err
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
		return pulumi.IDOutput{}, nil, nil, err
	}

	matcher := &compute.URLMapPathMatcherArgs{
		Name: pulumi.String(entry.Name), DefaultService: backend.ID(),
	}
	var hostRule *compute.URLMapHostRuleArgs
	if entry.Config.DomainName != "" {
		hostRule = &compute.URLMapHostRuleArgs{
			Hosts:       pulumi.ToStringArray([]string{entry.Config.DomainName}),
			PathMatcher: pulumi.String(entry.Name),
		}
	}
	return backend.ID(), matcher, hostRule, nil
}

// buildMIGLBEntry creates an LB backend for the first ingress port of a Compute Engine MIG.
func buildMIGLBEntry(
	ctx *pulumi.Context,
	entry LBServiceEntry,
	opts ...pulumi.ResourceOption,
) (pulumi.IDOutput, *compute.URLMapPathMatcherArgs, *compute.URLMapHostRuleArgs, error) {
	for _, port := range entry.Config.Ports {
		if port.Mode != portModeIngress {
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
			return pulumi.IDOutput{}, nil, nil, err
		}

		backend, err := compute.NewBackendService(ctx, entry.Name+"-"+portStr+"-gce-backend",
			&compute.BackendServiceArgs{
				Protocol:            pulumi.String("HTTP"),
				LoadBalancingScheme: pulumi.String("EXTERNAL_MANAGED"),
				Backends: compute.BackendServiceBackendArray{
					&compute.BackendServiceBackendArgs{Group: entry.InstanceGroup.InstanceGroup},
				},
				HealthChecks: hc.ID(),
			}, append(opts, pulumi.DependsOn([]pulumi.Resource{entry.InstanceGroup}))...)
		if err != nil {
			return pulumi.IDOutput{}, nil, nil, err
		}

		matcher := &compute.URLMapPathMatcherArgs{
			Name: pulumi.String(entry.Name), DefaultService: backend.ID(),
		}
		var hostRule *compute.URLMapHostRuleArgs
		if entry.Config.DomainName != "" {
			hostRule = &compute.URLMapHostRuleArgs{
				Hosts:       pulumi.ToStringArray([]string{entry.Config.DomainName}),
				PathMatcher: pulumi.String(entry.Name),
			}
		}
		return backend.ID(), matcher, hostRule, nil
	}
	return pulumi.IDOutput{}, nil, nil, nil
}

func createHTTPSForwardingRule(
	ctx *pulumi.Context,
	projectName string,
	publicIP *compute.GlobalAddress,
	urlMap *compute.URLMap,
	certMap *certificatemanager.CertificateMapResource,
	opts ...pulumi.ResourceOption,
) error {
	certMapRef := certMap.ID().ApplyT(func(id string) (string, error) {
		return fmt.Sprintf("//certificatemanager.googleapis.com/%v", id), nil
	}).(pulumi.StringOutput)

	httpsProxy, err := compute.NewTargetHttpsProxy(ctx, projectName+"-https-proxy",
		&compute.TargetHttpsProxyArgs{
			UrlMap:         urlMap.SelfLink,
			CertificateMap: certMapRef,
		}, opts...)
	if err != nil {
		return err
	}

	_, err = compute.NewGlobalForwardingRule(ctx, projectName+"-https-rule",
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
	projectName string,
	publicIP *compute.GlobalAddress,
	opts ...pulumi.ResourceOption,
) error {
	redirectMap, err := compute.NewURLMap(ctx, projectName+"-http-urlmap",
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

	httpProxy, err := compute.NewTargetHttpProxy(ctx, projectName+"-http-proxy",
		&compute.TargetHttpProxyArgs{UrlMap: redirectMap.ID()}, opts...)
	if err != nil {
		return err
	}

	_, err = compute.NewGlobalForwardingRule(ctx, projectName+"-http-rule",
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

// Based on cd/aws/defang_service.ts
var healthcheckUrlRegex = regexp.MustCompile(
	`(?i)(?:http:\/\/)?(?:localhost|127\.0\.0\.1)(?::(\d{1,5}))?([?/](?:[?/a-z0-9._~!$&()*+,;=:@-]|%[a-f0-9]{2}){0,333})?`)

func getHealthCheckPathAndPort(hc *compose.HealthCheckConfig) (string, int) {
	path := "/"
	port := 80
	if hc == nil || len(hc.Test) < 1 || (hc.Test[0] != "CMD" && hc.Test[0] != "CMD-SHELL") {
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
