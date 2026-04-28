package azure

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/authorization/v3"
	containerregistry "github.com/pulumi/pulumi-azure-native-sdk/containerregistry/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/managedidentity/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var (
	ErrBuildConfigNil       = errors.New("build config is nil")
	ErrNoImageOrBuildConfig = errors.New("no image or build config")
)

// BuildInfra holds the shared Azure Container Registry used across all builds in a project.
// Created once per project when at least one service requires a build.
type BuildInfra struct {
	registry       *containerregistry.Registry
	subscriptionID pulumi.StringOutput
	// managedIdentityID is the resource ID of the user-assigned managed identity
	// granted AcrPull on the registry. Container Apps use it for image pull auth.
	managedIdentityID pulumi.StringOutput
}

// ManagedIdentityID returns the resource ID of the AcrPull managed identity.
func (b *BuildInfra) ManagedIdentityID() pulumi.StringOutput { return b.managedIdentityID }

// LoginServer returns the ACR login server hostname (e.g. "myregistry.azurecr.io").
func (b *BuildInfra) LoginServer() pulumi.StringOutput { return b.registry.LoginServer }

// acrImageBuildResource is the Pulumi resource state for the ACRImageBuild custom resource.
// Used with ctx.RegisterResource to create the resource from within the component.
type acrImageBuildResource struct {
	pulumi.CustomResourceState
	RunID pulumi.StringOutput `pulumi:"runId"`
	Image pulumi.StringOutput `pulumi:"image"`
}

