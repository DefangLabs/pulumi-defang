package gcp

import (
	"crypto/sha1" //nolint:gosec
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/artifactregistry"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/config"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"gopkg.in/yaml.v3"
)

var errNoRemoteRepoConfigured = errors.New("no remote repository configured for registry")
var errMultiPlatformUnsupported = errors.New("multi-platform builds are unsupported on GCP Cloud Build")
var errBuildContextNotGCS = errors.New(
	"build context must be a gs:// URI; only `defang up` deploys are supported on GCP",
)

// Based on Cloud Run error:
// "Expected an image path like [host/]repo-path[:tag and/or @digest], where host is one of
// [region.]gcr.io, [region-]docker.pkg.dev or docker.io"
var gcrHostRE = regexp.MustCompile(`^(?:[a-z0-9-]+\.)*gcr\.io$`)
var dockerPkgRE = regexp.MustCompile(`^(?:[a-z][a-z0-9-]*-)?docker\.pkg\.dev$`)

// isCloudRunSupportedRegistry reports whether the given registry host is
// natively supported by Cloud Run (GCR, Artifact Registry, or Docker Hub).
func isCloudRunSupportedRegistry(registry string) bool {
	if registry == "" || registry == "docker.io" {
		return true
	}
	return gcrHostRE.MatchString(registry) || dockerPkgRE.MatchString(registry)
}

var nonLowerAlphaNumericOrDashRe = regexp.MustCompile(`[^a-z0-9-]`)

const (
	artifactRegistryRepositoryIDMaxLength  = 63
	artifactRegistryRepositoryIDHashLength = 8
)

// sanitizeRepoName produces a valid Artifact Registry repository ID:
// lowercase alphanumeric + hyphens, max 63 characters. Names longer than the
// limit are truncated with a hash of the full original name appended, so two
// names that share a long prefix still get distinct, bounded IDs.
func sanitizeRepoName(name string) string {
	original := name
	name = strings.ToLower(name)
	name = nonLowerAlphaNumericOrDashRe.ReplaceAllLiteralString(name, "-")
	name = strings.Trim(name, "-")
	if name == "" {
		return "repo-" + repositoryIDHash(original)
	}
	if len(name) > artifactRegistryRepositoryIDMaxLength {
		suffix := "-" + repositoryIDHash(original)
		maxPrefixLength := artifactRegistryRepositoryIDMaxLength - len(suffix)
		name = strings.TrimRight(name[:maxPrefixLength], "-") + suffix
	}
	return name
}

func repositoryIDHash(name string) string {
	digest := sha256.Sum256([]byte(name))
	return hex.EncodeToString(digest[:])[:artifactRegistryRepositoryIDHashLength]
}

// gcpBuildResource is the Pulumi resource state for the defang-gcp:defanggcp:Build custom resource.
type gcpBuildResource struct {
	pulumi.CustomResourceState
	BuildId     pulumi.StringOutput `pulumi:"buildId"`
	ImageDigest pulumi.StringOutput `pulumi:"imageDigest"`
}

type buildStep struct {
	Name string   `yaml:"name"`
	Args []string `yaml:"args"`
	Env  []string `yaml:"env,omitempty"`
}

// generateBuildSteps returns a YAML-encoded Cloud Build step list that builds
// a Docker image and loads it into the local Docker daemon. Cloud Build then
// pushes the image to the registry via the images: field so that build results
// contain the image digest.
//
// Multi-platform builds are rejected: the --load step and the images: push
// only handle a single-platform image (Cloud Build's daemon cannot load a
// multi-arch manifest list).
func generateBuildSteps(build *compose.BuildConfig, dest pulumi.StringOutput) pulumi.StringOutput {
	platform := "linux/amd64"
	if len(build.Platforms) == 1 {
		platform = build.Platforms[0]
	}
	buildArgs := []string{"buildx", "build", "--platform", platform}
	buildArgs = append(buildArgs, "-f", build.GetDockerfile())
	if target := build.GetTarget(); target != "" {
		buildArgs = append(buildArgs, "--target", target)
	}
	// Values travel in the step's env rather than on the command line, so they
	// stay out of the Pulumi state that records these steps. Sorted for a
	// stable ordering: a map walk would reshuffle the args every run and make
	// the Build resource look changed when it isn't.
	var buildEnv []string
	for k, v := range common.Sorted(build.Args) {
		buildArgs = append(buildArgs, "--build-arg", k)
		buildEnv = append(buildEnv, k+"="+v)
	}
	for _, c := range build.CacheFrom {
		buildArgs = append(buildArgs, "--cache-from="+c)
	}
	for _, c := range build.CacheTo {
		buildArgs = append(buildArgs, "--cache-to="+c)
	}
	return dest.ApplyT(func(d string) (string, error) {
		if len(build.Platforms) > 1 {
			return "", fmt.Errorf("%w (got %v)", errMultiPlatformUnsupported, build.Platforms)
		}
		steps := []buildStep{
			{
				Name: "gcr.io/cloud-builders/docker",
				Args: []string{
					"buildx", "create", "--use", "--name", "defangbuilder",
					"--driver", "docker-container",
				},
			},
			{
				Name: "gcr.io/cloud-builders/docker",
				Args: append(buildArgs, "-t", d, "--load", "."),
				Env:  buildEnv,
			},
		}
		b, err := yaml.Marshal(steps)
		if err != nil {
			return "", fmt.Errorf("marshaling build steps: %w", err)
		}
		return string(b), nil
	}).(pulumi.StringOutput)
}

