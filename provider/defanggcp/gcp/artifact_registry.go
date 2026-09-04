package gcp

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/artifactregistry"
	gcpconfig "github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/config"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// BuildInfra holds GCP infrastructure shared across all services with build configs.
// Created once per project when at least one service defines a build context.
type BuildInfra struct {
	Repository     *artifactregistry.Repository
	ServiceAccount *serviceaccount.Account
	// SourceBucket is the name of the shared Defang CD bucket that holds the
	// build-context archives; "" when defang:stateUrl is not a gs:// URL.
	SourceBucket    string
	BucketIAMMember *storage.BucketIAMMember // grants build SA objectViewer on SourceBucket
	RepositoryURL   pulumi.StringOutput      // e.g. "us-central1-docker.pkg.dev/project/repo"
	Region          string
	GcpProject      string
}

// cdSourceBucket returns the name of the shared Defang CD bucket, parsed out of
// the defang:stateUrl stack config (which the CD program sets from
// DEFANG_STATE_URL, e.g. "gs://defang-cd-abc123"). That bucket already holds the
// build-context archives: the CLI uploads each service's tarball to it with a
// presigned URL and rewrites build.context to "gs://<bucket>/uploads/<digest>.tar.gz"
// before CD ever runs. Returns "" when the config is unset or not a gs:// URL,
// e.g. when the Pulumi program is driven directly instead of by `defang up`.
//
// This is GCP-specific: Cloud Build needs an explicit bucket-scoped IAM grant to
// read the source object, so the build SA must be told the bucket's name (below).
// AWS's CodeBuild grants wildcard S3 read instead, and Azure's ACR Task consumes a
// CLI-issued SAS URL that's self-authorizing — neither needs to resolve a bucket/
// container name, so this stays local to the GCP package rather than living in
// provider/common.
func cdSourceBucket(ctx *pulumi.Context) string {
	stateURL := common.Defang.String("stateUrl", "").Get(ctx)
	if stateURL == "" {
		return ""
	}
	u, err := url.Parse(stateURL)
	if err != nil || u.Scheme != "gs" {
		return ""
	}
	return u.Host
}

// hasBuildConfig reports whether any service in the map defines a build context.
func hasBuildConfig(services map[string]compose.ServiceConfig) bool {
	for _, svc := range services {
		if svc.Build != nil {
			return true
		}
	}
	return false
}

// collectExternalRegistries returns the unique non-Cloud-Run-supported registries
// referenced by pre-built images across all services (build-only services are skipped
// because their image is produced by Cloud Build and pushed to Artifact Registry).
func collectExternalRegistries(services map[string]compose.ServiceConfig) []string {
	seen := map[string]bool{}
	var result []string
	for _, svc := range services {
		img := svc.StaticImage()
		if img == nil || svc.Build != nil {
			continue
		}
		info := common.ParseImage(*img)
		if info.Registry != "" && !isCloudRunSupportedRegistry(info.Registry) && !seen[info.Registry] {
			seen[info.Registry] = true
			result = append(result, info.Registry)
		}
	}
	return result
}

// artifactRegistryRepositoryID returns the explicit physical ID required by
// the Artifact Registry API. Pulumi cannot auto-name this property, so mirror
// the configured autonaming rule when present: AutonamingPrefix already
// checks for one and returns logicalName unchanged when none is set, so that
// no-op return is what tells us to fall back to the legacy prefix/project/
// stack scoping ourselves instead of re-parsing the pulumi:autonaming config.
//
// An empty logicalName asks for the project-scoped ID with no role segment at
// all; see projectRepositoryID.
func artifactRegistryRepositoryID(ctx *pulumi.Context, projectName, logicalName string) string {
	name := common.AutonamingPrefix(ctx, logicalName)
	if name == logicalName {
		parts := make([]string, 0, 4)
		if prefix := common.Prefix.Get(ctx); prefix != "" {
			parts = append(parts, prefix)
		}
		parts = append(parts, projectName, ctx.Stack())
		if logicalName != "" {
			parts = append(parts, logicalName)
		}
		name = strings.Join(parts, "-")
	}
	return sanitizeRepoName(name)
}

