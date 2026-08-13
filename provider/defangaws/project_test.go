package defangaws

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
)

func TestApexServiceName(t *testing.T) {
	ingress := compose.ServicePortConfig{Target: 80, Mode: compose.PortModeIngress}
	defaultMode := compose.ServicePortConfig{Target: 80} // empty mode defaults to ingress
	host := compose.ServicePortConfig{Target: 5432, Mode: compose.PortModeHost}

	tests := []struct {
		name     string
		services compose.Services
		want     string
	}{
		{
			name:     "no services",
			services: compose.Services{},
			want:     "",
		},
		{
			name: "single service single ingress port",
			services: compose.Services{
				"web": {Ports: []compose.ServicePortConfig{ingress}},
			},
			want: "web",
		},
		{
			name: "empty mode counts as ingress",
			services: compose.Services{
				"web": {Ports: []compose.ServicePortConfig{defaultMode}},
			},
			want: "web",
		},
		{
			name: "one ingress service alongside managed datastore and worker",
			services: compose.Services{
				"web":    {Ports: []compose.ServicePortConfig{ingress}},
				"db":     {Postgres: &compose.PostgresConfig{}, Ports: []compose.ServicePortConfig{defaultMode}},
				"cache":  {Redis: &compose.RedisConfig{}, Ports: []compose.ServicePortConfig{defaultMode}},
				"worker": {}, // no ports
			},
			want: "web",
		},
		{
			name: "host-only port does not qualify",
			services: compose.Services{
				"internal": {Ports: []compose.ServicePortConfig{host}},
			},
			want: "",
		},
		{
			name: "two ingress services -> unbound",
			services: compose.Services{
				"web": {Ports: []compose.ServicePortConfig{ingress}},
				"api": {Ports: []compose.ServicePortConfig{ingress}},
			},
			want: "",
		},
		{
			name: "single service with two ingress ports -> unbound",
			services: compose.Services{
				"web": {Ports: []compose.ServicePortConfig{ingress, {Target: 443, Mode: compose.PortModeIngress}}},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := apexServiceName(tt.services); got != tt.want {
				t.Errorf("apexServiceName() = %q, want %q", got, tt.want)
			}
		})
	}
}
