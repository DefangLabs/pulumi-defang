package gcp

import (
	"crypto/sha1" //nolint:gosec
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"gopkg.in/yaml.v3"
)

var errNoImageOrBuildConfig = errors.New("no image or build config")

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
// and pushes a Docker image to dest using buildx.
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
					"-t", d, "--push", ".",
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
	return pulumi.String(*svc.Image), nil
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

	var buildRes gcpBuildResource
	if err := ctx.RegisterResource(
		"defang-gcp:defanggcp:Build",
		serviceName+"-build",
		pulumi.Map{
			"projectId":      pulumi.String(infra.GcpProject),
			"location":       pulumi.String(infra.Region),
			"source":         svc.Build.Context,
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
