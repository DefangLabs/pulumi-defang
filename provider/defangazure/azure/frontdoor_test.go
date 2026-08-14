package azure

import (
	"errors"
	"slices"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
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
		_, err = CreateWildcardDomain(ctx, testService, wildcardSvc, nil, &SharedInfra{})
		if !errors.Is(err, errWildcardNeedsProject) {
			t.Errorf("CreateWildcardDomain(no front door) err = %v, want %v", err, errWildcardNeedsProject)
		}

		// No wildcard hostname: nothing to do even with a Front Door present.
		got, err := CreateWildcardDomain(ctx, testService, plainSvc, nil, &SharedInfra{FrontDoor: &FrontDoorInfra{}})
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
