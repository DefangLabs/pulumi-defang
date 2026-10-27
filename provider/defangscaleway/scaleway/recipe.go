package scaleway

import "github.com/DefangLabs/pulumi-defang/provider/common"

// Scaleway recipe config accessors. Cloud-specific keys read from defang-scaleway:<key>.
var recipe = common.NewRecipe("defang-scaleway")

// BuildInstanceFamily picks the Compute Instance family image builds run on;
// build.shm_size picks the size within it (see SelectBuildInstanceType). The
// affordable "dev" family also boots from a local disk, which is deleted along
// with the build instance.
var BuildInstanceFamily = recipe.String("build-instance-family", "dev")
