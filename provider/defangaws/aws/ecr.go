package aws

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecr"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

type ecrResult struct {
	repository *ecr.Repository
	repoURL    pulumix.Output[string]
}

// createECRRepo creates an ECR repository for built images.
func createECRRepo(
	ctx *pulumi.Context,
	name string,
	opts ...pulumi.ResourceOption,
) (*ecrResult, error) {
	repo, err := ecr.NewRepository(ctx, name, &ecr.RepositoryArgs{
		ForceDelete:        pulumi.Bool(true),
		ImageTagMutability: pulumi.String("MUTABLE"),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ECR repository: %w", err)
	}

	return &ecrResult{
		repository: repo,
		repoURL:    pulumix.Output[string](repo.RepositoryUrl),
	}, nil
}

// PullThroughCache holds a pull-through cache rule and its resolved prefix URL.
type PullThroughCache struct {
	Rule *ecr.PullThroughCacheRule
	// CachePrefix is the full ECR mirror URL prefix: {registryId}.dkr.ecr.{region}.amazonaws.com/{prefix}
	CachePrefix pulumi.StringOutput
}

// maxCacheRepoLength is the longest ecrRepositoryPrefix AWS accepts.
// It was 20 before https://github.com/hashicorp/terraform-provider-aws/pull/34716.
const maxCacheRepoLength = 30

// cachePrefixHashLength is how much of an over-long prefix is given over to a
// hash of the whole of it.
const cachePrefixHashLength = 6

// cachePrefixSeparators are the characters AWS allows inside an
// ecrRepositoryPrefix but not at its end.
const cachePrefixSeparators = "._-/"

// cachePrefixName fits a seed into ecrRepositoryPrefix's 30 characters.
//
// Plain truncation was not enough on two counts. AWS rejects a prefix that
// ends in a separator, and cutting "defang-mastra-extended-e2eaws-mastra-
// extended-ecr-public" at 30 characters produces exactly that -- the deploy
// failed with "invalid value for ecr_repository_prefix", whose message lists
// the allowed characters and so points away from the real cause. And the
// prefix is an ACCOUNT-GLOBAL namespace, so two projects whose names agree for
// the first 30 characters would otherwise claim one rule.
//
// Trading the tail for a hash of the full seed answers both. Seeds that fit
// are returned unchanged, and existing rules keep their current prefix through
// IgnoreChanges below, so nothing already deployed moves.
func cachePrefixName(prefix string) string {
	if len(prefix) <= maxCacheRepoLength {
		return strings.TrimRight(prefix, cachePrefixSeparators)
	}

	digest := sha256.Sum256([]byte(prefix))
	hash := hex.EncodeToString(digest[:])[:cachePrefixHashLength]
	head := strings.TrimRight(prefix[:maxCacheRepoLength-cachePrefixHashLength-1], cachePrefixSeparators)
	if head == "" {
		return hash
	}
	return head + "-" + hash
}

// createEcrPullThroughCache creates an ECR pull-through cache rule for the given upstream registry.
// Matches TS createEcrPullThroughCache in shared/aws/repos.ts. prefixSeed
// seeds the repository prefix — an ACCOUNT-global namespace, so it must be
// scoped (e.g. by project) to avoid colliding with rules owned by other
// stacks/programs in the same account; name stays the (stable) resource name.
func createEcrPullThroughCache(
	ctx *pulumi.Context,
	name string,
	prefixSeed string,
	upstreamRegistryURL pulumi.StringInput,
	opts ...pulumi.ResourceOption,
) (*PullThroughCache, error) {
	// PullThroughCacheRule does not support autonaming, so we need to generate a unique and compliant prefix ourselves.
	prefix := cachePrefixName(strings.ToLower(common.AutonamingPrefix(ctx, prefixSeed)))
	rule, err := ecr.NewPullThroughCacheRule(ctx, name, &ecr.PullThroughCacheRuleArgs{
		EcrRepositoryPrefix: pulumi.String(prefix),
		UpstreamRegistryUrl: upstreamRegistryURL,
	}, common.MergeOptions(opts,
		pulumi.IgnoreChanges([]string{"ecrRepositoryPrefix"}),
	)...)
	if err != nil {
		return nil, fmt.Errorf("creating ECR pull-through cache rule %q: %w", name, err)
	}

	// Build the full ECR mirror URL prefix: {registryId}.dkr.ecr.{region}.amazonaws.com/{prefix}
	cachePrefix := pulumi.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s",
		rule.RegistryId, rule.Region, rule.EcrRepositoryPrefix)

	return &PullThroughCache{
		Rule:        rule,
		CachePrefix: cachePrefix,
	}, nil
}
