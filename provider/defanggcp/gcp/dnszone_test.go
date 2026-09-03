package gcp

import (
	"errors"
	"sync"
	"testing"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/dns"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/require"
)

var errInjectedZoneList = errors.New("injected zone-list failure")

func TestZoneAllowsByodRecords(t *testing.T) {
	for _, tt := range []struct {
		name        string
		description string
		want        bool
	}{
		{name: "marker only", description: byodZoneAuthorizationMarker, want: true},
		{
			name:        "marker word among prose",
			description: "Customer zone " + byodZoneAuthorizationMarker + " managed by platform",
			want:        true,
		},
		{name: "missing", description: "Customer zone", want: false},
		{name: "substring", description: "prefix-" + byodZoneAuthorizationMarker, want: false},
		{name: "wrong value", description: "defang.dev/byod-dns=denied", want: false},
		{name: "case differs", description: "Defang.dev/byod-dns=authorized", want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, zoneAllowsByodRecords(tt.description))
		})
	}
}

func TestBestZoneMatchRequiresAuthorizationAndUsesLongestTrustedSuffix(t *testing.T) {
	zoneName := func(value string) *string { return &value }
	zones := []dns.GetManagedZonesManagedZone{
		{
			Name: zoneName("parent-zone"), DnsName: "example.com.",
			Description: byodZoneAuthorizationMarker, Visibility: "public",
		},
		{
			Name: zoneName("staging-zone"), DnsName: "staging.example.com.",
			Description: "owned " + byodZoneAuthorizationMarker,
		},
		// This is the closest suffix, but its owner has not authorized Defang.
		{
			Name: zoneName("untrusted-private-app"), DnsName: "private.staging.example.com.",
			Description: "another team's zone",
		},
		{
			Name: zoneName("private-zone"), DnsName: "private.example.com.",
			Description: byodZoneAuthorizationMarker, Visibility: "private",
		},
		{Name: zoneName("other-zone"), DnsName: "other.net.", Description: byodZoneAuthorizationMarker},
		{Name: nil, DnsName: "nameless.example.com.", Description: byodZoneAuthorizationMarker},
		{Name: zoneName(""), DnsName: "empty-name.example.com.", Description: byodZoneAuthorizationMarker},
	}

	for _, tt := range []struct {
		name     string
		hostname string
		want     string
	}{
		{name: "zone apex", hostname: "example.com", want: "parent-zone"},
		{name: "subdomain", hostname: "api.example.com", want: "parent-zone"},
		{name: "longest trusted suffix", hostname: "api.staging.example.com", want: "staging-zone"},
		{name: "untrusted longest suffix is ignored", hostname: "api.private.staging.example.com", want: "staging-zone"},
		{name: "private zone is ignored", hostname: "api.private.example.com", want: "parent-zone"},
		{name: "wildcard", hostname: "*.staging.example.com", want: "staging-zone"},
		{name: "normalizes case and trailing dot", hostname: "API.STAGING.EXAMPLE.COM.", want: "staging-zone"},
		{name: "suffix is not a parent", hostname: "notexample.com", want: ""},
		{name: "zone embedded in another domain", hostname: "example.com.evil.net", want: ""},
		{name: "empty hostname", hostname: "", want: ""},
		{name: "bare wildcard", hostname: "*.", want: ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bestZoneMatch(tt.hostname, zones)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestBestZoneMatchRejectsAmbiguousEquivalentZones(t *testing.T) {
	z := func(name string) dns.GetManagedZonesManagedZone {
		return dns.GetManagedZonesManagedZone{
			Name: &name, DnsName: "example.com.", Description: byodZoneAuthorizationMarker,
		}
	}
	_, err := bestZoneMatch("api.example.com", []dns.GetManagedZonesManagedZone{
		z("z-zone"), z("a-zone"), z("m-zone"), z("a-zone"),
	})
	require.ErrorContains(t, err, `multiple authorized public Cloud DNS managed zones match hostname "api.example.com" at DNS suffix "example.com"`)
	require.ErrorContains(t, err, "a-zone, m-zone, z-zone")
	require.ErrorContains(t, err, `keep "defang.dev/byod-dns=authorized" in the description of only the publicly delegated zone`)
}

type findZonesMocks struct {
	mu        sync.Mutex
	calls     int
	fail      bool
	callArgs  resource.PropertyMap
	zoneProps []resource.PropertyValue
}

func (m *findZonesMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (m *findZonesMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	m.callArgs = args.Args
	if m.fail {
		return nil, errInjectedZoneList
	}
	return resource.PropertyMap{
		"managedZones": resource.NewArrayProperty(m.zoneProps),
	}, nil
}

func (m *findZonesMocks) snapshot() (int, resource.PropertyMap) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls, m.callArgs.Copy()
}

func managedZoneProperty(name, dnsName, description, visibility string) resource.PropertyValue {
	return resource.NewObjectProperty(resource.PropertyMap{
		"name":        resource.NewStringProperty(name),
		"dnsName":     resource.NewStringProperty(dnsName),
		"description": resource.NewStringProperty(description),
		"visibility":  resource.NewStringProperty(visibility),
	})
}

func TestFindZonesListsOnceAndNormalizesDuplicateHostnames(t *testing.T) {
	mocks := &findZonesMocks{zoneProps: []resource.PropertyValue{
		managedZoneProperty("trusted-zone", "example.com.", byodZoneAuthorizationMarker, "public"),
	}}
	var got map[string]string
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		var err error
		got, err = FindZones(ctx, "dns-project", []string{
			"API.Example.COM.", "api.example.com", "elsewhere.net",
		})
		return err
	}, pulumi.WithMocks("project", "stack", mocks))
	require.NoError(t, err)
	require.Equal(t, map[string]string{"api.example.com": "trusted-zone"}, got)
	calls, args := mocks.snapshot()
	require.Equal(t, 1, calls)
	require.Equal(t, "dns-project", args["project"].StringValue())
}

func TestFindZonesSkipsInvokeForNoHostnames(t *testing.T) {
	mocks := &findZonesMocks{}
	var got map[string]string
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		var err error
		got, err = FindZones(ctx, "dns-project", nil)
		return err
	}, pulumi.WithMocks("project", "stack", mocks))
	require.NoError(t, err)
	require.Empty(t, got)
	calls, _ := mocks.snapshot()
	require.Zero(t, calls)
}

func TestFindZonesReturnsListFailure(t *testing.T) {
	mocks := &findZonesMocks{fail: true}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := FindZones(ctx, "dns-project", []string{"api.example.com"})
		return err
	}, pulumi.WithMocks("project", "stack", mocks))
	require.ErrorContains(t, err, "listing Cloud DNS managed zones")
	require.ErrorContains(t, err, "injected zone-list failure")
}
