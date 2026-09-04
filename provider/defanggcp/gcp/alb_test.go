package gcp

import (
	"strings"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudrunv2"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/require"
)

type albResource struct {
	name   string
	typeof string
	inputs resource.PropertyMap
	parent string
}

type albMocks struct {
	resources []albResource
}

func (m *albMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	var parent string
	if args.RegisterRPC != nil {
		parent = args.RegisterRPC.GetParent()
	}
	m.resources = append(m.resources, albResource{
		name: args.Name, typeof: args.TypeToken, inputs: args.Inputs, parent: parent,
	})
	return args.Name + "_id", args.Inputs, nil
}

func (m *albMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func TestCreateLoadBalancersMIGHostModeExternalBackendUsesRateAndPortName(t *testing.T) {
	mocks := &albMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		mig, err := testMIG(ctx, "api")
		if err != nil {
			return err
		}
		return CreateLoadBalancers(ctx, "proj", []LBServiceEntry{{
			Name:          "api",
			InstanceGroup: mig,
			PrivateFqdn:   "api.google.internal",
			Config: testServiceConfig([]compose.ServicePortConfig{
				{Target: 3000, Mode: compose.PortModeIngress},
				{Target: 8081, Mode: compose.PortModeHost},
			}),
		}}, testInfra(ctx))
	}, pulumi.WithMocks("proj", "stack", mocks))
	require.NoError(t, err)

	backend := requireResource(t, mocks.resources, "gcp:compute/backendService:BackendService", "api-3000-gce-backend")
	requirePropertyString(t, backend.inputs, "portName", "port-tcp-3000")
	backends := backend.inputs["backends"].ArrayValue()
	require.Len(t, backends, 1)
	require.Equal(t, "RATE", backends[0].ObjectValue()["balancingMode"].StringValue())
	require.InEpsilon(t, 10000.0, backends[0].ObjectValue()["maxRatePerInstance"].NumberValue(), 0)
}

func TestCreateLoadBalancersMIGSingleIngressLeavesBalancingModeUnset(t *testing.T) {
	mocks := &albMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		mig, err := testMIG(ctx, "api")
		if err != nil {
			return err
		}
		return CreateLoadBalancers(ctx, "proj", []LBServiceEntry{{
			Name:          "api",
			InstanceGroup: mig,
			PrivateFqdn:   "api.google.internal",
			Config: testServiceConfig([]compose.ServicePortConfig{
				{Target: 3000, Mode: compose.PortModeIngress},
			}),
		}}, testInfra(ctx))
	}, pulumi.WithMocks("proj", "stack", mocks))
	require.NoError(t, err)

	backend := requireResource(t, mocks.resources, "gcp:compute/backendService:BackendService", "api-3000-gce-backend")
	requirePropertyString(t, backend.inputs, "portName", "port-tcp-3000")
	backendArgs := backend.inputs["backends"].ArrayValue()[0].ObjectValue()
	require.False(t, backendArgs.HasValue("balancingMode"))
	require.False(t, backendArgs.HasValue("maxRatePerInstance"))
}

func TestCreateLoadBalancersMIGMultipleIngressPortsReturnsActionableError(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		mig, err := testMIG(ctx, "api")
		if err != nil {
			return err
		}
		return CreateLoadBalancers(ctx, "proj", []LBServiceEntry{{
			Name:          "api",
			InstanceGroup: mig,
			PrivateFqdn:   "api.google.internal",
			Config: testServiceConfig([]compose.ServicePortConfig{
				{Target: 3000, Mode: compose.PortModeIngress},
				{Target: 8080, Mode: compose.PortModeIngress},
			}),
		}}, testInfra(ctx))
	}, pulumi.WithMocks("proj", "stack", &albMocks{}))
	require.Error(t, err)
	require.Contains(t, err.Error(), "service api has multiple ingress ports")
	require.Contains(t, err.Error(), "use at most one ingress port")
}

