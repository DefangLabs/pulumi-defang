package aws

import (
	"encoding/json"

	"github.com/DefangLabs/pulumi-defang/provider/common"
)

// ECR retention. Without a lifecycle policy an ECR repository keeps every image
// forever: unlike awsx, the raw ecr.Repository resource applies no default at
// all, not even for untagged images. Measured on the Defang org 2026-08-11:
// 2.13 TB of ECR storage ($216/month) of which 90% was expirable, and one
// repository (prod1/kaniko-build) held 11,351 images.
// See DefangLabs/defang-global#112. The retention shape (common.KeepBuildImages,
// common.KeepCacheImages, common.ExpireUntaggedDays) is shared with the GCP and
// Azure equivalents — this provider's specific note: the repo holds a single
// mutable :latest tag (codebuild.go pushes every service's build there; nothing
// is deployed by digest), so the live image is always the newest and a count
// rule can never expire it — the count is a cap on sub-24h untagged churn, not
// an in-use-image guard (elsewhere in the Defang org, live tasks reference
// images up to 742 days old).

type lifecycleSelection struct {
	TagStatus   string `json:"tagStatus"`
	CountType   string `json:"countType"`
	CountUnit   string `json:"countUnit,omitempty"`
	CountNumber int    `json:"countNumber"`
}

type lifecycleRule struct {
	RulePriority int                `json:"rulePriority"`
	Description  string             `json:"description"`
	Selection    lifecycleSelection `json:"selection"`
	Action       struct {
		Type string `json:"type"`
	} `json:"action"`
}

type lifecyclePolicy struct {
	Rules []lifecycleRule `json:"rules"`
}

func expireRule(priority int, description string, selection lifecycleSelection) lifecycleRule {
	rule := lifecycleRule{
		RulePriority: priority,
		Description:  description,
		Selection:    selection,
	}
	rule.Action.Type = "expire"
	return rule
}

// buildLifecyclePolicy is the policy for a repository that holds images we build.
func buildLifecyclePolicy(keepImages int) (string, error) {
	policy := lifecyclePolicy{Rules: []lifecycleRule{
		expireRule(1, "expire untagged images", lifecycleSelection{
			TagStatus:   "untagged",
			CountType:   "sinceImagePushed",
			CountUnit:   "days",
			CountNumber: common.ExpireUntaggedDays,
		}),
		expireRule(2, "keep the newest images", lifecycleSelection{
			TagStatus:   "any",
			CountType:   "imageCountMoreThan",
			CountNumber: keepImages,
		}),
	}}
	out, err := json.Marshal(policy)
	return string(out), err
}

// cacheLifecyclePolicy is the policy for repositories that ECR creates for a
// pull-through cache. It has exactly one rule and no untagged rule on purpose:
// images pulled by digest arrive untagged, so an untagged rule empties the whole
// cache. An empty cache still works, but it puts every deploy back on the
// upstream registry and its rate limits (DefangLabs/defang-mvp#2487).
func cacheLifecyclePolicy(keepImages int) (string, error) {
	policy := lifecyclePolicy{Rules: []lifecycleRule{
		expireRule(1, "keep the newest mirrored images", lifecycleSelection{
			TagStatus:   "any",
			CountType:   "imageCountMoreThan",
			CountNumber: keepImages,
		}),
	}}
	out, err := json.Marshal(policy)
	return string(out), err
}
