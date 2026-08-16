package program

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	armappcontainers "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v3"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providerazure "github.com/DefangLabs/pulumi-defang/provider/defangazure/azure"
	"github.com/pulumi/pulumi-azure-native-sdk/app/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/v3/commontypesv5"
	azconfig "github.com/pulumi/pulumi-azure-native-sdk/v3/config"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	// The shared CD resource group and job the CLI provisions once per
	// subscription (see defang/src/pkg/clouds/azure: cd/driver.go and
	// aca/job.go). Their names are fixed by the CLI's conventions.
	cdResourceGroup = "defang-cd"
	cdJobName       = "defang-cd"

	// selfDestructJobName is deterministic so every redeploy updates the same
	// trigger in place (extending the stack's life to now + TTL). It is unique
	// within the project resource group, which is itself per project/stack.
	selfDestructJobName = "defang-self-destruct"

	// selfDestructTimeout caps the scheduled down run, mirroring the 30-minute
	// timeout the CLI applies to CD job executions (ByocAzure.runCdCommand).
	selfDestructTimeout = 30 * time.Minute

	// Replica sizing must be repeated on the trigger job: the CD job template
	// keeps the platform default (0.25 vCPU / 0.5 GiB) because the CLI sizes
	// each execution via start-time overrides — which a schedule-triggered
	// job cannot carry. Values mirror cdJobCPU/cdJobMemory in the CLI.
	selfDestructCPU    = 2.0
	selfDestructMemory = "4Gi"
)

// createAzureSelfDestruct schedules this stack's own `defang cd down`: an ACA
// Job in the project resource group with a Schedule trigger pinned to
// now + ttl, running the CD image with args ["down"] and (a filtered copy of)
// this run's environment. The job is a stack resource, so the down it starts
// destroys it along with everything else; failing to create it fails the
// deploy — a stack that silently outlives its requested TTL is the exact
// failure mode this feature exists to prevent.
func createAzureSelfDestruct(pctx *pulumi.Context, cf *compose.Project, ttl time.Duration, dep pulumi.Resource, opts ...pulumi.ResourceOption) error {
	subscriptionID := azconfig.GetSubscriptionId(pctx)
	if subscriptionID == "" {
		return fmt.Errorf("defang:ttl is set but AZURE_SUBSCRIPTION_ID is not in Pulumi config")
	}

	// Read the shared CD job's template: it carries the CD image, the managed
	// identity the down run must assume, and the Container Apps environment
	// (and thus the location) the trigger job must join. Reading it at deploy
	// time keeps the CLI out of the loop — no new env vars to pass.
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return fmt.Errorf("self-destruct: build credential: %w", err)
	}
	client, err := armappcontainers.NewJobsClient(subscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("self-destruct: build jobs client: %w", err)
	}
	resp, err := client.Get(pctx.Context(), cdResourceGroup, cdJobName, nil)
	if err != nil {
		return fmt.Errorf("self-destruct: read CD job %s/%s: %w", cdResourceGroup, cdJobName, err)
	}

	fireAt := time.Now().Add(ttl)
	args, err := azureSelfDestructJobArgs(resp.Job, os.Environ(), fireAt, providerazure.ProjectResourceGroupName(pctx, cf.Name), pctx.Stack())
	if err != nil {
		return fmt.Errorf("self-destruct: %w", err)
	}

	_ = pctx.Log.Info(fmt.Sprintf("self-destruct: this stack will run `defang cd down` on itself at %s (ttl %s); redeploying extends it",
		fireAt.UTC().Format(time.RFC3339), ttl), nil)

	_, err = app.NewJob(pctx, selfDestructJobName, args,
		append(opts, pulumi.DependsOn([]pulumi.Resource{dep}))...)
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

	identity, err := azureSelfDestructIdentity(cdJob.Identity)
	if err != nil {
		return nil, err
	}

	env := selfDestructEnv(environ)
	var envVars app.EnvironmentVarArray
	var secrets app.SecretArray
	for _, k := range sortedKeys(env) {
		if isSensitiveEnv(k) {
			// ACA job secrets keep the value out of plain-text template reads.
			secretName := strings.ToLower(strings.ReplaceAll(k, "_", "-"))
			secrets = append(secrets, app.SecretArgs{
				Name:  pulumi.String(secretName),
				Value: pulumi.String(env[k]),
			})
			envVars = append(envVars, app.EnvironmentVarArgs{
				Name:      pulumi.String(k),
				SecretRef: pulumi.String(secretName),
			})
			continue
		}
		envVars = append(envVars, app.EnvironmentVarArgs{
			Name:  pulumi.String(k),
			Value: pulumi.String(env[k]),
		})
	}

	args := &app.JobArgs{
		JobName:           pulumi.String(selfDestructJobName),
		ResourceGroupName: pulumi.String(resourceGroup),
		Location:          pulumi.String(*cdJob.Location), // must match the CD environment's region
		EnvironmentId:     pulumi.String(*cdJob.Properties.EnvironmentID),
		Identity:          identity,
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
			Secrets: secrets,
		},
		Template: app.JobTemplateArgs{
			Containers: app.ContainerArray{
				app.ContainerArgs{
					Name:    pulumi.String(selfDestructJobName),
					Image:   pulumi.String(image),
					Command: pulumi.ToStringArray([]string{"/app/cd"}), // matches ByocAzure.runCdCommand
					Args:    pulumi.ToStringArray([]string{"down"}),
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

// azureSelfDestructIdentity mirrors the CD job's user-assigned identity onto
// the trigger job, so the scheduled down runs with the same permissions as
// every other CD execution.
func azureSelfDestructIdentity(id *armappcontainers.ManagedServiceIdentity) (commontypesv5.ManagedServiceIdentityPtrInput, error) {
	if id == nil || len(id.UserAssignedIdentities) == 0 {
		return nil, fmt.Errorf("CD job has no user-assigned identity; run `defang up` with a recent CLI to provision it")
	}
	var ids []string
	for armID := range id.UserAssignedIdentities {
		ids = append(ids, armID)
	}
	// map iteration order is random; keep inputs deterministic
	sort.Strings(ids)
	return commontypesv5.ManagedServiceIdentityArgs{
		Type:                   pulumi.String("UserAssigned"),
		UserAssignedIdentities: pulumi.ToStringArray(ids),
	}, nil
}

// isSensitiveEnv reports whether the variable's value must be stored as an
// ACA secret rather than a plain-text template env var.
func isSensitiveEnv(name string) bool {
	return strings.Contains(name, "PASSPHRASE") || strings.Contains(name, "SECRET") || strings.Contains(name, "TOKEN") || strings.Contains(name, "PASSWORD")
}