// GetServiceImage returns the container image URI for a service.
// When svc.Build is set and infra is provided, it registers a Cloud Build
// custom resource to produce the image; otherwise it returns the pre-configured
// image string.
//
// For pre-configured images whose registry is not natively supported by Cloud Run
// (i.e. not GCR, Artifact Registry, or Docker Hub), the image reference is
// rewritten to point at the project's Artifact Registry so that Cloud Run can
// pull it. This mirrors the logic in getServiceImage in the CD implementation
// and assumes a corresponding Artifact Registry remote repository has been
// configured to proxy the original registry (see createRemoteRepos).
func GetServiceImage(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	repos map[string]*artifactregistry.Repository,
	infra *BuildInfra,
	pluginID common.PluginIdentity,
	opts ...pulumi.ResourceOption,
) (pulumi.StringOutput, error) {
	if svc.Build != nil && infra != nil {
		return buildServiceImage(ctx, serviceName, svc, infra, pluginID, opts...)
	}
	if svc.Image == nil {
		return pulumi.StringOutput{}, fmt.Errorf("service %s: %w", serviceName, common.ErrNoImageOrBuildConfig)
	}

	img := svc.StaticImage()
	if img == nil {
		// Dynamic (Output) image: registry can't be inspected synchronously, so
		// skip the unsupported-registry rewrite and pass it through as-is.
		return svc.Image.ToStringOutput(), nil
	}

	info := common.ParseImage(*img)
	if !isCloudRunSupportedRegistry(info.Registry) {
		gcpProject := config.GetProject(ctx)
		region := GcpRegion(ctx)
		if infra != nil {
			gcpProject = infra.GcpProject
			region = infra.Region
		}
		originalRegistry := info.Registry
		info.Registry = region + "-docker.pkg.dev"

		repo, ok := repos[originalRegistry]
		if !ok {
			return pulumi.StringOutput{}, fmt.Errorf(
				"%w %s (referenced by service %s)",
				errNoRemoteRepoConfigured, originalRegistry, serviceName,
			)
		}

		// RepositoryId is the short ID (e.g. "ghcr-io"); repo.Name is the full GCP resource path.
		image := repo.RepositoryId.ApplyT(func(repoId string) string {
			info.Repo = fmt.Sprintf("%s/%s/%s", gcpProject, repoId, info.Repo)
			msg := fmt.Sprintf("rewriting image for service %s: %s -> %s (registry not supported by Cloud Run)",
				serviceName, *img, info.FullImage())
			_ = ctx.Log.Info(msg, nil)
			return info.FullImage()
		}).(pulumi.StringOutput)
		return image, nil
	}
	return pulumi.String(info.FullImage()).ToStringOutput(), nil
}

// cloudBuildMachineType returns the Cloud Build machine type string for a given
// shm_size in bytes, using the same thresholds as the CD implementation.
func cloudBuildMachineType(shmBytes int) string {
	memMiB := shmBytes / (1024 * 1024)
	if memMiB == 0 {
		memMiB = 8192
	}
	if memMiB <= 4096 {
		return "E2_MEDIUM"
	}
	if memMiB <= 8192 {
		return "E2_HIGHCPU_8"
	}
	return "E2_HIGHCPU_32"
}

// cloudBuildDiskSizeGb returns the disk size in GB for a given shm_size in bytes
// (2× memory, minimum 16 GB).
func cloudBuildDiskSizeGb(shmBytes int) int {
	memMiB := shmBytes / (1024 * 1024)
	if memMiB == 0 {
		memMiB = 8192
	}
	gb := memMiB * 2 / 1024
	if gb < 16 {
		gb = 16
	}
	return gb
}