// createInternalLoadBalancer's host-mode branch (a single non-ingress TCP
// port, e.g. a Redis-like service on 6379) never forwarded opts to any of
// the resources it creates. Live smoketest result (defang-mvp#3181): every
// one of them tried to use the ambient default "gcp" provider instead of the
// explicit one this stack configures, and defang-playground-dev disables
// that default -- "Default provider for 'gcp' disabled ... must use an
// explicit provider." This exercises the exact failing shape and asserts
// opts (here, an explicit Parent) actually reaches each resource.
func TestCreateLoadBalancersHostModePropagatesOpts(t *testing.T) {
	mocks := &albMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		mig, err := testMIG(ctx, "smokeworker")
		if err != nil {
			return err
		}
		parent, err := compute.NewNetwork(ctx, "test-parent", &compute.NetworkArgs{})
		if err != nil {
			return err
		}
		return CreateLoadBalancers(ctx, "proj", []LBServiceEntry{{
			Name:          "smokeworker",
			InstanceGroup: mig,
			PrivateFqdn:   "smokeworker.google.internal",
			Config: testServiceConfig([]compose.ServicePortConfig{
				{Target: 6379, Mode: compose.PortModeHost},
			}),
		}}, testInfra(ctx), pulumi.Parent(parent))
	}, pulumi.WithMocks("proj", "stack", mocks))
	require.NoError(t, err)

	for _, typ := range []string{
		"gcp:compute/address:Address",
		"gcp:compute/firewall:Firewall",
		"gcp:compute/healthCheck:HealthCheck",
		"gcp:compute/regionBackendService:RegionBackendService",
		"gcp:compute/forwardingRule:ForwardingRule",
	} {
		found := false
		for _, r := range mocks.resources {
			if r.typeof == typ {
				found = true
				// Pulumi always assigns *some* parent (the implicit stack
				// pseudo-resource when none is given), so a plain non-empty
				// check would pass even with opts dropped -- assert it's
				// specifically our explicit parent.
				require.Contains(t, r.parent, "test-parent",
					"%s %q has parent %q, not the explicit one from opts -- opts not propagated", typ, r.name, r.parent)
			}
		}
		require.True(t, found, "no resource of type %s was created", typ)
	}

	// Two firewalls exist for one host-mode service (traffic + health-check
	// source ranges): a live smoketest hit "Duplicate resource URN" because
	// both used the bare service name. Pulumi's mocks don't enforce URN
	// uniqueness themselves, so check it explicitly.
	seen := map[string]bool{}
	for _, r := range mocks.resources {
		key := r.typeof + "::" + r.name
		require.False(t, seen[key], "duplicate resource: type=%s name=%q", r.typeof, r.name)
		seen[key] = true
	}

	// A live smoketest hit "smokeworkerhost-6379-backend-service" (68 chars,
	// over GCP's 63-char limit) from a missing separator plus a redundant
	// "-backend-service" type suffix. Assert the fixed, protocol-qualified
	// name (protocol matters: two protocols chunking to the same port set
	// would otherwise collide on one logical name).
	for _, typ := range []string{
		"gcp:compute/regionBackendService:RegionBackendService",
		"gcp:compute/forwardingRule:ForwardingRule",
	} {
		requireResource(t, mocks.resources, typ, "smokeworker-tcp-6379")
	}
}

// urlMapRouting is the routing table of the external URL map, decoded from the
// mocked resource inputs so tests can ask "where does this Host header land?"
// instead of picking apart nested PropertyValues.
type urlMapRouting struct {
	defaultService string
	hostToMatcher  map[string]string // host pattern -> path matcher name
	matcherService map[string]string // path matcher name -> backend id
	hostRuleCount  int
}

// backendFor resolves a Host header the way GCP does: find the host rule listing
// an exact host or the most specific leading-* pattern, follow it to its path
// matcher, and take that matcher's default service. The provider builds a global
// EXTERNAL_MANAGED load balancer, so the port (when present) remains part of the
// match. Falls back to the URL map default when nothing matches -- exactly the
// #373 failure mode, so the tests can assert on routing rather than structure.
func (r urlMapRouting) backendFor(hostHeader string) string {
	hostHeader = strings.ToLower(hostHeader)
	if matcher, ok := r.hostToMatcher[hostHeader]; ok {
		return r.matcherService[matcher]
	}

	bestPatternLength := -1
	bestMatcher := ""
	for pattern, matcher := range r.hostToMatcher {
		if !strings.HasPrefix(pattern, "*") || !strings.HasSuffix(hostHeader, pattern[1:]) {
			continue
		}
		if len(pattern) > bestPatternLength {
			bestPatternLength = len(pattern)
			bestMatcher = matcher
		}
	}
	if bestMatcher == "" {
		return r.defaultService
	}
	return r.matcherService[bestMatcher]
}