// projectRepositoryID returns the physical ID of the project's own image
// repository: "<prefix>-<project>-<stack>", with no role segment.
//
// The segment is deliberately absent. "repo" on an Artifact Registry
// repository says nothing the resource type does not already say, and dropping
// it makes this ID identical to the one the legacy defang-mvp CD chose
// (resourceName(project, config) with no extra parts -- see
// legacy_aliases.go). repositoryId is ForceNew, so an alias alone cannot stop
// a replacement when it differs: matching the legacy ID is what turns the
// adoption of a legacy repository into an update instead of a
// create-replacement.
//
// The two CDs still part company on names long enough to need trimming: the
// legacy CD hash-trimmed at 55 characters with a base36 hash, this one trims
// at 63 with a hex hash. Such a project adopts the URN and then replaces the
// repository anyway. It holds images rather than data and the replacement
// succeeds cleanly, so that residue is left alone rather than pinning today's
// naming to the legacy trim forever.
func projectRepositoryID(ctx *pulumi.Context, projectName string) string {
	return artifactRegistryRepositoryID(ctx, projectName, "")
}

// createRemoteRepos creates Artifact Registry REMOTE repositories that act as
// pull-through caches for external Docker registries not natively supported by
// Cloud Run. One stack-scoped repository is created per unique registry.
func createRemoteRepos(
	ctx *pulumi.Context,
	projectName string,
	registries []string,
	opts ...pulumi.ResourceOption,
) (map[string]*artifactregistry.Repository, error) {
	repos := make(map[string]*artifactregistry.Repository, len(registries))
	for _, registry := range registries {
		repoID := artifactRegistryRepositoryID(ctx, projectName, registry)
		repo, err := artifactregistry.NewRepository(ctx, registry, &artifactregistry.RepositoryArgs{
			// RepositoryId must be set explicitly. Some AWS resources say "if omitted, the provider
			// will assign a random, unique name" (e.g. https://www.pulumi.com/registry/packages/aws/api-docs/s3/bucket/),
			// but the GCP Artifact Registry docs make no such promise:
			// https://www.pulumi.com/registry/packages/gcp/api-docs/artifactregistry/repository/
			RepositoryId: pulumi.String(repoID),
			// Location:     pulumi.String(region),
			Description: pulumi.String("Remote pull-through cache for " + registry),
			Format:      pulumi.String("DOCKER"),
			Mode:        pulumi.String("REMOTE_REPOSITORY"),
			RemoteRepositoryConfig: &artifactregistry.RepositoryRemoteRepositoryConfigArgs{
				CommonRepository: &artifactregistry.RepositoryRemoteRepositoryConfigCommonRepositoryArgs{
					Uri: pulumi.String("https://" + registry),
				},
			},
			// Remote repositories contain only an upstream cache. Delete them normally;
			// retaining one after `down` leaves an untracked ID that blocks the next `up`.
		}, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating remote repository for %s: %w", registry, err)
		}
		repos[registry] = repo
	}
	return repos, nil
}

