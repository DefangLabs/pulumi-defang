package azure

import (
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
)

func TestBestZoneMatch(t *testing.T) {
	zone := func(name string) *armdns.Zone {
		return &armdns.Zone{
			Name: to.Ptr(name),
			ID:   to.Ptr("/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/dnszones/" + name),
		}
	}
	zones := []*armdns.Zone{
		zone("example.com"),
		zone("staging.example.com"),
		zone("other.com"),
		nil,
		{Name: to.Ptr("no-id.com")}, // skipped: no ID
		{ID: to.Ptr("/subscriptions/s/.../dnszones/no-name.com")}, // skipped: no name
	}
	tests := []struct {
		name   string
		domain string
		want   string
	}{
		{name: "apex", domain: "example.com", want: "example.com"},
		{name: "subdomain", domain: "api.example.com", want: "example.com"},
		{name: "longest suffix wins", domain: "api.staging.example.com", want: "staging.example.com"},
		{name: "case insensitive", domain: "API.Example.COM", want: "example.com"},
		{name: "trailing dot", domain: "api.example.com.", want: "example.com"},
		{name: "no match", domain: "elsewhere.net", want: ""},
		{name: "suffix but not a DNS parent", domain: "notexample.com", want: ""},
		{name: "zone name embedded in another domain", domain: "example.com.evil.com", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotID := bestZoneMatch(tt.domain, zones)
			if gotName != tt.want {
				t.Errorf("bestZoneMatch(%q) name = %q, want %q", tt.domain, gotName, tt.want)
			}
			if tt.want == "" {
				if gotID != "" {
					t.Errorf("bestZoneMatch(%q) id = %q, want empty", tt.domain, gotID)
				}
			} else if !strings.HasSuffix(gotID, "/"+tt.want) {
				t.Errorf("bestZoneMatch(%q) id = %q, want the id of zone %q", tt.domain, gotID, tt.want)
			}
		})
	}
}