func requireURLMapRouting(t *testing.T, resources []albResource) urlMapRouting {
	t.Helper()
	urlMap := requireResource(t, resources, "gcp:compute/uRLMap:URLMap", "urlmap")
	routing := urlMapRouting{
		hostToMatcher:  map[string]string{},
		matcherService: map[string]string{},
	}
	if v, ok := urlMap.inputs["defaultService"]; ok {
		routing.defaultService = v.StringValue()
	}
	if v, ok := urlMap.inputs["hostRules"]; ok {
		for _, rule := range v.ArrayValue() {
			fields := rule.ObjectValue()
			matcher := fields["pathMatcher"].StringValue()
			hosts := fields["hosts"].ArrayValue()
			require.NotEmpty(t, hosts, "host rule for path matcher %q has no hosts; GCP rejects that", matcher)
			routing.hostRuleCount++
			for _, host := range hosts {
				name := host.StringValue()
				_, dup := routing.hostToMatcher[name]
				require.False(t, dup, "hostname %q appears in more than one host rule", name)
				routing.hostToMatcher[name] = matcher
			}
		}
	}
	if v, ok := urlMap.inputs["pathMatchers"]; ok {
		for _, matcher := range v.ArrayValue() {
			fields := matcher.ObjectValue()
			routing.matcherService[fields["name"].StringValue()] = fields["defaultService"].StringValue()
		}
	}
	// Every host rule must name a path matcher that exists, or GCP rejects the map.
	for host, matcher := range routing.hostToMatcher {
		require.Contains(t, routing.matcherService, matcher,
			"host %q points at path matcher %q, which no path matcher defines", host, matcher)
	}
	return routing
}

// publicDNSNames returns the hostnames the provider put in A records. Read the
// deployed record name rather than its Pulumi logical name, then strip the FQDN
// trailing dot because URL-map hosts match the HTTP Host header without it.
func publicDNSNames(resources []albResource) []string {
	var names []string
	for _, r := range resources {
		if r.typeof != "gcp:dns/recordSet:RecordSet" || r.inputs["type"].StringValue() != "A" {
			continue
		}
		names = append(names, strings.TrimSuffix(r.inputs["name"].StringValue(), "."))
	}
	return names
}

// api's delegate hostname under testInfraWithDomain's "proj.example.com" domain,
// reused across tests that need a hostname api itself would resolve to.
const testAPIDelegateHostname = "api.proj.example.com"

// Before #373 a service only got a URL-map host rule when it had a BYOD
// `domainname`. Every other hostname -- including the "<service>.<domain>" names
// the provider itself creates DNS records for -- fell through to DefaultService,
// so in a multi-ingress project every name reached the FIRST service. This is the
// regression test for that: ui's own hostnames must reach ui's backend.
func TestCreateLoadBalancersRoutesEachServiceDelegateHostnamesToItsOwnBackend(t *testing.T) {
	mocks := &albMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return CreateLoadBalancers(ctx, "proj", []LBServiceEntry{
			testCloudRunEntry(t, ctx, "api", 8080),
			testCloudRunEntry(t, ctx, "ui", 3000),
		}, testInfraWithDomain(ctx))
	}, pulumi.WithMocks("proj", "stack", mocks))
	require.NoError(t, err)

	routing := requireURLMapRouting(t, mocks.resources)
	require.Equal(t, 2, routing.hostRuleCount, "expected one host rule per ingress service")

	// api's names reach api; ui's names reach ui -- not the first backend.
	require.Equal(t, "api-backend_id", routing.backendFor(testAPIDelegateHostname))
	require.Equal(t, "api-backend_id", routing.backendFor("api--8080.proj.example.com"))
	require.Equal(t, "ui-backend_id", routing.backendFor("ui.proj.example.com"))
	require.Equal(t, "ui-backend_id", routing.backendFor("ui--3000.proj.example.com"))

	// The pre-fix behavior, spelled out so a regression is unmistakable.
	require.NotEqual(t, routing.defaultService, routing.backendFor("ui.proj.example.com"),
		"ui.proj.example.com fell through to DefaultService -- the #373 regression is back")

	// Same invariant from the other end: every name that resolves to the LB's IP
	// has a route. Both lists come from delegateHostnames, so they cannot drift.
	dnsNames := publicDNSNames(mocks.resources)
	require.Len(t, dnsNames, 4, "expected <service>.<domain> and <service>--<port>.<domain> for two services")
	for _, name := range dnsNames {
		require.Contains(t, routing.hostToMatcher, name,
			"DNS record %q resolves to the load balancer but no host rule routes it", name)
	}
}

