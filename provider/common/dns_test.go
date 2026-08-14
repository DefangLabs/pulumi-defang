package common

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
)

func TestNormalizeDNS(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"foo.bar.", "foo.bar"},
		{"Foo.bar", "foo.bar"},
		{"foo.bar", "foo.bar"},
		{"FOO.BAR.", "foo.bar"},
		{"already.lower", "already.lower"},
	}
	for _, tt := range tests {
		if got := NormalizeDNS(tt.input); got != tt.want {
			t.Errorf("NormalizeDNS(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSafeLabel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"name", "name"},
		{"Name", "name"},
		{"name123", "name123"},
		{"name-with-hyphen", "name-with-hyphen"},
		{"name_hyphen", "name_hyphen"},
		{"name.hyphen", "name-hyphen"},
		{"name__hyphen", "name__hyphen"},
		{"12_34", "12_34"},
		{"1234", "1234"},
		{"1234.com", "1234-com"},
		{"_", "_"},
		{"...", "---"},
		{"---", "---"},
	}
	for _, tt := range tests {
		if got := SafeLabel(tt.input); got != tt.want {
			t.Errorf("SafeLabel(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestServiceLabel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"name", "name"},
		{"Name", "name"},
		{"my_svc", "my-svc"}, // underscores -> hyphens (unlike SafeLabel)
		{"My.Svc", "my-svc"}, // lowercased + dots -> hyphens
		{"a_b.c", "a-b-c"},   // combined
		{"name__hyphen", "name--hyphen"},
	}
	for _, tt := range tests {
		if got := ServiceLabel(tt.input); got != tt.want {
			t.Errorf("ServiceLabel(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestServiceFQDN(t *testing.T) {
	ingress := compose.ServicePortConfig{Target: 80, Mode: compose.PortModeIngress}
	host := compose.ServicePortConfig{Target: 5432, Mode: compose.PortModeHost}
	const pub, priv = "proj.tenant.defang.app", "proj.internal"

	tests := []struct {
		name          string
		serviceName   string
		svc           compose.ServiceConfig
		publicDomain  string
		privateDomain string
		want          string
	}{
		{
			"custom domain wins",
			"web",
			compose.ServiceConfig{DomainName: "example.com", Ports: []compose.ServicePortConfig{ingress}},
			pub,
			priv,
			"example.com",
		},
		{
			"public fqdn for ingress",
			"web",
			compose.ServiceConfig{Ports: []compose.ServicePortConfig{ingress}},
			pub,
			priv,
			"web." + pub,
		},
		{
			"private fqdn for host",
			"db",
			compose.ServiceConfig{Ports: []compose.ServicePortConfig{host}},
			pub,
			priv,
			"db." + priv,
		},
		{
			"ingress beats host",
			"web",
			compose.ServiceConfig{Ports: []compose.ServicePortConfig{ingress, host}},
			pub,
			priv,
			"web." + pub,
		},
		{
			"label sanitized",
			"my_api",
			compose.ServiceConfig{Ports: []compose.ServicePortConfig{ingress}},
			pub,
			priv,
			"my-api." + pub,
		},
		{
			"ingress but no public domain",
			"web",
			compose.ServiceConfig{Ports: []compose.ServicePortConfig{ingress}},
			"",
			priv,
			"",
		},
		{
			"host but no private domain (GCP/Azure)",
			"db",
			compose.ServiceConfig{Ports: []compose.ServicePortConfig{host}},
			pub,
			"",
			"",
		},
		{
			"no ports",
			"worker",
			compose.ServiceConfig{},
			pub,
			priv,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ServiceFQDN(tt.serviceName, tt.svc, tt.publicDomain, tt.privateDomain); got != tt.want {
				t.Errorf("ServiceFQDN(%q, ...) = %q, want %q", tt.serviceName, got, tt.want)
			}
		})
	}
}

func TestIsPrivateZone(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"foo.local", true},
		{"internal.", true},
		{"flan.", false},
		{"example.com", false},
		{"foo.internal", true},
		{"bar.private", true},
		{"corp.", true},
		{"home.arpa", true},
		{"foo.lan", true},
		{"foo.intranet", true},
	}
	for _, tt := range tests {
		if got := IsPrivateZone(tt.input); got != tt.want {
			t.Errorf("IsPrivateZone(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestGetZoneName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"bar.com", "bar.com"},
		{"bar.co.", "bar.co."},
		{"foo.bar.com", "bar.com"},
		{"foo.bar.co.", "bar.co."},
	}
	for _, tt := range tests {
		if got := GetZoneName(tt.input); got != tt.want {
			t.Errorf("GetZoneName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsWildcardHost(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"*.example.com", true},
		{"*.a.b.example.com", true},
		{"example.com", false},
		{"", false},
		// A wildcard is only a wildcard as the leftmost label; anywhere else it
		// is just an invalid hostname, not a pattern.
		{"a.*.example.com", false},
		{"*", false},
	}
	for _, tt := range tests {
		if got := IsWildcardHost(tt.input); got != tt.want {
			t.Errorf("IsWildcardHost(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestByodHostnames(t *testing.T) {
	withAliases := func(aliases ...string) map[compose.NetworkID]compose.ServiceNetworkConfig {
		return map[compose.NetworkID]compose.ServiceNetworkConfig{
			compose.DefaultNetwork: {Aliases: aliases},
		}
	}

	tests := []struct {
		name string
		svc  compose.ServiceConfig
		want []string
	}{
		{
			name: "no domainname yields nothing even with aliases",
			svc:  compose.ServiceConfig{Networks: withAliases("alias.example.com")},
			want: nil,
		},
		{
			name: "domainname alone",
			svc:  compose.ServiceConfig{DomainName: "auth.example.com"},
			want: []string{"auth.example.com"},
		},
		{
			name: "domainname comes first, then aliases in order",
			svc: compose.ServiceConfig{
				DomainName: "auth.example.com",
				Networks:   withAliases("*.auth.example.com", "login.example.com"),
			},
			want: []string{"auth.example.com", "*.auth.example.com", "login.example.com"},
		},
		{
			name: "aliases on a non-default network are ignored",
			svc: compose.ServiceConfig{
				DomainName: "auth.example.com",
				Networks: map[compose.NetworkID]compose.ServiceNetworkConfig{
					"internal": {Aliases: []string{"*.internal.example.com"}},
				},
			},
			want: []string{"auth.example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ByodHostnames(tt.svc)
			if len(got) != len(tt.want) {
				t.Fatalf("ByodHostnames() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("ByodHostnames() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
