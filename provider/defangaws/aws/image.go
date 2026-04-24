package aws

import (
	"errors"
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecr"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

var (
	ErrBuildConfigNil       = errors.New("build config is nil")
	ErrNoImageOrBuildConfig = errors.New("no image or build config")
)

// BuildInfra holds shared infrastructure for building container images.
// Created once per project, shared across all services that need builds.
type BuildInfra struct {
	codeBuildRole *iam.Role
	ecrRepo       *ecr.Repository
	ecrRepoURL    pulumix.Output[string]
	logGroup      *cloudwatch.LogGroup
	profile       string
	region        string
}

// CreateBuildInfra creates the shared infrastructure needed for image builds:
// ECR repository and CodeBuild IAM role.
func CreateBuildInfra(
	ctx *pulumi.Context,
	logGroup *cloudwatch.LogGroup,
	profile, region string,
	opts ...pulumi.ResourceOption,
) (*BuildInfra, error) {
	ecrResult, err := createECRRepo(ctx, "build", opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ECR repo for build: %w", err)
	}

	cbRole, err := createCodeBuildRole(ctx, "codebuild-role", logGroup, ecrResult.repository, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating CodeBuild role: %w", err)
	}

	return &BuildInfra{
		ecrRepo:       ecrResult.repository,
		ecrRepoURL:    ecrResult.repoURL,
		codeBuildRole: cbRole,
		logGroup:      logGroup,
		profile:       profile,
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

// codeBuildImageBuildResource is a Pulumi resource state for the Build custom resource.
// Used with ctx.RegisterResource to create the resource from within the component.
type codeBuildImageBuildResource struct {
	pulumi.CustomResourceState
	BuildID pulumi.StringOutput `pulumi:"buildId"`
	Image   pulumi.StringOutput `pulumi:"image"`
}

// buildServiceImage builds a container image via CodeBuild for a service.
// Creates a CodeBuild project and a Build custom resource that triggers the build.
func buildServiceImage(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	infra *BuildInfra,
	opts ...pulumi.ResourceOption,
) (*imageBuildResult, error) {
	if svc.Build == nil {
		return nil, fmt.Errorf("service %s: %w", serviceName, ErrBuildConfigNil)
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

	// Create the Build custom resource to trigger the actual build.
	// This resource calls the AWS SDK to start the build, polls until done, and returns the image URL.
	// If Create fails, the resource is not in state → next `pulumi up` retries automatically.
	triggerHash := common.BuildTriggerHash(svc.Build)

	// region, err := aws.GetRegion(ctx, &aws.GetRegionArgs{}, opts...)
	// if err != nil {
	// 	return nil, fmt.Errorf("getting AWS region: %w", err)
	// }
	// region.Region

	var buildResource codeBuildImageBuildResource
	err = ctx.RegisterResource("defang-aws:index:Build", serviceName+"-build", pulumi.Map{
		"projectName": cbResult.project.Name,
		"profile":     pulumi.String(infra.profile),
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
	infra *BuildInfra,
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
		return pulumi.StringOutput{}, fmt.Errorf("service %s: %w", serviceName, ErrNoImageOrBuildConfig)
	}

	// Use pre-built image (either service.image or default)
	return pulumi.String(*svc.Image).ToStringOutput(), nil
}
