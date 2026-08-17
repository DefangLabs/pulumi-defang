package azure

import (
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/cdn/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/dns/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testZone     = "example.com"
	testHost     = "auth." + testZone
	testWildcard = "*." + testHost
	testService  = "auth"
)

func ingressPorts() []compose.ServicePortConfig {
	return []compose.ServicePortConfig{{Target: 80, Mode: "ingress"}}
}

func aliasNetwork(aliases ...string) map[compose.NetworkID]compose.ServiceNetworkConfig {
	return map[compose.NetworkID]compose.ServiceNetworkConfig{
		compose.DefaultNetwork: {Aliases: aliases},
	}
}

// TestWildcardHostnames pins down which compose shapes count as a wildcard
// request. The two that matter in practice are a wildcard written straight into
// `domainname` and one added as a network alias next to a plain `domainname` —
// the second is how a service keeps a canonical hostname and still answers for
// every subdomain.
func TestWildcardHostnames(t *testing.T) {
	tests := []struct {
		name string
		svc  compose.ServiceConfig
		want []string
	}{
		{
			name: "no domainname",
			svc:  compose.ServiceConfig{Ports: ingressPorts()},
			want: nil,
		},
		{
			name: "plain domainname only",
			svc:  compose.ServiceConfig{Ports: ingressPorts(), DomainName: testHost},
			want: nil,
		},
		{
			name: "wildcard domainname",
			svc:  compose.ServiceConfig{Ports: ingressPorts(), DomainName: testWildcard},
			want: []string{testWildcard},
		},
		{
			name: "wildcard alias alongside plain domainname",
			svc: compose.ServiceConfig{
				Ports:      ingressPorts(),
				DomainName: testHost,
				Networks:   aliasNetwork(testWildcard),
			},
			want: []string{testWildcard},
		},
		{
			name: "several wildcards keep compose order",
			svc: compose.ServiceConfig{
				Ports:      ingressPorts(),
				DomainName: "*.a.example.com",
				Networks:   aliasNetwork("plain.example.com", "*.b.example.com"),
			},
			want: []string{"*.a.example.com", "*.b.example.com"},
		},
		{
			name: "hostnames are normalized",
			svc:  compose.ServiceConfig{Ports: ingressPorts(), DomainName: "*.Auth.Example.COM."},
			want: []string{testWildcard},
		},
		{
			// Front Door needs a public origin to forward to; an internal-only
			// service has no reachable hostname to offer it.
			name: "no ingress ports",
			svc:  compose.ServiceConfig{DomainName: testWildcard},
			want: nil,
		},
		{
			// An alias without a domainname isn't a BYOD request at all, matching
			// the AWS path.
			name: "wildcard alias without domainname",
			svc:  compose.ServiceConfig{Ports: ingressPorts(), Networks: aliasNetwork(testWildcard)},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := wildcardHostnames(tt.svc); !slices.Equal(got, tt.want) {
				t.Errorf("wildcardHostnames() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHasWildcardHostname covers the project-level gate. Getting a false
// positive here means provisioning a Front Door profile — a real monthly bill —
// for a project that has no use for one.
func TestHasWildcardHostname(t *testing.T) {
	tests := []struct {
		name     string
		services compose.Services
		want     bool
	}{
		{name: "no services", services: compose.Services{}, want: false},
		{
			name: "plain domains only",
			services: compose.Services{
				testService: {Ports: ingressPorts(), DomainName: testHost},
				"web":       {Ports: ingressPorts(), DomainName: testZone},
			},
			want: false,
		},
		{
			name: "one service out of several wants a wildcard",
			services: compose.Services{
				testService: {Ports: ingressPorts(), DomainName: testHost, Networks: aliasNetwork(testWildcard)},
				"web":       {Ports: ingressPorts(), DomainName: testZone},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasWildcardHostname(tt.services); got != tt.want {
				t.Errorf("HasWildcardHostname() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRelativeRecordName covers the decision of whether the provider can write a
// wildcard's DNS records itself. A false "in zone" would make Pulumi write into
// a zone that doesn't hold the name; a false "not in zone" only downgrades to
// printing the records, so the two failures are not symmetric.
func TestRelativeRecordName(t *testing.T) {
	tests := []struct {
		name, fqdn, zone string
		wantName         string
		wantIn           bool
	}{
		{name: "subdomain", fqdn: testHost, zone: testZone, wantName: testService, wantIn: true},
		{name: "deep subdomain", fqdn: "a.b.example.com", zone: testZone, wantName: "a.b", wantIn: true},
		{name: "apex", fqdn: testZone, zone: testZone, wantName: "@", wantIn: true},
		{name: "case and trailing dot", fqdn: "Auth.Example.com.", zone: testZone, wantName: testService, wantIn: true},
		{name: "different zone", fqdn: "auth.other.com", zone: testZone, wantIn: false},
		{name: "empty zone", fqdn: testHost, zone: "", wantIn: false},
		{name: "fqdn is a parent of the zone", fqdn: testZone, zone: testHost, wantIn: false},
		{
			// "notexample.com" ends with the zone as a string but is a
			// different domain — matching on the label boundary is the point.
			name: "suffix match that isn't a label boundary",
			fqdn: "auth.notexample.com", zone: testZone, wantIn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotIn := relativeRecordName(tt.fqdn, tt.zone)
			if gotIn != tt.wantIn {
				t.Fatalf("relativeRecordName(%q, %q) in-zone = %v, want %v", tt.fqdn, tt.zone, gotIn, tt.wantIn)
			}
			if gotIn && gotName != tt.wantName {
				t.Errorf("relativeRecordName(%q, %q) = %q, want %q", tt.fqdn, tt.zone, gotName, tt.wantName)
			}
		})
	}
}

func TestJoinLabels(t *testing.T) {
	tests := []struct{ prefix, relative, want string }{
		{dnsAuthPrefix, testService, dnsAuthPrefix + "." + testService},
		{dnsAuthPrefix, "@", dnsAuthPrefix},
		{"*", testService, "*." + testService},
		{"*", "@", "*"},
	}
	for _, tt := range tests {
		if got := joinLabels(tt.prefix, tt.relative); got != tt.want {
			t.Errorf("joinLabels(%q, %q) = %q, want %q", tt.prefix, tt.relative, got, tt.want)
		}
	}
}

// TestFrontDoorShortCircuits checks the branches that must create nothing, so
// EnsureFrontDoor and CreateWildcardDomain stay callable unconditionally from
// the project dispatcher and from CreateContainerApp.
func TestFrontDoorShortCircuits(t *testing.T) {
	wildcardSvc := compose.ServiceConfig{Ports: ingressPorts(), DomainName: testWildcard}
	plainSvc := compose.ServiceConfig{Ports: ingressPorts(), DomainName: testHost}

	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		fd, err := EnsureFrontDoor(ctx, compose.Services{testService: plainSvc}, &SharedInfra{}, nil)
		if err != nil {
			t.Errorf("EnsureFrontDoor(no wildcards) err: %v", err)
		}
		if fd != nil {
			t.Errorf("EnsureFrontDoor(no wildcards) = %+v, want nil", fd)
		}

		if fd, err := EnsureFrontDoor(ctx, compose.Services{testService: wildcardSvc}, nil, nil); err != nil || fd != nil {
			t.Errorf("EnsureFrontDoor(nil infra) = %+v, %v; want nil, nil", fd, err)
		}

		// A wildcard with no Front Door to serve it is the standalone Service
		// path. Failing beats deploying a service configured for a hostname that
		// nothing will answer on.
		_, err = CreateWildcardDomain(ctx, testService, wildcardSvc, nil, &SharedInfra{}, nil)
		if !errors.Is(err, errWildcardNeedsProject) {
			t.Errorf("CreateWildcardDomain(no front door) err = %v, want %v", err, errWildcardNeedsProject)
		}

		// No wildcard hostname: nothing to do even with a Front Door present.
		got, err := CreateWildcardDomain(ctx, testService, plainSvc, nil, &SharedInfra{FrontDoor: &FrontDoorInfra{}}, nil)
		if err != nil {
			t.Errorf("CreateWildcardDomain(no wildcard) err: %v", err)
		}
		if got != nil {
			t.Errorf("CreateWildcardDomain(no wildcard) = %+v, want nil", got)
		}
		return nil
	}, pulumi.WithMocks("project", "stack", &azureNoopMocks{}))
	if err != nil {
		t.Fatalf("pulumi.RunErr: %v", err)
	}
}

// TestWildcardZoneTarget covers which zone a wildcard's two records land in.
// The delegate-domain zone wins when it holds the name — the provider owns it, so
// the records are torn down with the project. Otherwise the BYOD zone the CD task
// resolved is used, which is what makes a wildcard live in a zone Defang didn't
// create (e.g. an Azure zone delegated from Route 53). With neither, the caller
// falls back to printing the records for the operator.
func TestWildcardZoneTarget(t *testing.T) {
	const byodZoneID = "/subscriptions/s/resourceGroups/byod-rg/providers/Microsoft.Network/dnszones/" + testHost

	tests := []struct {
		name         string
		base         string
		delegate     string
		withZone     bool // delegate zone imported into state
		dnsZoneID    string
		wantLabel    string
		wantByodZone string // set when the resolved zone is the literal BYOD one
		wantResolved bool
	}{
		{
			name: "delegate zone holds the name",
			base: testHost, delegate: testZone, withZone: true,
			wantLabel: testService, wantResolved: true,
		},
		{
			name: "byod zone at its apex",
			base: testHost, dnsZoneID: byodZoneID,
			wantLabel: "@", wantByodZone: testHost, wantResolved: true,
		},
		{
			// The delegate domain is configured but its zone was never imported
			// (the standalone path), so records can't be ordered against it.
			name: "delegate domain without an imported zone falls through to byod",
			base: testHost, delegate: testZone, dnsZoneID: byodZoneID,
			wantLabel: "@", wantByodZone: testHost, wantResolved: true,
		},
		{
			name: "delegate zone preferred over byod",
			base: testHost, delegate: testZone, withZone: true, dnsZoneID: byodZoneID,
			wantLabel: testService, wantResolved: true,
		},
		{
			// A wildcard alias pointing outside the zone resolved for the service's
			// own domainname: nothing reachable here can host it.
			name: "byod zone can't host the name",
			base: "auth.other.com", dnsZoneID: byodZoneID,
		},
		{name: "no zone at all", base: testHost},
		{name: "unparseable byod zone id", base: testHost, dnsZoneID: "garbage"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				infra := &SharedInfra{Domain: tt.delegate}
				if tt.withZone {
					rg, err := resources.NewResourceGroup(ctx, "rg", &resources.ResourceGroupArgs{})
					require.NoError(t, err)
					zone, err := dns.NewZone(ctx, "zone", &dns.ZoneArgs{
						ResourceGroupName: rg.Name,
						ZoneName:          pulumi.String(tt.delegate),
					})
					require.NoError(t, err)
					infra.ResourceGroup, infra.DomainZone = rg, zone
				}

				_, zoneName, label, ok := wildcardZoneTarget(tt.base, infra, tt.dnsZoneID)
				require.Equal(t, tt.wantResolved, ok, "resolved")
				if !ok {
					return nil
				}
				assert.Equal(t, tt.wantLabel, label)
				if tt.wantByodZone != "" {
					// The BYOD zone is a plain string parsed out of the ARM id.
					assert.Equal(t, pulumi.String(tt.wantByodZone), zoneName)
				} else {
					// The delegate zone is referenced through the imported
					// resource, so records order against it rather than racing it.
					assert.NotEqual(t, pulumi.String(tt.delegate), zoneName,
						"delegate zone must be referenced by resource, not by literal name")
				}
				return nil
			}, pulumi.WithMocks("project", "stack", azureNoopMocks{}))
			require.NoError(t, err)
		})
	}
}

// TestWildcardZoneTargetNilInfra pins the standalone-caller guard: no infra means
// no zone to write into, and certainly no panic.
func TestWildcardZoneTargetNilInfra(t *testing.T) {
	rgName, zoneName, label, ok := wildcardZoneTarget(testHost, nil, "")
	assert.False(t, ok, "nil infra must resolve no zone")
	assert.Nil(t, rgName)
	assert.Nil(t, zoneName)
	assert.Empty(t, label)
}

// TestPublishValidationByodZone is the end-to-end shape of the portal case: the
// wildcard's zone is one the CD task found (not the delegate domain), so both
// records must be written into it — the `_dnsauth` TXT and the `*` CNAME, named
// relative to that zone. Before this, they were only ever logged.
func TestPublishValidationByodZone(t *testing.T) {
	const byodZoneID = "/subscriptions/s/resourceGroups/byod-rg/providers/Microsoft.Network/dnszones/" + testZone

	m := &recordZoneMocks{byRelative: map[string]recordZone{}}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		domain, err := cdn.NewAFDCustomDomain(ctx, "domain", &cdn.AFDCustomDomainArgs{
			ResourceGroupName: pulumi.String("rg"),
			ProfileName:       pulumi.String("profile"),
			HostName:          pulumi.String(testWildcard),
		})
		require.NoError(t, err)
		endpoint, err := cdn.NewAFDEndpoint(ctx, "endpoint", &cdn.AFDEndpointArgs{
			ResourceGroupName: pulumi.String("rg"),
			ProfileName:       pulumi.String("profile"),
		})
		require.NoError(t, err)

		records, err := publishValidation(ctx, testService, testWildcard, domain,
			&FrontDoorInfra{Endpoint: endpoint}, &SharedInfra{}, byodZoneID)
		require.NoError(t, err)
		require.Len(t, records, 2, "both records must be written into the resolved zone")
		return nil
	}, pulumi.WithMocks("project", "stack", m))
	require.NoError(t, err)

	assert.Equal(t, map[string]recordZone{
		dnsAuthPrefix + "." + testService: {recordType: "TXT", zone: testZone, resourceGroup: "byod-rg"},
		"*." + testService:                {recordType: "CNAME", zone: testZone, resourceGroup: "byod-rg"},
	}, m.byRelative)
}

// TestPublishValidationNoZone keeps the warn-and-degrade path: with no zone this
// deployment can write to, the records are logged and none are created.
func TestPublishValidationNoZone(t *testing.T) {
	m := &recordZoneMocks{byRelative: map[string]recordZone{}}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		domain, err := cdn.NewAFDCustomDomain(ctx, "domain", &cdn.AFDCustomDomainArgs{
			ResourceGroupName: pulumi.String("rg"),
			ProfileName:       pulumi.String("profile"),
			HostName:          pulumi.String(testWildcard),
		})
		require.NoError(t, err)
		endpoint, err := cdn.NewAFDEndpoint(ctx, "endpoint", &cdn.AFDEndpointArgs{
			ResourceGroupName: pulumi.String("rg"),
			ProfileName:       pulumi.String("profile"),
		})
		require.NoError(t, err)

		records, err := publishValidation(ctx, testService, testWildcard, domain,
			&FrontDoorInfra{Endpoint: endpoint}, &SharedInfra{}, "")
		require.NoError(t, err)
		assert.Empty(t, records)
		return nil
	}, pulumi.WithMocks("project", "stack", m))
	require.NoError(t, err)
	assert.Empty(t, m.byRelative, "no zone to write to means no records")
}

// recordZone is what a created DNS record set is asserted on: its type and the
// zone (and resource group) it was written into.
type recordZone struct {
	recordType    string
	zone          string
	resourceGroup string
}

// recordZoneMocks captures every dns RecordSet by its relative name.
// Pulumi registers resources from concurrent goroutines, so the map needs a lock.
type recordZoneMocks struct {
	mu         sync.Mutex
	byRelative map[string]recordZone
}

func (m *recordZoneMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if strings.HasSuffix(args.TypeToken, ":RecordSet") {
		m.mu.Lock()
		m.byRelative[args.Inputs["relativeRecordSetName"].StringValue()] = recordZone{
			recordType:    args.Inputs["recordType"].StringValue(),
			zone:          args.Inputs["zoneName"].StringValue(),
			resourceGroup: args.Inputs["resourceGroupName"].StringValue(),
		}
		m.mu.Unlock()
	}
	return args.Name + "_id", args.Inputs, nil
}

func (recordZoneMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}
