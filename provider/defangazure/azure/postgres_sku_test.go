package azure

import (
	"testing"

	"github.com/pulumi/pulumi-azure-native-sdk/dbforpostgresql/v3"
	"github.com/stretchr/testify/assert"
)

func TestPostgresFlexibleSku(t *testing.T) {
	tests := []struct {
		name     string
		cpus     float64
		memMiB   int
		tier     string
		wantSku  string
		wantTier dbforpostgresql.SkuTier
	}{
		{
			name:     "default-burstable-no-reservation",
			cpus:     0,
			memMiB:   0,
			tier: "burstable",
			wantSku:  "Standard_B1ms",
			wantTier: dbforpostgresql.SkuTierBurstable,
		},
		{
			name:     "burstable-2cpu-4gib",
			cpus:     2,
			memMiB:   4096,
			tier: "burstable",
			wantSku:  "Standard_B2s",
			wantTier: dbforpostgresql.SkuTierBurstable,
		},
		{
			name:     "burstable-2cpu-8gib-needs-B2ms",
			cpus:     2,
			memMiB:   8192,
			tier: "burstable",
			wantSku:  "Standard_B2ms",
			wantTier: dbforpostgresql.SkuTierBurstable,
		},
		{
			// 128 GiB doesn't fit in any burstable; fallback chain goes
			// burstable → general, where D32ds_v5 is the cheapest 128 GiB option.
			// We don't continue to memory-optimized once general has a match.
			name:     "burstable-overflows-to-general",
			cpus:     2,
			memMiB:   128 * 1024,
			tier: "burstable",
			wantSku:  "Standard_D32ds_v5",
			wantTier: dbforpostgresql.SkuTierGeneralPurpose,
		},
		{
			// 500 GiB doesn't fit in burstable or general; only E64ds_v5 (512 GiB)
			// in memory-optimized matches. Exercises the full fallback chain.
			name:     "exceeds-general-falls-through-to-memory-optimized",
			cpus:     2,
			memMiB:   500 * 1024,
			tier: "burstable",
			wantSku:  "Standard_E64ds_v5",
			wantTier: dbforpostgresql.SkuTierMemoryOptimized,
		},
		{
			name:     "general-tier-explicit",
			cpus:     4,
			memMiB:   16 * 1024,
			tier: "general",
			wantSku:  "Standard_D4ds_v5",
			wantTier: dbforpostgresql.SkuTierGeneralPurpose,
		},
		{
			name:     "memory-optimized-tier",
			cpus:     2,
			memMiB:   16 * 1024,
			tier: "memory-optimized",
			wantSku:  "Standard_E2ds_v5",
			wantTier: dbforpostgresql.SkuTierMemoryOptimized,
		},
		{
			name:     "unknown-node-type-falls-back-to-burstable",
			cpus:     0,
			memMiB:   0,
			tier: "wat",
			wantSku:  "Standard_B1ms",
			wantTier: dbforpostgresql.SkuTierBurstable,
		},
		{
			name:     "fractional-cpu-rounds-up",
			cpus:     1.1,
			memMiB:   2048,
			tier: "burstable",
			wantSku:  "Standard_B2s",
			wantTier: dbforpostgresql.SkuTierBurstable,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotSku, gotTier := postgresFlexibleSku(tc.cpus, tc.memMiB, tc.tier)
			assert.Equal(t, tc.wantSku, gotSku)
			assert.Equal(t, tc.wantTier, gotTier)
		})
	}
}
