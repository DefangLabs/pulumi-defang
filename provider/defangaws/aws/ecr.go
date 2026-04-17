package aws

import (
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

// createEcrPullThroughCache creates an ECR pull-through cache rule for the given upstream registry.
// Matches TS createEcrPullThroughCache in shared/aws/repos.ts.
func createEcrPullThroughCache(
	ctx *pulumi.Context,
	name string,
	upstreamRegistryURL pulumi.StringInput,
	opts ...pulumi.ResourceOption,
) (*PullThroughCache, error) {
	const maxCacheRepoLength = 30 // was 20 https://github.com/hashicorp/terraform-provider-aws/pull/34716
	// PullThroughCacheRule does not support autonaming, so we need to generate a unique and compliant prefix ourselves.
	prefix := strings.ToLower(common.AutonamingPrefix(ctx, name))
	if len(prefix) > maxCacheRepoLength {
		// 	TODO: hashTrim/truncate prefix smartly
		prefix = prefix[:maxCacheRepoLength]
	}
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
