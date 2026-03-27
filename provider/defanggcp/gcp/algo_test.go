package gcp

import (
	"fmt"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/stretchr/testify/assert"
)

func TestIsCloudRunService(t *testing.T) {
	tests := []struct {
		name     string
		ports    []compose.ServicePortConfig
		expected bool
	}{
		{
			name:     "no ports",
			ports:    nil,
			expected: false,
		},
		{
			name:     "single ingress port",
			ports:    []compose.ServicePortConfig{{Target: 8080, Mode: "ingress"}},
			expected: true,
		},
		{
			name:     "single host port",
			ports:    []compose.ServicePortConfig{{Target: 5432, Mode: "host"}},
			expected: false,
		},
		{
			name: "two ingress ports",
			ports: []compose.ServicePortConfig{
				{Target: 80, Mode: "ingress"},
				{Target: 443, Mode: "ingress"},
			},
			expected: false,
		},
		{
			name: "one ingress and one host port",
			ports: []compose.ServicePortConfig{
				{Target: 8080, Mode: "ingress"},
				{Target: 9090, Mode: "host"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := compose.ServiceConfig{Ports: tt.ports}
			assert.Equal(t, tt.expected, IsCloudRunService(svc))
		})
	}
}

func TestGetComputeMachineType(t *testing.T) {
	tests := []struct {
		name        string
		cpus        float64
		memoryMiB   int
		wantMachine string
	}{
		{
			name:        "no reservations defaults to e2-micro",
			cpus:        0.25, // GetCPUs default
			memoryMiB:   512,  // GetMemoryMiB default
			wantMachine: "e2-micro",
		},
		{
			name:        "1 CPU 512MiB fits e2-medium",
			cpus:        1.0,
			memoryMiB:   512,
			wantMachine: "e2-medium",
		},
		{
			name:        "2 CPU 8GiB fits e2-standard-2",
			cpus:        2.0,
			memoryMiB:   8 * 1024,
			wantMachine: "e2-standard-2",
		},
		{
			name:        "0.5 CPU 2GiB fits e2-small",
			cpus:        0.5,
			memoryMiB:   2 * 1024,
			wantMachine: "e2-small",
		},
		{
			name:        "4 CPU 16GiB fits e2-standard-4",
			cpus:        4.0,
			memoryMiB:   16 * 1024,
			wantMachine: "e2-standard-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mem := tt.memoryMiB
			cpus := tt.cpus
			memStr := "512M"
			if mem != 512 {
				memStr = formatMiB(mem)
			}
			svc := compose.ServiceConfig{
				Deploy: &compose.DeployConfig{
					Resources: &compose.Resources{
						Reservations: &compose.ResourceConfig{
							CPUs:   &cpus,
							Memory: &memStr,
						},
					},
				},
			}
			got := getComputeMachineType(svc)
			assert.Equal(t, tt.wantMachine, got)
		})
	}
}

// formatMiB formats a MiB value as a memory string for ResourceConfig.
func formatMiB(mib int) string {
	if mib >= 1024 && mib%1024 == 0 {
		return fmt.Sprintf("%dGi", mib/1024)
	}
	return fmt.Sprintf("%dMi", mib)
}
