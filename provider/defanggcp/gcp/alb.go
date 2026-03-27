package gcp

import (
	"fmt"
	"strconv"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/certificatemanager"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudrunv2"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const portModeIngress = "ingress"

// LBServiceEntry holds the data needed to wire a service into the external load balancer.
// Exactly one of Service or InstanceGroup should be non-nil.
type LBServiceEntry struct {
	Name          string
	Service       *cloudrunv2.Service                 // non-nil for Cloud Run services
	InstanceGroup *compute.RegionInstanceGroupManager // non-nil for Compute Engine services
	Config        compose.ServiceConfig
}

// CreateExternalLoadBalancer creates a Global HTTPS Load Balancer for Cloud Run services with
// ingress ports. It mirrors the structure from the cd program's alb.go.
func CreateExternalLoadBalancer(
	ctx *pulumi.Context,
	projectName string,
	config *GlobalConfig,
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
		if backendID == nil {
			continue
		}
		pathMatchers = append(pathMatchers, matcher)
		if hostRule != nil {
			hostRules = append(hostRules, hostRule)
		}
		if firstBackendID == nil {
			firstBackendID = backendID
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
) (pulumi.StringPtrInput, *compute.URLMapPathMatcherArgs, *compute.URLMapHostRuleArgs, error) {
	if entry.Service != nil {
		return buildCloudRunLBEntry(ctx, entry, region, opts...)
	}
	if entry.InstanceGroup != nil {
		return buildMIGLBEntry(ctx, entry, opts...)
	}
	return nil, nil, nil, nil
}

func buildCloudRunLBEntry(
	ctx *pulumi.Context,
	entry LBServiceEntry,
	region string,
	opts ...pulumi.ResourceOption,
) (pulumi.StringPtrInput, *compute.URLMapPathMatcherArgs, *compute.URLMapHostRuleArgs, error) {
	neg, err := compute.NewRegionNetworkEndpointGroup(ctx, entry.Name+"-neg",
		&compute.RegionNetworkEndpointGroupArgs{
			NetworkEndpointType: pulumi.String("SERVERLESS"),
			Region:              pulumi.String(region),
			CloudRun: &compute.RegionNetworkEndpointGroupCloudRunArgs{
				Service: entry.Service.Name,
			},
		}, opts...)
	if err != nil {
		return nil, nil, nil, err
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
		return nil, nil, nil, err
	}

	matcher := &compute.URLMapPathMatcherArgs{
		Name: pulumi.String(entry.Name), DefaultService: backend.ID(),
	}
	var hostRule *compute.URLMapHostRuleArgs
	if entry.Config.DomainName != nil {
		hostRule = &compute.URLMapHostRuleArgs{
			Hosts:       pulumi.ToStringArray([]string{*entry.Config.DomainName}),
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
) (pulumi.StringPtrInput, *compute.URLMapPathMatcherArgs, *compute.URLMapHostRuleArgs, error) {
	for _, port := range entry.Config.Ports {
		if port.Mode != portModeIngress {
			continue
		}
		portStr := strconv.Itoa(port.Target)
		hc, err := compute.NewHealthCheck(ctx, entry.Name+"-"+portStr+"-public-lb-hc",
			&compute.HealthCheckArgs{
				CheckIntervalSec: pulumi.Int(5),
				TimeoutSec:       pulumi.Int(5),
				TcpHealthCheck: &compute.HealthCheckTcpHealthCheckArgs{
					Port: pulumi.Int(port.Target),
				},
			}, opts...)
		if err != nil {
			return nil, nil, nil, err
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
			return nil, nil, nil, err
		}

		matcher := &compute.URLMapPathMatcherArgs{
			Name: pulumi.String(entry.Name), DefaultService: backend.ID(),
		}
		var hostRule *compute.URLMapHostRuleArgs
		if entry.Config.DomainName != nil {
			hostRule = &compute.URLMapHostRuleArgs{
				Hosts:       pulumi.ToStringArray([]string{*entry.Config.DomainName}),
				PathMatcher: pulumi.String(entry.Name),
			}
		}
		return backend.ID(), matcher, hostRule, nil
	}
	return nil, nil, nil, nil
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
