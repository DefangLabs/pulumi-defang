package program

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	armappcontainers "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v3"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providerazure "github.com/DefangLabs/pulumi-defang/provider/defangazure/azure"
	"github.com/pulumi/pulumi-azure-native-sdk/app/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/authorization/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/v3/commontypesv5"
	azconfig "github.com/pulumi/pulumi-azure-native-sdk/v3/config"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	// The shared CD resource group and job the CLI provisions once per
	// subscription (see defang/src/pkg/clouds/azure: cd/driver.go and
	// aca/job.go). Their names are fixed by the CLI's conventions.
	CdResourceGroup = "defang-cd"
	cdJobName       = "defang-cd"

	// selfDestructName is the Pulumi logical name and the container name inside
	// the job. It is deliberately NOT set as the job's physical JobName: Container
	// Apps requires job names to be unique within the MANAGED ENVIRONMENT, and every
	// project's trigger job is created in the shared defang-cd environment. A fixed
	// physical name therefore let only the first project in a subscription have a
	// TTL; every later one failed the whole deploy with HTTP 409
	// ContainerAppsJobNameConflictInCluster. Letting Pulumi auto-name keeps it unique
	// per stack while staying stable across redeploys of the same stack, so a
	// redeploy still updates the trigger in place and extends the stack's life.
	selfDestructName = "defang-self-destruct"

	// The trigger execution only STARTS the down (one ARM call); the down
	// itself runs in the shared defang-cd job. It must not run the destroy
	// in-place: the destroy deletes the trigger job early (it depends on the
	// project resource group), which would kill its own running execution.
	// Minimal sizing and a short timeout suffice for the one call.
	selfDestructCPU     = 0.25
	selfDestructMemory  = "0.5Gi"
	selfDestructTimeout = 10 * time.Minute

	// Built-in Contributor role, assigned to the trigger job's own
	// system-assigned identity, scoped to the defang-cd job only — just
	// enough to start a down execution. The CD job's identity cannot be
	// reused: it is system-assigned (subscription-wide Contributor + User
	// Access Administrator, see aca.SetUpManagedIdentity in the CLI) and a
	// system-assigned identity is bound to its own resource.
	contributorRoleDefinitionID = "b24988ac-6180-42a0-ab88-20f7382dd24c"
)

// createAzureSelfDestruct schedules this stack's own `defang cd down`: an ACA
// Job in the project resource group with a Schedule trigger pinned to
// now + ttl, running this CD image with args ["trigger-down"], which starts a
// regular down execution on the shared defang-cd job. All pieces (job, role
// assignment) are ordinary Pulumi resources in the stack state, so an
// explicit down — or the scheduled one — cleans them up, and dropping the TTL
// on a redeploy deletes them. Failing to create them fails the deploy: a
// stack that silently outlives its requested TTL is the exact failure mode
// this feature exists to prevent.
func createAzureSelfDestruct(pctx *pulumi.Context, cf *compose.Project, ttl time.Duration, dep pulumi.Resource, opts ...pulumi.ResourceOption) error {
	subscriptionID := azconfig.GetSubscriptionId(pctx)
	if subscriptionID == "" {
		return fmt.Errorf("defang:ttl is set but AZURE_SUBSCRIPTION_ID is not in Pulumi config")
	}

	// Read the shared CD job: it carries the CD image and the Container Apps
	// environment (and thus the location) the trigger job must join. Reading
	// it at deploy time keeps the CLI out of the loop — no new env vars.
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return fmt.Errorf("self-destruct: build credential: %w", err)
	}
	client, err := armappcontainers.NewJobsClient(subscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("self-destruct: build jobs client: %w", err)
	}
	resp, err := client.Get(pctx.Context(), CdResourceGroup, cdJobName, nil)
	if err != nil {
		return fmt.Errorf("self-destruct: read CD job %s/%s: %w", CdResourceGroup, cdJobName, err)
	}
	if resp.ID == nil {
		return fmt.Errorf("self-destruct: CD job has no resource id")
	}

	// Round up to the next whole minute: the cron expression has minute
	// granularity, and truncating would fire up to 59s before now + TTL.
	fireAt := time.Now().Add(ttl).Truncate(time.Minute).Add(time.Minute)
	args, err := azureSelfDestructJobArgs(resp.Job, os.Environ(), fireAt, providerazure.ProjectResourceGroupName(pctx, cf.Name), pctx.Stack())
	if err != nil {
		return fmt.Errorf("self-destruct: %w", err)
	}

	_ = pctx.Log.Info(fmt.Sprintf("self-destruct: this stack will run `defang cd down` on itself at %s (ttl %s); redeploying extends it",
		fireAt.UTC().Format(time.RFC3339), ttl), nil)

	job, err := app.NewJob(pctx, selfDestructName, args,
		append(opts, pulumi.DependsOn([]pulumi.Resource{dep}))...)
	if err != nil {
		return err
	}

	// Let the trigger's identity start executions on the shared CD job. The
	// assignment is created now (by the CD's own subscription-wide identity,
	// which holds User Access Administrator) and used only at fire time, so
	// AAD propagation delays are moot.
	_, err = authorization.NewRoleAssignment(pctx, "self-destruct-starter", &authorization.RoleAssignmentArgs{
		PrincipalId:      job.Identity.PrincipalId().Elem(),
		PrincipalType:    pulumi.String("ServicePrincipal"),
		RoleDefinitionId: pulumi.String(fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Authorization/roleDefinitions/%s", subscriptionID, contributorRoleDefinitionID)),
		Scope:            pulumi.String(*resp.ID),
	}, opts...)
	return err
}