// createBuildInfra creates the shared GCP infrastructure required to build container images:
// an Artifact Registry repository, a build service account, and the associated IAM bindings.
//
// No GCS bucket is created for build sources: the build context already lives in the
// shared Defang CD bucket (see cdSourceBucket), so the build service account only needs
// bucket-scoped read access to it. Creating a per-project bucket here needed
// project-level storage.buckets.create on the CD service account, which it does not have.
func createBuildInfra(
	ctx *pulumi.Context,
	projectName string,
	opts ...pulumi.ResourceOption,
) (*BuildInfra, error) {
	region := GcpRegion(ctx)
	gcpProject := gcpconfig.GetProject(ctx)

	bsa, err := serviceaccount.NewAccount(ctx, "builder", &serviceaccount.AccountArgs{
		DisplayName: pulumi.String("Image build service account for " + projectName),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating build service account: %w", err)
	}

	ar, err := artifactregistry.NewRepository(ctx, "repo", &artifactregistry.RepositoryArgs{
		// RepositoryId is required by the GCP API; unlike AWS, GCP does not auto-generate resource IDs.
		RepositoryId: pulumi.String(projectRepositoryID(ctx, projectName)),
		Location:     pulumi.String(region),
		Description:  pulumi.String("Docker images for " + projectName),
		Format:       pulumi.String("DOCKER"),
	}, common.MergeOptions(opts,
		// resourceName(project, config) with no extra parts in the legacy CD --
		// which is also the repositoryId above, so the adoption is an update
		// rather than a create-replacement.
		legacyAlias(legacyResourceName(ctx, projectName)))...)
	if err != nil {
		return nil, fmt.Errorf("creating artifact registry repository: %w", err)
	}

	saOpts := make([]pulumi.ResourceOption, 0, len(opts)+1)
	saOpts = append(append(saOpts, opts...), pulumi.DeleteBeforeReplace(true))

	// Project is deliberately omitted: RepositoryIamBinding falls back to the
	// provider's configured project when unset (unlike projects.IAMMember
	// below, whose Project field is required and not inferred from the
	// provider -- see https://github.com/DefangLabs/pulumi-defang/pull/423#discussion_r3832857083).
	repoIAMArgs := &artifactregistry.RepositoryIamBindingArgs{
		Location:   pulumi.String(region),
		Repository: ar.Name,
		Role:       pulumi.String("roles/artifactregistry.admin"),
		Members:    pulumi.StringArray{pulumi.Sprintf("serviceAccount:%v", bsa.Email)},
	}
	// Logical names below deliberately omit projectName, same as the VPC/firewalls
	// in gcp.go: Pulumi's default resource ID already prefixes it with
	// <pulumi-project>-<stack>, which includes projectName, so repeating it here
	// risked exceeding GCP's 63-char resource ID limit.
	if _, err := artifactregistry.NewRepositoryIamBinding(
		ctx, "registry-admin", repoIAMArgs, saOpts...,
	); err != nil {
		return nil, fmt.Errorf("binding artifact registry admin role: %w", err)
	}

	// Bucket-scoped read access on the shared CD bucket, so Cloud Build can fetch
	// the source archive the CLI uploaded there. The CD service account holds
	// roles/storage.admin on that bucket (granted by the CLI), which includes
	// storage.buckets.setIamPolicy, so it can add this member. Targeting the bucket
	// by name does not require Pulumi to manage the bucket itself.
	sourceBucket := cdSourceBucket(ctx)
	var sourceViewer *storage.BucketIAMMember
	if sourceBucket != "" {
		sourceViewer, err = storage.NewBucketIAMMember(ctx, "source-viewer", &storage.BucketIAMMemberArgs{
			Bucket: pulumi.String(sourceBucket),
			Role:   pulumi.String("roles/storage.objectViewer"),
			Member: pulumi.Sprintf("serviceAccount:%v", bsa.Email),
		}, saOpts...)
		if err != nil {
			return nil, fmt.Errorf("binding storage.objectViewer role: %w", err)
		}
	}

	if _, err := projects.NewIAMMember(ctx, "logWriter", &projects.IAMMemberArgs{
		Project: pulumi.String(gcpProject),
		Role:    pulumi.String("roles/logging.logWriter"),
		Member:  pulumi.Sprintf("serviceAccount:%v", bsa.Email),
	}, saOpts...); err != nil {
		return nil, fmt.Errorf("binding logging.logWriter role: %w", err)
	}

	if _, err := projects.NewIAMMember(ctx, "bucketWriter", &projects.IAMMemberArgs{
		Project: pulumi.String(gcpProject),
		Role:    pulumi.String("roles/logging.bucketWriter"),
		Member:  pulumi.Sprintf("serviceAccount:%v", bsa.Email),
	}, saOpts...); err != nil {
		return nil, fmt.Errorf("binding logging.bucketWriter role: %w", err)
	}

	repoURL := pulumi.Sprintf("%s-docker.pkg.dev/%s/%s", region, gcpProject, ar.RepositoryId)

	return &BuildInfra{
		Repository:      ar,
		ServiceAccount:  bsa,
		SourceBucket:    sourceBucket,
		BucketIAMMember: sourceViewer,
		RepositoryURL:   repoURL,
		Region:          region,
		GcpProject:      gcpProject,
	}, nil
}
