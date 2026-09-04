package gcp

import (
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

	backend := requireResource(t, mocks.resources, "gcp:compute/backendService:BackendService", "api-public")
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

	backend := requireResource(t, mocks.resources, "gcp:compute/backendService:BackendService", "api-public")
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
		DomainName:  "api.example.com",
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

// Two private Cloud Run services used to claim one URN: the internal backend
// service was registered under the constant "private-lb-cloudrun-backend", so
// the second registration was a duplicate and the deploy failed outright.
func TestCreateLoadBalancersTwoPrivateCloudRunServices(t *testing.T) {
	mocks := &albMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		entries := make([]LBServiceEntry, 0, 2)
		for _, name := range []string{"api", "worker"} {
			svc, err := cloudrunv2.NewService(ctx, name, &cloudrunv2.ServiceArgs{
				Location: pulumi.String("us-central1"),
				Template: &cloudrunv2.ServiceTemplateArgs{},
			})
			if err != nil {
				return err
			}
			entries = append(entries, LBServiceEntry{
				Name:            name,
				CloudRunService: svc,
				PrivateFqdn:     name + ".google.internal",
				Config: compose.ServiceConfig{
					Ports: []compose.ServicePortConfig{{Target: 3000, Mode: compose.PortModeIngress}},
				},
			})
		}
		return CreateLoadBalancers(ctx, "proj", entries, testInfra(ctx))
	}, pulumi.WithMocks("proj", "stack", mocks))
	require.NoError(t, err)

	const backendType = "gcp:compute/regionBackendService:RegionBackendService"
	requireResource(t, mocks.resources, backendType, "api")
	requireResource(t, mocks.resources, backendType, "worker")
}