// GCP rejects a URL map that lists one hostname in two host rules, with an error
// that doesn't name the services involved. Two services can collide on a
// hostname once delegate names are in the map -- here a BYOD `domainname` that
// happens to be another service's delegate name.
func TestCreateLoadBalancersRejectsTwoServicesClaimingOneHostname(t *testing.T) {
	mocks := &albMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		proxy := testCloudRunEntry(t, ctx, "proxy", 80)
		proxy.Config.DomainName = testAPIDelegateHostname // == api's delegate hostname
		return CreateLoadBalancers(ctx, "proj", []LBServiceEntry{
			testCloudRunEntry(t, ctx, "api", 8080),
			proxy,
		}, testInfraWithDomain(ctx))
	}, pulumi.WithMocks("proj", "stack", mocks))
	require.ErrorIs(t, err, errDuplicateRoute)
	require.Contains(t, err.Error(), `"api" and "proxy" both claim hostname "api.proj.example.com"`)
	requireNoExternalLBResources(t, mocks.resources)
}

// Two service names that differ only in characters pathMatcherName strips (here
// "_" vs "-") reduce to the same path matcher name. Duplicate path matcher names
// are rejected by GCP; say which services collided instead.
func TestCreateLoadBalancersRejectsTwoServicesReducingToOneRoute(t *testing.T) {
	mocks := &albMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return CreateLoadBalancers(ctx, "proj", []LBServiceEntry{
			testCloudRunEntry(t, ctx, "my_api", 8080),
			testCloudRunEntry(t, ctx, "my-api", 9090),
		}, testInfraWithDomain(ctx))
	}, pulumi.WithMocks("proj", "stack", mocks))
	require.ErrorIs(t, err, errDuplicateRoute)
	require.Contains(t, err.Error(), `both reduce to load balancer route "my-api"`)
	requireNoExternalLBResources(t, mocks.resources)
}

func TestCreateLoadBalancersRejectsNormalizedAliasClaimedByTwoServices(t *testing.T) {
	mocks := &albMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		api := testCloudRunEntry(t, ctx, "api", 8080)
		api.Config.DomainName = "api.example.com"
		api.Config.Networks = map[compose.NetworkID]compose.ServiceNetworkConfig{
			compose.DefaultNetwork: {Aliases: []string{"shared.example.com"}},
		}
		ui := testCloudRunEntry(t, ctx, "ui", 3000)
		ui.Config.DomainName = "ui.example.com"
		ui.Config.Networks = map[compose.NetworkID]compose.ServiceNetworkConfig{
			compose.DefaultNetwork: {Aliases: []string{"SHARED.EXAMPLE.COM."}},
		}
		return CreateLoadBalancers(ctx, "proj", []LBServiceEntry{api, ui}, testInfraWithDomain(ctx))
	}, pulumi.WithMocks("proj", "stack", mocks))
	require.ErrorIs(t, err, errDuplicateRoute)
	require.Contains(t, err.Error(), `"api" and "ui" both claim hostname "shared.example.com"`)
	requireNoExternalLBResources(t, mocks.resources)
}

func requireNoExternalLBResources(t *testing.T, resources []albResource) {
	t.Helper()
	for _, res := range resources {
		require.NotContains(t, []string{
			"gcp:dns/recordSet:RecordSet",
			"gcp:compute/regionNetworkEndpointGroup:RegionNetworkEndpointGroup",
			"gcp:compute/backendService:BackendService",
			"gcp:compute/uRLMap:URLMap",
		}, res.typeof, "external load balancer resource %s %q was registered before route validation", res.typeof, res.name)
	}
}