// buildSourceDigest computes a SHA1 over the build config fields that affect
// the build output (context path, dockerfile, target, args). It mirrors
// getCloudBuildHash from the CD implementation, using json.Marshal for args
// so that map keys are sorted deterministically. It is returned as a
// StringOutput because Context is a pulumi.StringInput.
func buildSourceDigest(build *compose.BuildConfig) pulumi.StringOutput {
	return build.Context.ToStringOutput().ApplyT(func(ctx string) (string, error) {
		h := sha1.New() //nolint:gosec
		h.Write([]byte(ctx))
		h.Write([]byte(build.GetDockerfile()))
		h.Write([]byte(build.GetTarget()))
		h.Write([]byte(strings.Join(build.Platforms, ",")))
		argBytes, err := json.Marshal(build.Args)
		if err != nil {
			return "", fmt.Errorf("marshaling build args: %w", err)
		}
		h.Write(argBytes)
		return hex.EncodeToString(h.Sum(nil)), nil
	}).(pulumi.StringOutput)
}

// resolveSourceURI asserts that the build context is already a GCS URI.
//
// Every `defang up` deploy arrives here with a gs:// URI already: the CLI archives
// the build context, uploads it to the shared Defang CD bucket with a presigned URL,
// and rewrites build.context to "gs://<cd-bucket>/uploads/<digest>.tar.gz" before CD
// runs. Those are passed through untouched.
//
// A local path only reaches here when the Pulumi program is driven directly (not by
// `defang up`) or via `compose config` dry runs (which never invoke CD). Neither is
// a supported way to build on GCP, so it's a hard error rather than an upload-on-the-
// fly fallback: there is no real caller to preserve that behavior for.
func resolveSourceURI(serviceName string, build *compose.BuildConfig) (pulumi.StringOutput, error) {
	if ps, ok := build.Context.(pulumi.String); ok && !strings.HasPrefix(string(ps), "gs://") {
		return pulumi.StringOutput{}, fmt.Errorf("%w: service %s has build context %q",
			errBuildContextNotGCS, serviceName, string(ps))
	}
	// Already a GCS URI (or an unresolved output) — use as-is.
	return build.Context.ToStringOutput(), nil
}

// buildServiceImage creates a defang-gcp:defanggcp:Build custom resource that
// runs Cloud Build and returns the resulting image URI (repo@digest).
func buildServiceImage(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	infra *BuildInfra,
	pluginID common.PluginIdentity,
	opts ...pulumi.ResourceOption,
) (pulumi.StringOutput, error) {
	dest := pulumi.Sprintf("%s/%s:latest", infra.RepositoryURL, serviceName)
	steps := generateBuildSteps(svc.Build, dest)
	shmBytes := svc.Build.GetShmSizeBytes()

	sourceURI, err := resolveSourceURI(serviceName, svc.Build)
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	// The Build resource must depend on the BucketIAMMember (objectViewer on the
	// shared CD bucket) so Pulumi waits for the IAM binding before submitting the
	// Cloud Build job. GCP IAM changes can take ~60 s to propagate globally; without
	// this explicit edge the build SA may not yet have read access when Cloud Build
	// attempts to fetch the source archive, causing a 403 on first deploy.
	buildOpts := make([]pulumi.ResourceOption, 0, len(opts)+1)
	buildOpts = append(buildOpts, opts...)
	if infra.BucketIAMMember != nil {
		buildOpts = append(buildOpts, pulumi.DependsOn([]pulumi.Resource{infra.BucketIAMMember}))
	}

	var buildRes gcpBuildResource
	if err := ctx.RegisterResource(
		"defang-gcp:defanggcp:Build",
		serviceName,
		pulumi.Map{
			"projectId":      pulumi.String(infra.GcpProject),
			"location":       pulumi.String(infra.Region),
			"source":         sourceURI,
			"sourceDigest":   buildSourceDigest(svc.Build),
			"steps":          steps,
			"images":         pulumi.StringArray{dest},
			"serviceAccount": infra.ServiceAccount.Email,
			"machineType":    pulumi.String(cloudBuildMachineType(shmBytes)),
			"diskSizeGb":     pulumi.Int(cloudBuildDiskSizeGb(shmBytes)),
		},
		&buildRes,
		pluginID.ResourceOptions(buildOpts...)...,
	); err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("creating Build resource for %s: %w", serviceName, err)
	}

	return pulumi.Sprintf("%s@%s", dest, buildRes.ImageDigest), nil
}
