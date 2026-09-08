package gcp

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/artifactregistry"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// buildCleanupPolicies is the Artifact Registry equivalent of the AWS build
// repo's lifecycle policy (see provider/defangaws/aws/ecr_policy.go and
// provider/common.KeepBuildImages): without it, a repository keeps every
// version pushed to it forever.
//
// Artifact Registry evaluates cleanup policies differently from ECR: a
// version matching ANY "KEEP" policy is protected even if it also matches a
// "DELETE" policy, whereas ECR's rules each act independently. That means
// keepNewest below fully protects the newest common.KeepBuildImages versions
// from expireRest, so expireUntagged only ever fires on an orphaned untagged
// version once it has already aged out of that window — a narrower window
// than ECR's independent 1-day rule, but the same direction (safer, not more
// aggressive), so it's an acceptable difference rather than a bug to chase.
func buildCleanupPolicies() artifactregistry.RepositoryCleanupPolicyArray {
	return artifactregistry.RepositoryCleanupPolicyArray{
		&artifactregistry.RepositoryCleanupPolicyArgs{
			Id:     pulumi.String("expire-untagged"),
			Action: pulumi.String("DELETE"),
			Condition: &artifactregistry.RepositoryCleanupPolicyConditionArgs{
				TagState:  pulumi.String("UNTAGGED"),
				OlderThan: pulumi.String(fmt.Sprintf("%dd", common.ExpireUntaggedDays)),
			},
		},
		&artifactregistry.RepositoryCleanupPolicyArgs{
			Id:     pulumi.String("keep-newest"),
			Action: pulumi.String("KEEP"),
			MostRecentVersions: &artifactregistry.RepositoryCleanupPolicyMostRecentVersionsArgs{
				KeepCount: pulumi.Int(common.KeepBuildImages),
			},
		},
		&artifactregistry.RepositoryCleanupPolicyArgs{
			Id:     pulumi.String("expire-rest"),
			Action: pulumi.String("DELETE"),
			Condition: &artifactregistry.RepositoryCleanupPolicyConditionArgs{
				TagState: pulumi.String("ANY"),
			},
		},
	}
}