// A `domainname` set to the service's own delegate hostname is a plausible user
// config. It must not produce the same host twice in one host rule.
func TestCreateLoadBalancersDeduplicatesByodMatchingOwnDelegateHostname(t *testing.T) {
	mocks := &albMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		entry := testCloudRunEntry(t, ctx, "api", 80)
		entry.Config.DomainName = testAPIDelegateHostname
		return CreateLoadBalancers(ctx, "proj", []LBServiceEntry{entry}, testInfraWithDomain(ctx))
	}, pulumi.WithMocks("proj", "stack", mocks))
	require.NoError(t, err)

	urlMap := requireResource(t, mocks.resources, "gcp:compute/uRLMap:URLMap", "urlmap")
	hosts := urlMap.inputs["hostRules"].ArrayValue()[0].ObjectValue()["hosts"].ArrayValue()
	require.Len(t, hosts, 4, "expected two delegate names with bare and explicit-443 forms, no repeat")
	requireURLMapRouting(t, mocks.resources) // also asserts no duplicate across rules
}

// PR #499 gives a service's custom domain and its default-network aliases DNS
// and certificate treatment from common.ByodHostnames. All of those Host values
// must reach that same service rather than the URL-map default.
func TestCreateLoadBalancersRoutesCustomDomainAndDefaultNetworkAliases(t *testing.T) {
	mocks := &albMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		entry := testCloudRunEntry(t, ctx, "api", 8080)
		entry.Config.DomainName = "shop.example.com"
		entry.Config.Networks = map[compose.NetworkID]compose.ServiceNetworkConfig{
			compose.DefaultNetwork: {Aliases: []string{"admin.example.com", "*.preview.example.com"}},
		}
		return CreateLoadBalancers(ctx, "proj", []LBServiceEntry{
			testCloudRunEntry(t, ctx, "ui", 3000),
			entry,
		}, testInfraWithDomain(ctx))
	}, pulumi.WithMocks("proj", "stack", mocks))
	require.NoError(t, err)

	routing := requireURLMapRouting(t, mocks.resources)
	require.Equal(t, "ui-backend_id", routing.defaultService,
		"test fixture must not let api pass through the URL-map fallback")
	require.Equal(t, 2, routing.hostRuleCount, "custom names should extend the service's host rule, not add one")
	require.Equal(t, "api-backend_id", routing.backendFor("shop.example.com"))
	require.Equal(t, "api-backend_id", routing.backendFor("ADMIN.EXAMPLE.COM"))
	require.Equal(t, "api-backend_id", routing.backendFor("pr-42.preview.example.com"))
	require.Equal(t, "api-backend_id", routing.backendFor(testAPIDelegateHostname))
	require.Equal(t, "ui-backend_id", routing.backendFor("ui.proj.example.com"))
	// A global EXTERNAL_MANAGED URL map considers an explicit port, so the URL
	// map must list :443 variants instead of letting these requests fall back.
	require.Equal(t, "api-backend_id", routing.backendFor("shop.example.com:443"))
	require.Equal(t, "api-backend_id", routing.backendFor("admin.example.com:443"))
	require.Equal(t, "api-backend_id", routing.backendFor("pr-42.preview.example.com:443"))
	require.Equal(t, "api-backend_id", routing.backendFor(testAPIDelegateHostname+":443"))
	require.Equal(t, "api-backend_id", routing.backendFor("api--8080.proj.example.com:443"))
}

// The standalone Service path has no delegate domain (SharedInfra.Domain == "").
// A service with no BYOD name then contributes no hostnames, and must not produce
// a host rule with an empty Hosts array -- GCP rejects that.
func TestCreateLoadBalancersWithoutDelegateDomainEmitsOnlyByodHostRules(t *testing.T) {
	mocks := &albMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		byod := testCloudRunEntry(t, ctx, "api", 8080)
		byod.Config.DomainName = "shop.example.com"
		return CreateLoadBalancers(ctx, "proj", []LBServiceEntry{
			byod,
			testCloudRunEntry(t, ctx, "ui", 3000), // no domainname, no delegate domain
		}, testInfra(ctx)) // testInfra leaves Domain empty
	}, pulumi.WithMocks("proj", "stack", mocks))
	require.NoError(t, err)

	routing := requireURLMapRouting(t, mocks.resources)
	require.Equal(t, 1, routing.hostRuleCount, "only the BYOD service should get a host rule")
	require.Equal(t, "api-backend_id", routing.backendFor("shop.example.com"))
	require.Empty(t, publicDNSNames(mocks.resources), "no delegate domain means no public A records")
}