// subscriptionIDFromResourceID parses the subscription ID from an Azure resource ID.
// Azure resource IDs have the form /subscriptions/{subId}/resourceGroups/...
func subscriptionIDFromResourceID(resourceID string) string {
	parts := strings.Split(resourceID, "/")
	for i, p := range parts {
		if strings.EqualFold(p, "subscriptions") && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// sanitizeRegistryName strips all non-alphanumeric characters to satisfy the
// Azure Container Registry naming constraint: ^[a-zA-Z0-9]*$ (5–50 chars).
func sanitizeRegistryName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if len(s) > 50 {
		s = s[:50]
	}
	return s
}

// acrPullRoleDefinitionID is the built-in Azure role for pulling images from ACR.
const acrPullRoleDefinitionID = "7f951dda-4ed3-4680-a7ca-43fe172d538d"

// CreateBuildInfra creates the shared Azure Container Registry for image builds,
// a user-assigned managed identity, and an AcrPull role assignment so Container
// Apps can pull built images without admin credentials.
func CreateBuildInfra(
	ctx *pulumi.Context,
	name string,
	infra *SharedInfra,
	location string,
	opts ...pulumi.ResourceOption,
) (*BuildInfra, error) {
	// Use sanitized name as the Pulumi logical name so auto-naming produces an
	// alphanumeric base; Pulumi appends a random suffix for global uniqueness.
	registry, err := containerregistry.NewRegistry(ctx, sanitizeRegistryName(name), &containerregistry.RegistryArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		Location:          pulumi.String(location),
		Sku: &containerregistry.SkuArgs{
			Name: pulumi.String(string(containerregistry.SkuNameStandard)),
		},
		Tags: DefangTags(ctx, infra.Etag, ""),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ACR registry: %w", err)
	}

	subID := registry.ID().ToStringOutput().ApplyT(subscriptionIDFromResourceID).(pulumi.StringOutput)

	// User-assigned managed identity for Container Apps to pull built images.
	identity, err := managedidentity.NewUserAssignedIdentity(
		ctx, "acr", &managedidentity.UserAssignedIdentityArgs{
			ResourceGroupName: infra.ResourceGroup.Name,
			Location:          pulumi.String(location),
			Tags:              DefangTags(ctx, infra.Etag, ""),
		}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating managed identity: %w", err)
	}

	// Grant the identity AcrPull on the registry.
	roleDefID := pulumi.Sprintf(
		"/subscriptions/%s/providers/Microsoft.Authorization/roleDefinitions/%s",
		subID, acrPullRoleDefinitionID,
	)
	// Parent defaults to the surrounding component (via opts). Pulumi's SDK
	// discourages using custom resources as parents — destruction order here is
	// already enforced by the implicit data dependency on identity.PrincipalId.
	roleAssignment, err := authorization.NewRoleAssignment(ctx, "acr-pull", &authorization.RoleAssignmentArgs{
		Scope:            registry.ID(),
		RoleDefinitionId: roleDefID,
		PrincipalId:      identity.PrincipalId,
		PrincipalType:    pulumi.String("ServicePrincipal"),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating AcrPull role assignment: %w", err)
	}

	// Derive managedIdentityID from both the identity AND the role assignment so
	// the Container App implicitly depends on the role assignment being active
	// before it attempts to pull images from ACR.
	managedIdentityID := pulumi.All(identity.ID(), roleAssignment.ID()).ApplyT(
		func(args []interface{}) string {
			return string(args[0].(pulumi.ID))
		},
	).(pulumi.StringOutput)

	return &BuildInfra{
		registry:          registry,
		subscriptionID:    subID,
		managedIdentityID: managedIdentityID,
	}, nil
}

// imageBuildResult holds the result of building a container image.
type imageBuildResult struct {
	imageURI pulumi.StringOutput
}

// buildServiceImage builds a container image via ACR Tasks for a service.
// Creates an ACR task definition and an ACRImageBuild custom resource that triggers the run.
func buildServiceImage(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	infra *BuildInfra,
	sharedInfra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*imageBuildResult, error) {
	if svc.Build == nil {
		return nil, fmt.Errorf("service %s: %w", serviceName, ErrBuildConfigNil)
	}

	platform := svc.GetPlatform()
	imageName := serviceName

	taskYAML, err := generateTaskYAML(imageName, svc.Build.GetDockerfile(), svc.Build.Args, platform)
	if err != nil {
		return nil, fmt.Errorf("generating ACR task YAML for %s: %w", serviceName, err)
	}
	encodedYAML := base64.StdEncoding.EncodeToString([]byte(taskYAML))

	task, err := createACRTask(
		ctx,
		serviceName+"-build",
		encodedYAML,
		svc.Build.Context,
		infra.registry,
		sharedInfra,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("creating ACR task for %s: %w", serviceName, err)
	}

	triggerHash := common.BuildTriggerHash(svc.Build)

	var buildResource acrImageBuildResource
	// IgnoreChanges on contextPath: the defang CLI hands us a fresh SAS URL every
	// deploy (same blob, new signature/expiry), so a literal diff would trigger a
	// rebuild even when source is unchanged. Real source changes propagate via
	// triggers (buildTriggerHash strips the SAS before hashing).
	buildOpts := append([]pulumi.ResourceOption{pulumi.IgnoreChanges([]string{"contextPath"})}, opts...)
	err = ctx.RegisterResource("defang-azure:index:ACRImageBuild", serviceName+"-build", pulumi.Map{
		"subscriptionId":     infra.subscriptionID,
		"resourceGroupName":  sharedInfra.ResourceGroup.Name,
		"registryName":       infra.registry.Name,
		"taskName":           task.Name,
		"imageName":          pulumi.String(imageName),
		"loginServer":        infra.registry.LoginServer,
		"contextPath":        svc.Build.Context,
		"encodedTaskContent": pulumi.String(encodedYAML),
		"triggers":           pulumi.StringArray{triggerHash},
	}, &buildResource, buildOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating ACRImageBuild resource for %s: %w", serviceName, err)
	}

	return &imageBuildResult{imageURI: buildResource.Image}, nil
}

// GetServiceImage returns the container image URI for a service.
// If the service has a build config, it builds the image via ACR Tasks.
// Otherwise, it returns the pre-configured image string.
func GetServiceImage(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	buildInfra *BuildInfra,
	sharedInfra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (pulumi.StringOutput, error) {
	if svc.Build != nil && buildInfra != nil {
		result, err := buildServiceImage(ctx, serviceName, svc, buildInfra, sharedInfra, opts...)
		if err != nil {
			return pulumi.StringOutput{}, err
		}
		// TODO: Handle image push if svc.Image is specified alongside build
		return result.imageURI, nil
	}

	if svc.Image == nil {
		return pulumi.StringOutput{}, fmt.Errorf("service %s: %w", serviceName, ErrNoImageOrBuildConfig)
	}

	return pulumi.String(*svc.Image).ToStringOutput(), nil
}
