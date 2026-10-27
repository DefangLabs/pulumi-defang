package scaleway

import "github.com/DefangLabs/pulumi-defang/provider/common"

// Scaleway recipe config accessors. Cloud-specific keys read from defang-scaleway:<key>.
var recipe = common.NewRecipe("defang-scaleway")

var (
	// BackupRetentionDays keeps daily managed Postgres backups for this many
	// days; 0 disables them.
	BackupRetentionDays = recipe.Int("backup-retention-days", 0)
	// BuildInstanceFamily picks the Compute Instance family image builds run
	// on; build.shm_size picks the size within it (see SelectBuildInstanceType).
	// The affordable "dev" family also boots from a local disk, which is
	// deleted along with the build instance.
	BuildInstanceFamily = recipe.String("build-instance-family", "dev")
	// HighAvailability adds a standby node to managed Postgres instances.
	HighAvailability = recipe.Bool("high-availability", false)
	// LogRetentionDays is how long the project's Cockpit build-logs data
	// source keeps logs.
	LogRetentionDays = recipe.Int("log-retention-days", 1)
	// PostgresNodeType picks the managed Postgres node family: "dev" (shared
	// vCPU, then general purpose beyond its largest size) or "gp" (dedicated
	// vCPU). The service's CPU and memory pick the size within it.
	PostgresNodeType = recipe.String("postgres-node-type", "dev")
	// TransitEncryptionDisabled turns off TLS to managed Redis. It defaults to
	// true, unlike GCP: Scaleway's Redis TLS certificate is not publicly
	// trusted, so a client connecting with rediss:// fails verification unless
	// it is given the cluster's certificate.
	TransitEncryptionDisabled = recipe.Bool("transit-encryption-disabled", true)
)