// A single-service project has nothing to mis-route, but it must not regress:
// the service still gets its host rule and DefaultService is still populated.
func TestCreateLoadBalancersSingleServiceKeepsDefaultServiceAndHostRule(t *testing.T) {
	mocks := &albMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return CreateLoadBalancers(ctx, "proj", []LBServiceEntry{
			testCloudRunEntry(t, ctx, "api", 8080),
		}, testInfraWithDomain(ctx))
	}, pulumi.WithMocks("proj", "stack", mocks))
	require.NoError(t, err)

	routing := requireURLMapRouting(t, mocks.resources)
	require.Equal(t, 1, routing.hostRuleCount)
	require.Equal(t, "api-backend_id", routing.defaultService,
		"an unmatched Host header still reaches the first ingress service")
	require.Equal(t, "api-backend_id", routing.backendFor(testAPIDelegateHostname))
}

// Compute Engine services route through buildMIGLBEntry, a separate code path
// that had the same BYOD-only host rule. It must get delegate host rules too.
func TestCreateLoadBalancersMIGGetsDelegateHostRules(t *testing.T) {
	mocks := &albMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		mig, err := testMIG(ctx, "worker")
		if err != nil {
			return err
		}
		config := testServiceConfig([]compose.ServicePortConfig{{Target: 3000, Mode: compose.PortModeIngress}})
		return CreateLoadBalancers(ctx, "proj", []LBServiceEntry{
			testCloudRunEntry(t, ctx, "api", 8080),
			{Name: "worker", InstanceGroup: mig, PrivateFqdn: "worker.google.internal", Config: config},
		}, testInfraWithDomain(ctx))
	}, pulumi.WithMocks("proj", "stack", mocks))
	require.NoError(t, err)

	routing := requireURLMapRouting(t, mocks.resources)
	require.Equal(t, 2, routing.hostRuleCount)
	require.Equal(t, "worker-3000-gce-backend_id", routing.backendFor("worker.proj.example.com"))
	require.Equal(t, "worker-3000-gce-backend_id", routing.backendFor("worker--3000.proj.example.com"))
	require.Equal(t, "api-backend_id", routing.backendFor(testAPIDelegateHostname))
}

// Cloud Run allows several ingress ports; each gets a "--<port>" DNS record, so
// each needs a route. They all reach the one Cloud Run backend -- Cloud Run
// serves a single port -- matching the legacy CD, which put every endpoint name
// in one host rule.
func TestCreateLoadBalancersCloudRunMultipleIngressPortsGetHostRules(t *testing.T) {
	mocks := &albMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		entry := testCloudRunEntry(t, ctx, "api", 8080)
		entry.Config.Ports = append(entry.Config.Ports, compose.ServicePortConfig{
			Target: 9090, Mode: compose.PortModeIngress,
		})
		// "ui" is listed first so it -- not api -- is the DefaultService. Without
		// that, api's names would resolve correctly through the fallback and this
		// test would pass even with no host rules at all.
		return CreateLoadBalancers(ctx, "proj", []LBServiceEntry{
			testCloudRunEntry(t, ctx, "ui", 3000),
			entry,
		}, testInfraWithDomain(ctx))
	}, pulumi.WithMocks("proj", "stack", mocks))
	require.NoError(t, err)

	routing := requireURLMapRouting(t, mocks.resources)
	require.Equal(t, "ui-backend_id", routing.defaultService)
	for _, host := range []string{
		testAPIDelegateHostname, "api--8080.proj.example.com", "api--9090.proj.example.com",
	} {
		require.Equal(t, "api-backend_id", routing.backendFor(host))
	}
}

