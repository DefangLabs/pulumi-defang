package gcp

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/require"
)

type albResource struct {
	name   string
	typeof string
	inputs resource.PropertyMap
}

type albMocks struct {
	resources []albResource
}

func (m *albMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	m.resources = append(m.resources, albResource{name: args.Name, typeof: args.TypeToken, inputs: args.Inputs})
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
