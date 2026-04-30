package gcp

import (
	"fmt"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
)

// TestBuildMIGUpdatePolicyMaxSurgeConstraint verifies that MaxSurgeFixed is
// always either 0 or >= numZones, which is the constraint enforced by the GCP
// API for regional managed instance groups.
func TestBuildMIGUpdatePolicyMaxSurgeConstraint(t *testing.T) {
	tests := []struct {
		name       string
		numZones   int
		targetSize int
	}{
		{"single replica, 4-zone region (us-central1)", 4, 1},
		{"single replica, 3-zone region", 3, 1},
		{"two replicas, 4-zone region", 4, 2},
		{"ten replicas, 4-zone region", 4, 10},
		{"zero zones (unknown/mock)", 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := buildMIGUpdatePolicy(tt.numZones, tt.targetSize)

			// Percent-based policies (targetSize > 10) don't use fixed counts.
			if policy.MaxSurgeFixed == nil {
				assert.NotNil(t, policy.MaxSurgePercent, "expected percent-based surge for large targetSize")
				return
			}

			surgeFixed := int(policy.MaxSurgeFixed.(pulumi.Int))
			unavailFixed := int(policy.MaxUnavailableFixed.(pulumi.Int))

			effectiveZones := tt.numZones
			if effectiveZones < 1 {
				effectiveZones = 1
			}

			assert.True(t, surgeFixed == 0 || surgeFixed >= effectiveZones,
				"MaxSurgeFixed=%d must be 0 or >= numZones=%d", surgeFixed, effectiveZones)
			assert.True(t, unavailFixed == 0 || unavailFixed >= effectiveZones,
				"MaxUnavailableFixed=%d must be 0 or >= numZones=%d", unavailFixed, effectiveZones)
		})
	}
}

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
			// compose YAML may omit `mode:`; the loader leaves Mode == "" and we
			// must still treat it as ingress (Cloud Run), otherwise the service
			// falls through to Compute Engine and Endpoints returns a raw IP.
			name:     "single port with unset mode defaults to ingress",
			ports:    []compose.ServicePortConfig{{Target: 80}},
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
			svc := &compose.ServiceConfig{Ports: tt.ports}
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