// azureSelfDestructJobArgs derives the trigger job's inputs from the shared
// CD job and the current process environment. Pure, for testing.
func azureSelfDestructJobArgs(cdJob armappcontainers.Job, environ []string, fireAt time.Time, resourceGroup, stack string) (*app.JobArgs, error) {
	if cdJob.Properties == nil || cdJob.Properties.EnvironmentID == nil {
		return nil, fmt.Errorf("CD job has no environment id")
	}
	if cdJob.Location == nil {
		return nil, fmt.Errorf("CD job has no location")
	}
	if cdJob.Properties.Template == nil || len(cdJob.Properties.Template.Containers) == 0 || cdJob.Properties.Template.Containers[0].Image == nil {
		return nil, fmt.Errorf("CD job template has no container image")
	}
	image := *cdJob.Properties.Template.Containers[0].Image

	env := SelfDestructEnv(environ)
	// trigger-down re-runs this image on the defang-cd job; tell it which.
	env["DEFANG_CD_IMAGE"] = image
	// All values ride as plain env vars — exactly how the CLI passes them to
	// every CD execution today (see the "TODO: make secret" in
	// ByocAzure.environment). Real credentials never reach this point: the
	// SelfDestructEnv exclusion list drops them, and the scheduled run
	// authenticates with the trigger job's own managed identity. When the CLI
	// grows a real secret channel, it should hand the CD an explicit list of
	// secret names to mirror here — not a name heuristic.
	var envVars app.EnvironmentVarArray
	// Sorted for deterministic inputs: an unordered env list diffs on every deploy.
	for _, k := range slices.Sorted(maps.Keys(env)) {
		envVars = append(envVars, app.EnvironmentVarArgs{
			Name:  pulumi.String(k),
			Value: pulumi.String(env[k]),
		})
	}

	args := &app.JobArgs{
		// No JobName: Pulumi auto-names it, keeping it unique within the shared
		// defang-cd managed environment. See selfDestructName above.
		ResourceGroupName: pulumi.String(resourceGroup),
		Location:          pulumi.String(*cdJob.Location), // must match the CD environment's region
		EnvironmentId:     pulumi.String(*cdJob.Properties.EnvironmentID),
		Identity: commontypesv5.ManagedServiceIdentityArgs{
			Type: pulumi.String("SystemAssigned"),
		},
		Tags: pulumi.StringMap{
			"defang-fire-at": pulumi.String(fireAt.UTC().Format(time.RFC3339)),
			"defang-stack":   pulumi.String(stack),
		},
		Configuration: app.JobConfigurationArgs{
			TriggerType:       pulumi.String("Schedule"),
			ReplicaTimeout:    pulumi.Int(int(selfDestructTimeout.Seconds())),
			ReplicaRetryLimit: pulumi.Int(1),
			ScheduleTriggerConfig: app.JobConfigurationScheduleTriggerConfigArgs{
				CronExpression:         pulumi.String(selfDestructCron(fireAt)),
				Parallelism:            pulumi.Int(1),
				ReplicaCompletionCount: pulumi.Int(1),
			},
		},
		Template: app.JobTemplateArgs{
			Containers: app.ContainerArray{
				app.ContainerArgs{
					Name:    pulumi.String(selfDestructName),
					Image:   pulumi.String(image),
					Command: pulumi.ToStringArray([]string{"/app/cd"}),
					Args:    pulumi.ToStringArray([]string{"trigger-down"}),
					Env:     envVars,
					Resources: app.ContainerResourcesArgs{
						Cpu:    pulumi.Float64(selfDestructCPU),
						Memory: pulumi.String(selfDestructMemory),
					},
				},
			},
		},
	}
	if wp := cdJob.Properties.WorkloadProfileName; wp != nil {
		args.WorkloadProfileName = pulumi.StringPtr(*wp)
	}
	return args, nil
}
