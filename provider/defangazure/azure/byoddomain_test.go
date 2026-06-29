package azure

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func TestParseDNSZoneID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantRG  string
		wantZon string
		wantOK  bool
	}{
		{
			name:    "standard",
			id:      "/subscriptions/abc/resourceGroups/my-rg/providers/Microsoft.Network/dnszones/example.com",
			wantRG:  "my-rg",
			wantZon: "example.com",
			wantOK:  true,
		},
		{
			name:    "mixed-case segments",
			id:      "/subscriptions/abc/resourcegroups/RG1/providers/microsoft.network/dnsZones/foo.io",
			wantRG:  "RG1",
			wantZon: "foo.io",
			wantOK:  true,
		},
		{name: "empty", id: "", wantOK: false},
		{name: "no resource group", id: "/subscriptions/abc/providers/Microsoft.Network/dnszones/example.com", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rg, zone, ok := parseDNSZoneID(tt.id)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && (rg != tt.wantRG || zone != tt.wantZon) {
				t.Errorf("got rg=%q zone=%q, want rg=%q zone=%q", rg, zone, tt.wantRG, tt.wantZon)
			}
		})
	}
}

func TestRelativeRecordName(t *testing.T) {
	tests := []struct {
		domain, zone, want string
		wantOK             bool
	}{
		{"api.example.com", "example.com", "api", true},
		{"a.b.example.com", "example.com", "a.b", true},
		{"API.Example.com", "example.com", "API", true}, // case-insensitive suffix, original case preserved
		{"example.com", "example.com", "", false},       // apex unsupported
		{"other.com", "example.com", "", false},         // not in zone
	}
	for _, tt := range tests {
		got, ok := relativeRecordName(tt.domain, tt.zone)
		if ok != tt.wantOK || got != tt.want {
			t.Errorf("relativeRecordName(%q,%q) = (%q,%v), want (%q,%v)", tt.domain, tt.zone, got, ok, tt.want, tt.wantOK)
		}
	}
}

// TestCreateByodDomainShortCircuits checks the no-op branches that don't touch
// Pulumi: no custom domain, no zone id, no ingress, and an apex domain (CNAME
// at apex is unsupported).
func TestCreateByodDomainShortCircuits(t *testing.T) {
	ingress := []compose.ServicePortConfig{{Target: 80, Mode: "ingress"}}
	const zoneID = "/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/dnszones/example.com"

	tests := []struct {
		name   string
		svc    compose.ServiceConfig
		zoneID string
	}{
		{name: "no domain", svc: compose.ServiceConfig{Ports: ingress}, zoneID: zoneID},
		{name: "no zone id", svc: compose.ServiceConfig{DomainName: "api.example.com", Ports: ingress}, zoneID: ""},
		{name: "no ingress", svc: compose.ServiceConfig{DomainName: "api.example.com"}, zoneID: zoneID},
		{name: "apex domain", svc: compose.ServiceConfig{DomainName: "example.com", Ports: ingress}, zoneID: zoneID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				got, err := CreateByodDomain(ctx, "svc", tt.svc, nil, nil, tt.zoneID)
				if err != nil {
					t.Errorf("CreateByodDomain err: %v", err)
				}
				if got != nil {
					t.Errorf("CreateByodDomain result = %+v, want nil", got)
				}
				return nil
			}, pulumi.WithMocks("project", "stack", azureNoopMocks{}))
			if err != nil {
				t.Fatalf("pulumi.RunErr: %v", err)
			}
		})
	}
}
