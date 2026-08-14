package program

import (
	"sort"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
)

func TestCollectCertJobs(t *testing.T) {
	const (
		zone  = "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/dnszones/example.com"
		web   = "web"
		byod  = "api.example.com"
		shard = "shard.defang.dev"
	)

	ingress := []compose.ServicePortConfig{{Target: 8080, Mode: compose.PortModeIngress}}
	host := []compose.ServicePortConfig{{Target: 8080, Mode: compose.PortModeHost}}

	tests := []struct {
		name     string
		services compose.Services
		domain   string
		dnsZones map[string]string
		want     []certJob
	}{
		{
			name:     "delegate domain only",
			services: compose.Services{web: {Ports: ingress}, "worker": {Ports: host}},
			domain:   shard,
			want:     []certJob{{service: web, hostname: "web." + shard}},
		},
		{
			name:     "BYOD subdomain adds a second job for the same service",
			services: compose.Services{web: {Ports: ingress, DomainName: byod}},
			domain:   shard,
			dnsZones: map[string]string{byod: zone},
			want: []certJob{
				{service: web, hostname: "web." + shard},
				{service: web, hostname: byod},
			},
		},
		{
			name:     "BYOD apex is eligible",
			services: compose.Services{web: {Ports: ingress, DomainName: "example.com"}},
			dnsZones: map[string]string{"example.com": zone},
			want:     []certJob{{service: web, hostname: "example.com"}},
		},
		{
			// The provider won't create records for a domain the zone can't host, so
			// a cert job would sit in aca.IssueCert's DNS wait until it times out.
			name:     "domain the resolved zone cannot host is skipped",
			services: compose.Services{web: {Ports: ingress, DomainName: "api.elsewhere.net"}},
			dnsZones: map[string]string{"api.elsewhere.net": zone},
			want:     nil,
		},
		{
			name:     "zone for a hostname no service asks for is ignored",
			services: compose.Services{web: {Ports: ingress}},
			dnsZones: map[string]string{"ghost.example.com": zone},
			want:     nil,
		},
		{
			// Keyed by hostname, so a wildcard alias and a plain domainname are
			// resolved separately: the wildcard is Front Door's (no ACA cert job),
			// the plain one still gets one.
			name: "wildcard alias yields no cert job",
			services: compose.Services{web: {
				Ports:      ingress,
				DomainName: byod,
				Networks: map[compose.NetworkID]compose.ServiceNetworkConfig{
					compose.DefaultNetwork: {Aliases: []string{"*." + byod}},
				},
			}},
			dnsZones: map[string]string{byod: zone, "*." + byod: zone},
			want:     []certJob{{service: web, hostname: byod}},
		},
		{
			name:     "no delegate domain and no zones yields nothing",
			services: compose.Services{web: {Ports: ingress, DomainName: byod}},
			want:     nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collectCertJobs(&compose.Project{Name: "proj", Services: tt.services}, tt.domain, tt.dnsZones)
			// Map iteration order is unspecified; compare as sets.
			sortJobs(got)
			want := append([]certJob(nil), tt.want...)
			sortJobs(want)
			if len(got) != len(want) {
				t.Fatalf("collectCertJobs() = %v, want %v", got, want)
			}
			for i := range got {
				if got[i] != want[i] {
					t.Errorf("collectCertJobs()[%d] = %v, want %v", i, got[i], want[i])
				}
			}
		})
	}
}

func sortJobs(jobs []certJob) {
	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].service != jobs[j].service {
			return jobs[i].service < jobs[j].service
		}
		return jobs[i].hostname < jobs[j].hostname
	})
}
