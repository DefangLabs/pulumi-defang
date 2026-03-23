package aws

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecr"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

// ImageInfra holds shared infrastructure for building container images.
// Created once per project, shared across all services that need builds.
type ImageInfra struct {
	ecrRepo       *ecr.Repository
	ecrRepoURL    pulumix.Output[string]
	codeBuildRole *iam.Role
	logGroup      *cloudwatch.LogGroup
	region        string
}

// CreateImageInfra creates the shared infrastructure needed for image builds:
// ECR repository and CodeBuild IAM role.
func CreateImageInfra(
	ctx *pulumi.Context,
	logGroup *cloudwatch.LogGroup,
	region string,
	opts ...pulumi.ResourceOption,
) (*ImageInfra, error) {
	ecrResult, err := createECRRepo(ctx, "builds", opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ECR repo for builds: %w", err)
	}

	cbRole, err := createCodeBuildRole(ctx, "codebuild-role", logGroup, ecrResult.repository, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating CodeBuild role: %w", err)
	}

	return &ImageInfra{
		ecrRepo:       ecrResult.repository,
		ecrRepoURL:    ecrResult.repoURL,
		codeBuildRole: cbRole,
		logGroup:      logGroup,
		region:        region,
	}, nil
}

// imageBuildResult holds the result of building a container image.
type imageBuildResult struct {
	// imageURI is the ECR image URI to use in task definitions.
	// For built images: "123.dkr.ecr.region.amazonaws.com/repo:tag"
	// For pre-built images: the original image string.
	imageURI pulumi.StringOutput
}

// codeBuildImageBuildResource is a Pulumi resource state for the CodeBuildImageBuild custom resource.
// Used with ctx.RegisterResource to create the resource from within the component.
type codeBuildImageBuildResource struct {
	pulumi.CustomResourceState
	BuildID pulumi.StringOutput `pulumi:"buildId"`
	Image   pulumi.StringOutput `pulumi:"image"`
}

func isEphemeralBuildArg(key string) bool {
	return strings.HasSuffix(key, "_TOKEN")
}

// Hide ephemeral build args (eg. GITHUB_TOKEN) so we get the same imageTag each CI run
func removeEphemeralBuildArgs(args map[string]string) map[string]string {
	args = maps.Clone(args) // shallow clone
	for key := range args {
		if isEphemeralBuildArg(key) {
			args[key] = "Removed ephemeral token"
		}
	}
	return args
}

func sha1hash(inputs ...string) string {
	h := sha256.New()
	for _, c := range inputs {
		h.Write([]byte(c))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// buildTriggerHash computes a hash of build inputs to trigger replacements when they change.
func buildTriggerHash(build *compose.BuildConfig) pulumi.StringOutput {
	// Must also hash buildArgs, in case tarball is the same; stably serialize to a string
	argsStr, err := json.Marshal(removeEphemeralBuildArgs(build.Args))
	if err != nil {
		return pulumi.StringOutput{}
	}
	var dockerfile, target string
	if build.Dockerfile != nil {
		dockerfile = *build.Dockerfile
	}
	if build.Target != nil {
		target = *build.Target
	}
	return pulumi.StringOutput(pulumix.Apply(
		pulumix.Output[string](build.Context.ToStringOutput()), func(ctx string) string {
		contextEtag, _, _ := strings.Cut(ctx, "?") // remove sig query param; FIXME: get actual etag from URL, not path
		return sha1hash(contextEtag, string(argsStr), dockerfile, target)[0:8]
	}))
}

// buildServiceImage builds a container image via CodeBuild for a service.
// Creates a CodeBuild project and a CodeBuildImageBuild custom resource that triggers the build.
func buildServiceImage(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	infra *ImageInfra,
	opts ...pulumi.ResourceOption,
) (*imageBuildResult, error) {
	if svc.Build == nil {
		return nil, fmt.Errorf("build config is nil for service %s", serviceName)
	}

	platform := svc.GetPlatform()

	cbResult, err := createCodeBuildProject(
		ctx,
		serviceName+"-image",
		*svc.Build,
		platform,
		infra.codeBuildRole,
		infra.logGroup,
		infra.ecrRepoURL,
		infra.region,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("creating CodeBuild project for %s: %w", serviceName, err)
	}

	// Create the CodeBuildImageBuild custom resource to trigger the actual build.
	// This resource calls the AWS SDK to start the build, polls until done, and returns the image URL.
	// If Create fails, the resource is not in state → next `pulumi up` retries automatically.
	triggerHash := buildTriggerHash(svc.Build)

	var buildResource codeBuildImageBuildResource
	err = ctx.RegisterResource("defang-aws:defangaws:CodeBuildImageBuild", serviceName+"-build", pulumi.Map{
		"projectName": cbResult.project.Name,
		"region":      pulumi.String(infra.region),
		"destination": cbResult.destination,
		"triggers":    pulumi.StringArray{triggerHash},
	}, &buildResource, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating CodeBuild build resource for %s: %w", serviceName, err)
	}

	// Use the image output from the build resource (destination with potential digest)
	return &imageBuildResult{
		imageURI: buildResource.Image,
	}, nil
}

// GetServiceImage returns the container image URI for a service.
// If the service has a build config, it builds the image via CodeBuild.
// Otherwise, it returns the pre-configured image.
// Matches TS getServiceImage in cd/aws/defang_service.ts.
func GetServiceImage(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	infra *ImageInfra,
	opts ...pulumi.ResourceOption,
) (pulumi.StringOutput, error) {
	if svc.Build != nil && infra != nil {
		result, err := buildServiceImage(ctx, serviceName, svc, infra, opts...)
		if err != nil {
			return pulumi.StringOutput{}, err
		}
		return result.imageURI, nil
	}

	if svc.Image == nil {
		return pulumi.StringOutput{}, fmt.Errorf("service %s has no image or build config", serviceName)
	}

	// Use pre-built image (either service.image or default)
	return pulumi.String(*svc.Image).ToStringOutput(), nil
}
