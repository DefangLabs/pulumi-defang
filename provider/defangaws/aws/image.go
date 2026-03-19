package aws

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecr"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

// imageInfra holds shared infrastructure for building container images.
// Created once per project, shared across all services that need builds.
type imageInfra struct {
	ecrRepo       *ecr.Repository
	ecrRepoURL    pulumix.Output[string]
	codeBuildRole *iam.Role
	logGroup      *cloudwatch.LogGroup
	region        string
}

// createImageInfra creates the shared infrastructure needed for image builds:
// ECR repository and CodeBuild IAM role.
func createImageInfra(
	ctx *pulumi.Context,
	logGroup *cloudwatch.LogGroup,
	region string,
	opts ...pulumi.ResourceOption,
) (*imageInfra, error) {
	ecrResult, err := createECRRepo(ctx, "builds", opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ECR repo for builds: %w", err)
	}

	cbRole, err := createCodeBuildRole(ctx, "codebuild-role", logGroup, ecrResult.repository, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating CodeBuild role: %w", err)
	}

	return &imageInfra{
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
	imageURI pulumix.Output[string]
}

// codeBuildImageBuildResource is a Pulumi resource state for the CodeBuildImageBuild custom resource.
// Used with ctx.RegisterResource to create the resource from within the component.
type codeBuildImageBuildResource struct {
	pulumi.CustomResourceState
	BuildID pulumi.StringOutput `pulumi:"buildId"`
	Image   pulumi.StringOutput `pulumi:"image"`
}

// buildTriggerHash computes a hash of build inputs to trigger replacements when they change.
func buildTriggerHash(build shared.BuildInput) pulumix.Output[string] {
	staticHash := func(context string) string {
		h := sha256.New()
		h.Write([]byte(context))
		h.Write([]byte(build.GetDockerfile()))
		h.Write([]byte(build.GetTarget()))
		h.Write([]byte(fmt.Sprintf("%d", build.GetShmSizeBytes())))
		if len(build.Args) > 0 {
			keys := make([]string, 0, len(build.Args))
			for k := range build.Args {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			b, _ := json.Marshal(build.Args)
			h.Write(b)
		}
		return hex.EncodeToString(h.Sum(nil))[:16]
	}
	return pulumix.Apply(pulumi.String(build.Context), staticHash)
}

// buildServiceImage builds a container image via CodeBuild for a service.
// Creates a CodeBuild project and a CodeBuildImageBuild custom resource that triggers the build.
func buildServiceImage(
	ctx *pulumi.Context,
	serviceName string,
	svc shared.ServiceInput,
	infra *imageInfra,
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
	triggerHash := buildTriggerHash(*svc.Build)

	var buildResource codeBuildImageBuildResource
	err = ctx.RegisterResource("defang-aws:defangaws:CodeBuildImageBuild", serviceName+"-build", pulumi.Map{
		"projectName": cbResult.project.Name,
		"region":      pulumi.String(infra.region),
		"destination": pulumi.StringOutput(cbResult.destination),
		"triggers":    pulumi.StringArray{pulumi.StringOutput(triggerHash)},
	}, &buildResource, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating CodeBuild build resource for %s: %w", serviceName, err)
	}

	// Use the image output from the build resource (destination with potential digest)
	imageURI := pulumix.Output[string](buildResource.Image)

	return &imageBuildResult{
		imageURI: imageURI,
	}, nil
}

// getServiceImage returns the container image URI for a service.
// If the service has a build config, it builds the image via CodeBuild.
// Otherwise, it returns the pre-configured image.
// Matches TS getServiceImage in cd/aws/defang_service.ts.
func getServiceImage(
	ctx *pulumi.Context,
	serviceName string,
	svc shared.ServiceInput,
	infra *imageInfra,
	opts ...pulumi.ResourceOption,
) (pulumix.Output[string], error) {
	if svc.Build != nil && infra != nil {
		result, err := buildServiceImage(ctx, serviceName, svc, infra, opts...)
		if err != nil {
			return pulumix.Output[string]{}, err
		}
		return result.imageURI, nil
	}

	// Use pre-built image (either service.image or default)
	return pulumix.Val(svc.GetImage()), nil
}
