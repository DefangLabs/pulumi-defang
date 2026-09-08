package azure

import (
	"encoding/base64"
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-azure-native-sdk/authorization/v3"
	containerregistry "github.com/pulumi/pulumi-azure-native-sdk/containerregistry/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"gopkg.in/yaml.v3"
)

// acrDeleteRoleDefinitionID is the built-in Azure role that can delete
// repositories, manifests and tags in a Container Registry — narrower than
// the AcrPull role Container Apps use (acrPullRoleDefinitionID in image.go),
// so a leaked pull credential can never delete anything.
const acrDeleteRoleDefinitionID = "c2f4ef07-c644-48eb-af81-4b1b4947fb11"

// acrCliImage is Microsoft's own purge tool, invoked as a task "cmd" step
// (not a "build" step: there's nothing to build, and a cmd step's image
// isn't tracked by a base-image-update trigger the way a build step's is).
const acrCliImage = "mcr.microsoft.com/acr/acr-cli:0.19"

type acrPurgeTaskStep struct {
	Cmd                             string `yaml:"cmd"`
	DisableWorkingDirectoryOverride bool   `yaml:"disableWorkingDirectoryOverride"`
	Timeout                         int    `yaml:"timeout"`
}

type acrPurgeTaskSpec struct {
	Version string             `yaml:"version"`
	Steps   []acrPurgeTaskStep `yaml:"steps"`
}

// purgeTaskYAML is the ACR task recipe for a scheduled purge of repo, mirroring
// the retention shape common.KeepBuildImages/common.ExpireUntaggedDays applied
// on AWS (ecr_policy.go) and GCP (registry_policy.go): reclaim untagged
// manifests quickly, then bound total count regardless of tag state.
func purgeTaskYAML(repo string) (string, error) {
	filter := repo + ":.*"
	spec := acrPurgeTaskSpec{
		Version: "v1.1.0",
		Steps: []acrPurgeTaskStep{
			{
				Cmd: fmt.Sprintf("%s purge --filter %q --ago %dd --untagged",
					acrCliImage, filter, common.ExpireUntaggedDays),
				DisableWorkingDirectoryOverride: true,
				Timeout:                         3600,
			},
			{
				Cmd: fmt.Sprintf("%s purge --filter %q --ago 0d --keep %d",
					acrCliImage, filter, common.KeepBuildImages),
				DisableWorkingDirectoryOverride: true,
				Timeout:                         3600,
			},
		},
	}
	out, err := yaml.Marshal(spec)
	return string(out), err
}

// createRegistryPurgeTask creates a daily-scheduled ACR task that purges old
// images from repo, plus the identity it needs to do so.
//
// ACR's native retention policy (Policies.RetentionPolicy on the Registry
// resource) is a Premium-SKU-only feature; RegistrySku defaults to "Basic"
// (recipe.go) and this provider does not require Premium. A scheduled purge
// task is Microsoft's documented alternative on Basic/Standard, so that's
// what this builds instead of a native policy.
//
// The task runs under its own system-assigned identity granted AcrDelete —
// not the AcrPull identity CreateBuildInfra hands to Container Apps — so a
// pull-scoped credential leak can never delete anything.
func createRegistryPurgeTask(
	ctx *pulumi.Context,
	registry *containerregistry.Registry,
	subID pulumi.StringOutput,
	resourceGroupName pulumi.StringInput,
	repo string,
	opts ...pulumi.ResourceOption,
) error {
	taskYAML, err := purgeTaskYAML(repo)
	if err != nil {
		return fmt.Errorf("generating ACR purge task YAML: %w", err)
	}

	task, err := containerregistry.NewTask(ctx, "registry-purge", &containerregistry.TaskArgs{
		ResourceGroupName: resourceGroupName,
		RegistryName:      registry.Name,
		Identity: &containerregistry.IdentityPropertiesArgs{
			Type: containerregistry.ResourceIdentityTypePtr("SystemAssigned"),
		},
		Platform: &containerregistry.PlatformPropertiesArgs{
			Os: pulumi.String("Linux"),
		},
		Step: containerregistry.EncodedTaskStepArgs{
			Type:               pulumi.String("EncodedTask"),
			EncodedTaskContent: pulumi.String(base64.StdEncoding.EncodeToString([]byte(taskYAML))),
		},
		AgentConfiguration: &containerregistry.AgentPropertiesArgs{
			Cpu: pulumi.Int(2),
		},
		Timeout: pulumi.Int(3600),
		Trigger: &containerregistry.TriggerPropertiesArgs{
			TimerTriggers: containerregistry.TimerTriggerArray{
				&containerregistry.TimerTriggerArgs{
					Name:     pulumi.String("daily"),
					Schedule: pulumi.String("0 3 * * *"),
				},
			},
		},
		Tags: ServiceTags("registry-purge"),
	}, opts...)
	if err != nil {
		return fmt.Errorf("creating ACR purge task: %w", err)
	}

	roleDefID := pulumi.Sprintf(
		"/subscriptions/%s/providers/Microsoft.Authorization/roleDefinitions/%s",
		subID, acrDeleteRoleDefinitionID,
	)
	if _, err := authorization.NewRoleAssignment(ctx, "acr-purge-delete", &authorization.RoleAssignmentArgs{
		Scope:            registry.ID(),
		RoleDefinitionId: roleDefID,
		PrincipalId:      task.Identity.PrincipalId().Elem(),
		PrincipalType:    pulumi.String("ServicePrincipal"),
	}, opts...); err != nil {
		return fmt.Errorf("creating AcrDelete role assignment for purge task: %w", err)
	}

	return nil
}