// Path matcher names must match GCP's '[a-z]([-a-z0-9]*[a-z0-9])?' and the host
// rule has to reference the same sanitized name. The external path used the raw
// service name, so an underscore (legal in a Compose service name, and mapped to
// "-" in the hostname by common.ServiceLabel) produced an invalid, mismatched name.
func TestCreateLoadBalancersSanitizesPathMatcherName(t *testing.T) {
	mocks := &albMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return CreateLoadBalancers(ctx, "proj", []LBServiceEntry{
			testCloudRunEntry(t, ctx, "my_api", 8080),
		}, testInfraWithDomain(ctx))
	}, pulumi.WithMocks("proj", "stack", mocks))
	require.NoError(t, err)

	routing := requireURLMapRouting(t, mocks.resources)
	// The hostname uses ServiceLabel ("_" -> "-"); the Pulumi logical name keeps the
	// raw service name. requireURLMapRouting already checked the host rule resolves
	// to a matcher that exists -- which is what breaks when the two disagree.
	require.Equal(t, "my_api-backend_id", routing.backendFor("my-api.proj.example.com"))
	for name := range routing.matcherService {
		require.Regexp(t, `^[a-z]([-a-z0-9]{0,61}[a-z0-9])?$`, name)
	}
}

func testInfraWithDomain(ctx *pulumi.Context) *SharedInfra {
	infra := testInfra(ctx)
	infra.Domain = "proj.example.com"
	infra.PublicZoneId = pulumi.String("public-zone")
	return infra
}

func testCloudRunEntry(t *testing.T, ctx *pulumi.Context, name string, port int32) LBServiceEntry {
	t.Helper()
	service, err := cloudrunv2.NewService(ctx, name+"-cloudrun", &cloudrunv2.ServiceArgs{
		Location: pulumi.String("us-central1"),
		Template: &cloudrunv2.ServiceTemplateArgs{},
	})
	require.NoError(t, err)
	config := testServiceConfig([]compose.ServicePortConfig{{Target: port, Mode: compose.PortModeIngress}})
	return LBServiceEntry{Name: name, CloudRunService: service, Config: config}
}

func testInfra(ctx *pulumi.Context) *SharedInfra {
	publicIP, err := compute.NewGlobalAddress(ctx, "public-ip", &compute.GlobalAddressArgs{})
	if err != nil {
		panic(err)
	}
	return &SharedInfra{
		Region:         "us-central1",
		VpcId:          pulumi.String("vpc").ToStringOutput(),
		SubnetId:       pulumi.String("subnet").ToStringOutput(),
		PrivateZone:    pulumi.String("private-zone").ToStringOutput(),
		ProxySubnetId:  "proxy-subnet",
		PublicIP:       publicIP,
		WildcardCertId: pulumi.String("wildcard-cert"),
	}
}

func testMIG(ctx *pulumi.Context, name string) (*compute.RegionInstanceGroupManager, error) {
	return compute.NewRegionInstanceGroupManager(ctx, name+"-mig", &compute.RegionInstanceGroupManagerArgs{
		BaseInstanceName: pulumi.String(name),
		Region:           pulumi.String("us-central1"),
		TargetSize:       pulumi.Int(1),
		Versions: compute.RegionInstanceGroupManagerVersionArray{&compute.RegionInstanceGroupManagerVersionArgs{
			InstanceTemplate: pulumi.String("template"),
		}},
	})
}

func testServiceConfig(ports []compose.ServicePortConfig) compose.ServiceConfig {
	return compose.ServiceConfig{
		HealthCheck: &compose.HealthCheckConfig{Test: []string{"CMD", "curl", "http://localhost:3000/"}},
		Ports:       ports,
	}
}

func requirePropertyString(t *testing.T, props resource.PropertyMap, key, want string) {
	t.Helper()
	value, ok := props[resource.PropertyKey(key)]
	require.True(t, ok, "missing property %s", key)
	require.Equal(t, want, value.StringValue())
}

func requireResource(t *testing.T, resources []albResource, typ, name string) albResource {
	t.Helper()
	for _, r := range resources {
		if r.typeof == typ && r.name == name {
			return r
		}
	}
	require.Failf(t, "missing resource", "type=%s name=%s resources=%v", typ, name, resources)
	return albResource{}
}
