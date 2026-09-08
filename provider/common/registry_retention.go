package common

// Registry image-retention policy shared across providers' build repositories.
// None of the three providers' registry resources expire anything by default
// (AWS ecr.Repository, GCP artifactregistry.Repository, Azure ContainerRegistry
// all keep every pushed image forever), so every project accumulates storage
// without bound. Measured on the Defang org 2026-08-11: 2.13 TB of ECR storage
// alone ($216/month), 90% of it expirable. See DefangLabs/defang-global#112.
const (
	// KeepBuildImages bounds a per-project build repo. Each provider's build
	// path pushes every build to (effectively) one mutable tag rather than
	// deploying by digest, so the live image is always the newest one and a
	// count rule can never expire it — the count is a cap on churn from
	// superseded builds, not an in-use-image guard. A count is preferred over
	// an age rule so a rarely-rebuilt project keeps its image indefinitely
	// (elsewhere in the Defang org, live tasks reference images up to 742 days
	// old).
	KeepBuildImages = 20

	// KeepCacheImages bounds each pull-through/remote cache repo. Mirrored
	// images can always be re-fetched from upstream, so this can be small.
	KeepCacheImages = 10

	// ExpireUntaggedDays clears superseded build layers quickly, without
	// waiting for KeepBuildImages to churn them out. Applied to build repos
	// only: a cache repo's mirrored images arrive untagged by design (pulled
	// by digest), so an untagged rule there would empty the whole cache and
	// put every deploy back on the upstream registry's rate limits
	// (DefangLabs/defang-mvp#2487).
	ExpireUntaggedDays = 1
)
