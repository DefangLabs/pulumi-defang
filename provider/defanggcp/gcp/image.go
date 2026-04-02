package gcp

import (
	"crypto/sha1" //nolint:gosec
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"gopkg.in/yaml.v3"
)

var errNoImageOrBuildConfig = errors.New("no image or build config")

// imgRE parses a Docker image reference into registry, repo, tag, and digest parts.
// Mirrors the regex used by DefangLabs/defang/src/pkg/dockerhub.
//
//nolint:lll
var imgRE = regexp.MustCompile(`^((?:((?:[0-9a-z](?:[0-9a-z-]{0,61}[0-9a-z])?\.)+[a-z]{2,63})\/)?(.{1,127}?))(?::(\w[\w.-]{0,127}))?(?:@(sha256:[0-9a-f]{64}))?$`)

type imageInfo struct {
	registry string
	repo     string
	tag      string
	digest   string
}

func parseImage(image string) imageInfo {
	m := imgRE.FindStringSubmatch(image)
	if m == nil {
		return imageInfo{repo: image}
	}
	return imageInfo{registry: m[2], repo: m[3], tag: m[4], digest: m[5]}
}

func (i imageInfo) fullImage() string {
	s := i.repo
	if i.registry != "" {
		s = i.registry + "/" + s
	}
	if i.tag != "" {
		s += ":" + i.tag
	}
	if i.digest != "" {
		s += "@" + i.digest
	}
	return s
}

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

// sanitizeRepoName produces a valid Artifact Registry repository ID:
// lowercase alphanumeric + hyphens, max 63 characters.
var nonAlphaNumRE = regexp.MustCompile(`[^a-z0-9-]`)

func sanitizeRepoName(name string) string {
	name = strings.ToLower(name)
	name = nonAlphaNumRE.ReplaceAllLiteralString(name, "-")
	name = strings.Trim(name, "-")
	if len(name) > 63 {
		name = name[:63]
	}
	return strings.TrimRight(name, "-")
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
}

// generateBuildSteps returns a YAML-encoded Cloud Build step list that builds
// a Docker image and loads it into the local Docker daemon. Cloud Build then
// pushes the image to the registry via the images: field so that build results
// contain the image digest.
func generateBuildSteps(dest pulumi.StringOutput) pulumi.StringOutput {
	return dest.ApplyT(func(d string) (string, error) {
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
				Args: []string{
					"buildx", "build", "--platform", "linux/amd64",
					"-t", d, "--load", ".",
				},
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
	infra *BuildInfra,
	opts ...pulumi.ResourceOption,
) (pulumi.StringInput, error) {
	if svc.Build != nil && infra != nil {
		return buildServiceImage(ctx, serviceName, svc, infra, opts...)
	}
	if svc.Image == nil {
		return nil, fmt.Errorf("service %s: %w", serviceName, errNoImageOrBuildConfig)
	}

	info := parseImage(*svc.Image)
	if !isCloudRunSupportedRegistry(info.registry) {
		gcpProject := gcpProjectId(ctx)
		region := GcpRegion(ctx)
		if infra != nil {
			gcpProject = infra.GcpProject
			region = infra.Region
		}
		originalRegistry := info.registry
		info.registry = region + "-docker.pkg.dev"
		info.repo = fmt.Sprintf("%s/%s/%s", gcpProject, sanitizeRepoName(originalRegistry), info.repo)
		msg := fmt.Sprintf("rewriting image for service %s: %s -> %s (registry not supported by Cloud Run)",
			serviceName, *svc.Image, info.fullImage())
		_ = ctx.Log.Info(msg, nil)
	}
	return pulumi.String(info.fullImage()), nil
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
		argBytes, err := json.Marshal(build.Args)
		if err != nil {
			return "", fmt.Errorf("marshaling build args: %w", err)
		}
		h.Write(argBytes)
		return hex.EncodeToString(h.Sum(nil)), nil
	}).(pulumi.StringOutput)
}

// resolveSourceURI ensures the build context is available at a GCS URI.
// If the context is already a gs:// URI it is returned as-is. If it is a local
// path (expressed as a pulumi.String literal), the directory is archived and
// uploaded to the project's build-artifacts bucket via a BucketObject.
func resolveSourceURI(
	ctx *pulumi.Context,
	serviceName string,
	build *compose.BuildConfig,
	infra *BuildInfra,
	opts ...pulumi.ResourceOption,
) (pulumi.StringInput, error) {
	ps, ok := build.Context.(pulumi.String)
	if ok && !strings.HasPrefix(string(ps), "gs://") {
		obj, err := storage.NewBucketObject(ctx, serviceName+"-context", &storage.BucketObjectArgs{
			Bucket: infra.BuildBucket.Name,
			Name:   pulumi.Sprintf("%s-context.zip", serviceName),
			Source: pulumi.NewFileArchive(string(ps)),
		}, opts...)
		if err != nil {
			return nil, fmt.Errorf("uploading build context for %s: %w", serviceName, err)
		}
		return pulumi.Sprintf("gs://%s/%s", infra.BuildBucket.Name, obj.Name), nil
	}
	// Already a GCS URI (or an unresolved output) — use as-is.
	return build.Context, nil
}

// buildServiceImage creates a defang-gcp:defanggcp:Build custom resource that
// runs Cloud Build and returns the resulting image URI (repo@digest).
func buildServiceImage(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	infra *BuildInfra,
	opts ...pulumi.ResourceOption,
) (pulumi.StringInput, error) {
	dest := pulumi.Sprintf("%s/%s:latest", infra.RepositoryURL, serviceName)
	steps := generateBuildSteps(dest)
	shmBytes := svc.Build.GetShmSizeBytes()

	sourceURI, err := resolveSourceURI(ctx, serviceName, svc.Build, infra, opts...)
	if err != nil {
		return nil, err
	}

	var buildRes gcpBuildResource
	if err := ctx.RegisterResource(
		"defang-gcp:defanggcp:Build",
		serviceName+"-build",
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
		opts...,
	); err != nil {
		return nil, fmt.Errorf("creating Build resource for %s: %w", serviceName, err)
	}

	return pulumi.Sprintf("%s@%s", dest, buildRes.ImageDigest), nil
}
