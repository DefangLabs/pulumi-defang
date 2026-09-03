package program

import (
	"errors"
	"sync"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/require"
)

const gcpByodZoneAuthorizationMarker = "defang.dev/byod-dns=authorized"

var errInjectedGcpByodZoneLookup = errors.New("injected managed-zone listing failure")

type gcpByodZoneMocks struct {
	mu       sync.Mutex
	calls    int
	failCall bool
	zones    []resource.PropertyValue
}

func (m *gcpByodZoneMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (m *gcpByodZoneMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.failCall {
		return nil, errInjectedGcpByodZoneLookup
	}
	return resource.PropertyMap{"managedZones": resource.NewArrayProperty(m.zones)}, nil
}

func (m *gcpByodZoneMocks) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func gcpManagedZone(name, dnsName, description string) resource.PropertyValue {
	return resource.NewObjectProperty(resource.PropertyMap{
		"name":        resource.NewStringProperty(name),
		"dnsName":     resource.NewStringProperty(dnsName),
		"description": resource.NewStringProperty(description),
		"visibility":  resource.NewStringProperty("public"),
	})
}

func gcpByodService(domain string, ingress bool, aliases ...string) compose.ServiceConfig {
	svc := compose.ServiceConfig{
		DomainName: domain,
		Networks: map[compose.NetworkID]compose.ServiceNetworkConfig{
			compose.DefaultNetwork: {Aliases: aliases},
		},
	}
	if ingress {
		svc.Ports = []compose.ServicePortConfig{{Target: 443, Mode: compose.PortModeIngress}}
	}
	return svc
}

func TestFindByodZonesGCPUsesFullIngressHostnameSetAndTrustedZones(t *testing.T) {
	mocks := &gcpByodZoneMocks{zones: []resource.PropertyValue{
		gcpManagedZone("trusted-parent", "example.com.", "Customer-owned "+gcpByodZoneAuthorizationMarker),
		gcpManagedZone("untrusted-child", "apps.example.com.", "Another team's zone"),
	}}
	var got map[string]string
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		provider, err := gcp.NewProvider(ctx, "gcp", &gcp.ProviderArgs{Project: pulumi.String("dns-project")})
		if err != nil {
			return err
		}
		got = findByodZonesGCP(ctx, &compose.Project{Services: compose.Services{
			"api":      gcpByodService("API.Apps.Example.COM.", true, "*.apps.example.com", "login.example.com"),
			"worker":   gcpByodService("worker.example.com", false, "worker-alias.example.com"),
			"noDomain": gcpByodService("", true, "ignored.example.com"),
		}}, provider)
		return nil
	}, pulumi.WithMocks("project", "stack", mocks))
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"api.apps.example.com": "trusted-parent",
		"*.apps.example.com":   "trusted-parent",
		"login.example.com":    "trusted-parent",
	}, got)
	require.Equal(t, 1, mocks.Calls())
}

func TestFindByodZonesGCPFallsBackWhenListingFails(t *testing.T) {
	mocks := &gcpByodZoneMocks{failCall: true}
	var got map[string]string
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		provider, err := gcp.NewProvider(ctx, "gcp", &gcp.ProviderArgs{})
		if err != nil {
			return err
		}
		got = findByodZonesGCP(ctx, &compose.Project{Services: compose.Services{
			"api": gcpByodService("api.example.com", true),
		}}, provider)
		return nil
	}, pulumi.WithMocks("project", "stack", mocks))
	require.NoError(t, err)
	require.Nil(t, got)
	require.Equal(t, 1, mocks.Calls())
}

func TestByodZoneLookupFailureWarningDescribesBothFallbacks(t *testing.T) {
	warning := byodZoneLookupFailureWarning(errInjectedGcpByodZoneLookup)
	require.Contains(t, warning, "injected managed-zone listing failure")
	require.Contains(t, warning, "exact hostnames fall back to load balancer authorized certificates")
	require.Contains(t, warning, "wildcard certificates are skipped because they require DNS authorization")
}

func TestNoTrustedByodZoneWarningDistinguishesWildcard(t *testing.T) {
	exact := noTrustedByodZoneWarning("API.Example.COM.")
	require.Contains(t, exact, "load balancer authorization")
	require.NotContains(t, exact, "will be skipped")

	wildcard := noTrustedByodZoneWarning("*.Example.COM.")
	require.Contains(t, wildcard, "wildcard certificate will be skipped")
	require.Contains(t, wildcard, "requires DNS authorization")
	require.NotContains(t, wildcard, "load balancer authorization")
}

func TestFindByodZonesGCPSkipsListingWithoutIngressByodHostnames(t *testing.T) {
	mocks := &gcpByodZoneMocks{}
	var got map[string]string
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		provider, err := gcp.NewProvider(ctx, "gcp", &gcp.ProviderArgs{})
		if err != nil {
			return err
		}
		got = findByodZonesGCP(ctx, &compose.Project{Services: compose.Services{
			"worker": gcpByodService("worker.example.com", false),
		}}, provider)
		return nil
	}, pulumi.WithMocks("project", "stack", mocks))
	require.NoError(t, err)
	require.Nil(t, got)
	require.Zero(t, mocks.Calls())
}
