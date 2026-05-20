package azure

import (
	"math"
	"sort"

	"github.com/pulumi/pulumi-azure-native-sdk/dbforpostgresql/v3"
)

// skuInfo describes a Postgres Flexible Server SKU's resources and a relative cost.
type skuInfo struct {
	vcpu int
	gib  float64
	cost float64 // approximate USD/hour, used only to rank candidates within a tier
}

// SKU catalogs for Azure Database for PostgreSQL Flexible Server.
// Specs from MS Learn (https://learn.microsoft.com/azure/postgresql/flexible-server/concepts-compute);
// approximate East US PAYG hourly prices for ranking only — not authoritative for billing.
//
// We expose `_ds_v5` (Intel) for D/E series; AMD `_ads_v5` and v4/v3 generations are
// also accepted by the API but `_ds_v5` is current-gen and the broadest fit.

// Burstable (B-series): single CPU credit family, no generation suffix.
var burstableSkus = map[string]skuInfo{
	"Standard_B1ms":  {vcpu: 1, gib: 2, cost: 0.022},
	"Standard_B2s":   {vcpu: 2, gib: 4, cost: 0.041},
	"Standard_B2ms":  {vcpu: 2, gib: 8, cost: 0.083},
	"Standard_B4ms":  {vcpu: 4, gib: 16, cost: 0.166},
	"Standard_B8ms":  {vcpu: 8, gib: 32, cost: 0.331},
	"Standard_B12ms": {vcpu: 12, gib: 48, cost: 0.497},
	"Standard_B16ms": {vcpu: 16, gib: 64, cost: 0.662},
	"Standard_B20ms": {vcpu: 20, gib: 80, cost: 0.828},
}

// General Purpose (D-series, v5 Intel).
var generalSkus = map[string]skuInfo{
	"Standard_D2ds_v5":  {vcpu: 2, gib: 8, cost: 0.182},
	"Standard_D4ds_v5":  {vcpu: 4, gib: 16, cost: 0.364},
	"Standard_D8ds_v5":  {vcpu: 8, gib: 32, cost: 0.728},
	"Standard_D16ds_v5": {vcpu: 16, gib: 64, cost: 1.456},
	"Standard_D32ds_v5": {vcpu: 32, gib: 128, cost: 2.911},
	"Standard_D48ds_v5": {vcpu: 48, gib: 192, cost: 4.367},
	"Standard_D64ds_v5": {vcpu: 64, gib: 256, cost: 5.823},
	"Standard_D96ds_v5": {vcpu: 96, gib: 384, cost: 8.734},
}

// Memory Optimized (E-series, v5 Intel).
var memoryOptimizedSkus = map[string]skuInfo{
	"Standard_E2ds_v5":  {vcpu: 2, gib: 16, cost: 0.234},
	"Standard_E4ds_v5":  {vcpu: 4, gib: 32, cost: 0.468},
	"Standard_E8ds_v5":  {vcpu: 8, gib: 64, cost: 0.936},
	"Standard_E16ds_v5": {vcpu: 16, gib: 128, cost: 1.872},
	"Standard_E20ds_v5": {vcpu: 20, gib: 160, cost: 2.340},
	"Standard_E32ds_v5": {vcpu: 32, gib: 256, cost: 3.744},
	"Standard_E48ds_v5": {vcpu: 48, gib: 384, cost: 5.616},
	"Standard_E64ds_v5": {vcpu: 64, gib: 512, cost: 7.487},
	"Standard_E96ds_v5": {vcpu: 96, gib: 672, cost: 11.231},
}

// skuCatalog pairs a tier with its SKU table so the picker can return both
// without having to parse the tier back out of the chosen SKU name.
type skuCatalog struct {
	tier dbforpostgresql.SkuTier
	skus map[string]skuInfo
}

var (
	burstableCatalog       = skuCatalog{dbforpostgresql.SkuTierBurstable, burstableSkus}
	generalCatalog         = skuCatalog{dbforpostgresql.SkuTierGeneralPurpose, generalSkus}
	memoryOptimizedCatalog = skuCatalog{dbforpostgresql.SkuTierMemoryOptimized, memoryOptimizedSkus}
)

// tierCatalogs maps recipe postgres-tier values to their catalog search order.
// A workload that won't fit in the preferred tier falls back to the next larger one,
// matching the AWS provider's behavior.
var tierCatalogs = map[string][]skuCatalog{
	"burstable":        {burstableCatalog, generalCatalog, memoryOptimizedCatalog},
	"general":          {generalCatalog, memoryOptimizedCatalog},
	"memory-optimized": {memoryOptimizedCatalog},
}

// postgresFlexibleSku returns the cheapest SKU that meets the requested CPU/memory
// budget along with its tier. tier selects the preferred tier ("burstable",
// "general", "memory-optimized"); unknown values fall back to "burstable".
func postgresFlexibleSku(minCPUs float64, minMemoryMiB int, tier string) (string, dbforpostgresql.SkuTier) {
	minGiB := float64(minMemoryMiB) / 1024

	catalogs, ok := tierCatalogs[tier]
	if !ok {
		catalogs = tierCatalogs["burstable"]
	}

	for _, c := range catalogs {
		if name := cheapestSkuMatch(c.skus, minCPUs, minGiB); name != "" {
			return name, c.tier
		}
	}

	// Fallback: smallest burstable. Reached only if every catalog is empty,
	// which shouldn't happen with the tables above.
	return "Standard_B1ms", dbforpostgresql.SkuTierBurstable
}

// cheapestSkuMatch finds the cheapest SKU in catalog meeting the minimum requirements.
func cheapestSkuMatch(catalog map[string]skuInfo, minCPUs float64, minGiB float64) string {
	type entry struct {
		name string
		cost float64
	}
	var matches []entry
	minVcpu := int(math.Ceil(minCPUs))
	for name, info := range catalog {
		if info.vcpu >= minVcpu && info.gib >= minGiB {
			matches = append(matches, entry{name, info.cost})
		}
	}
	if len(matches) == 0 {
		return ""
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].cost != matches[j].cost {
			return matches[i].cost < matches[j].cost
		}
		// Stable order on ties so the same inputs always pick the same SKU.
		return matches[i].name < matches[j].name
	})
	return matches[0].name
}
