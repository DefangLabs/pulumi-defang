package program

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	scalewayconfig "github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/config"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/job"
)

// ScalewayTriggerDownCommand starts a regular down on the shared CD job.
const ScalewayTriggerDownCommand = "scaleway-trigger-down"

var errScalewayTTLMetadata = errors.New("scheduling Scaleway teardown requires the deploy job, its Secret Manager " +
	"reference, and the deployment ID; redeploy with a Defang CLI that supports Scaleway TTL")

func createScalewaySelfDestruct(
	ctx *pulumi.Context, ttl time.Duration, etag, image string, provider pulumi.ProviderResource,
	deps []pulumi.Resource, ready pulumi.Output,
) error {
	targetJob := os.Getenv("DEFANG_SCW_CD_JOB_NAME")
	authSecretID := os.Getenv("DEFANG_SCW_CD_SECRET_ID")
	projectID := scalewayconfig.GetProjectId(ctx)
	region := scalewayconfig.GetRegion(ctx)
	stateURL, err := parseStateURL(ctx)
	if err != nil {
		return err
	}
	if err := checkCronTTL(ttl); err != nil {
		return err
	}
	if stateURL == nil || stateURL.Scheme != "s3" || stateURL.Host == "" {
		return errScalewayStateURL
	}
	if targetJob == "" || authSecretID == "" || projectID == "" || region == "" || image == "" || etag == "" {
		return errScalewayTTLMetadata
	}

	cron := selfDestructCronOf(selfDestructDeadlineAfter(ctx, ttl, ready))
	nameHash := sha256.Sum256([]byte(ctx.Project() + "/" + ctx.Stack()))
	// The scheduled job belongs to the application stack. The down runs on
	// the CD job outside that stack so deleting this resource cannot stop it.
	// Pass only the secret reference, never the key value, into stack state.
	_, err = job.NewDefinition(ctx, "defang-self-destruct", &job.DefinitionArgs{
		Name:                 pulumi.StringPtr(fmt.Sprintf("defang-ttl-%x", nameHash[:10])),
		ProjectId:            pulumi.StringPtr(projectID),
		Region:               pulumi.StringPtr(region),
		ImageUri:             pulumi.String(image),
		CpuLimit:             pulumi.Int(250),
		MemoryLimit:          pulumi.Int(512),
		LocalStorageCapacity: pulumi.Int(1000),
		Timeout:              pulumi.StringPtr("5m"),
		StartupCommands:      pulumi.ToStringArray([]string{"/app/cd"}),
		Args: pulumi.ToStringArray([]string{
			ScalewayTriggerDownCommand, targetJob, etag, projectPbKey(ctx),
		}),
		Cron: job.DefinitionCronArgs{
			Schedule: cron,
			Timezone: pulumi.String("Etc/UTC"),
		},
		RetryPolicy: job.DefinitionRetryPolicyArgs{MaxRetries: pulumi.IntPtr(3)},
		Env: pulumi.ToStringMap(map[string]string{
			"PROJECT":                ctx.Project(),
			"STACK":                  ctx.Stack(),
			"SCW_DEFAULT_PROJECT_ID": projectID,
			"SCW_DEFAULT_REGION":     region,
			"DEFANG_STATE_URL":       "s3://" + stateURL.Host,
		}),
		SecretReferences: job.DefinitionSecretReferenceArray{
			job.DefinitionSecretReferenceArgs{
				SecretId:      pulumi.String(authSecretID),
				SecretVersion: pulumi.StringPtr("latest"),
				Environment:   pulumi.StringPtr("SCW_SECRET_KEY"),
			},
		},
	}, pulumi.Provider(provider), pulumi.DependsOn(deps))
	return err
}
