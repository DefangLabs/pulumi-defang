package scaleway

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// BuildInfra holds shared infrastructure for building container images on Scaleway.
// Created once per project by buildSharedInfra, shared across all services that need builds.
type BuildInfra struct {
	// RegistryEndpoint is the Scaleway Container Registry endpoint
	// (e.g., "rg.fr-par.scw.cloud/namespace")
	RegistryEndpoint pulumi.StringOutput
}

// imageBuildResource is a Pulumi resource state for the Build custom resource.
type imageBuildResource struct {
	pulumi.CustomResourceState
	BuildId pulumi.StringOutput `pulumi:"buildId"`
	Image   pulumi.StringOutput `pulumi:"image"`
}

// buildServiceImage registers a Build custom resource that triggers a Kaniko build
// via Scaleway Serverless Jobs. The resource creates a temporary job definition,
// runs the Kaniko executor to build and push the image, then cleans up.
func buildServiceImage(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	infra *BuildInfra,
	sharedInfra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (pulumi.StringOutput, error) {
	if svc.Build == nil {
		return pulumi.StringOutput{}, fmt.Errorf("service %s: build config is nil", serviceName)
	}

	// Destination image: registryEndpoint/serviceName:etag
	tag := "latest"
	if sharedInfra != nil && sharedInfra.Etag != "" {
		tag = sharedInfra.Etag
	}
	destination := infra.RegistryEndpoint.ApplyT(func(endpoint string) string {
		return fmt.Sprintf("%s/%s:%s", endpoint, serviceName, tag)
	}).(pulumi.StringOutput)

	triggerHash := common.BuildTriggerHash(svc.Build)

	var buildResource imageBuildResource
	err := ctx.RegisterResource("defang-scaleway:index:Build", serviceName, pulumi.Map{
		"region":      pulumi.String(sharedInfra.Region),
		"projectId":   pulumi.String(sharedInfra.ProjectID),
		"source":      svc.Build.Context.ToStringOutput(),
		"destination": destination,
		"dockerfile":  pulumi.StringPtrFromPtr(svc.Build.Dockerfile),
		"target":      pulumi.StringPtrFromPtr(svc.Build.Target),
		"buildArgs":   pulumi.ToStringMap(svc.Build.Args),
		"triggers":    pulumi.StringArray{triggerHash},
	}, &buildResource, opts...)
	if err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("creating build resource for %s: %w", serviceName, err)
	}

	return buildResource.Image, nil
}

// GetServiceImage returns the container image URI for a service.
// If the service has a build config and build infrastructure is available,
// it triggers a Kaniko build via Scaleway Serverless Jobs.
// Otherwise, it returns the pre-configured image.
func GetServiceImage(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	buildInfra *BuildInfra,
	sharedInfra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (pulumi.StringOutput, error) {
	if svc.Build != nil && buildInfra != nil {
		return buildServiceImage(ctx, serviceName, svc, buildInfra, sharedInfra, opts...)
	}

	if svc.Image == nil {
		return pulumi.StringOutput{}, fmt.Errorf("service %s: %w", serviceName, common.ErrNoImageOrBuildConfig)
	}

	return pulumi.String(*svc.Image).ToStringOutput(), nil
}
