package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/DefangLabs/defang/src/pkg/cli/client"
	"github.com/DefangLabs/defang/src/pkg/clouds/scaleway"
	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"github.com/aws/smithy-go"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"google.golang.org/protobuf/proto"
)

const cdCommandScalewayScheduledDown = client.CdCommand("scaleway-scheduled-down")

const (
	scalewayLockRetryDelay = 10 * time.Second
	scalewayLockRetryLimit = 10 * time.Minute
)

var (
	errScalewayTriggerAuth = errors.New("scheduled Scaleway teardown requires a project, region, " +
		"and Secret Manager credential")
	errScalewayTriggerStateURL = errors.New("scheduled Scaleway teardown requires an s3:// DEFANG_STATE_URL")
	errScalewayCDJobDuplicate  = errors.New("multiple Scaleway CD jobs match the schedule")
	errScalewayCDJobMissing    = errors.New("could not find the Scaleway CD job")
	errScalewayCDAccessMissing = errors.New("shared Scaleway CD job is missing SCW_ACCESS_KEY")
	errScalewayStaleDeployment = errors.New("deployment changed before scheduled teardown acquired the stack lock")
)

type scalewayScheduledDown struct {
	project string
	stack   string
	jobName string
	etag    string
	key     string
}

func triggerScalewayDown(ctx context.Context, project, stack string, args []string) error {
	if len(args) != 3 || args[0] == "" || args[1] == "" || args[2] == "" || stack == "" {
		return &usageError{msg: "usage: cd scaleway-trigger-down <CD job name> <deployment ID> <state key>"}
	}
	bucket, err := scalewayStateBucket()
	if err != nil {
		return err
	}
	projectID := os.Getenv("SCW_DEFAULT_PROJECT_ID")
	region := os.Getenv("SCW_DEFAULT_REGION")
	secretKey := os.Getenv("SCW_SECRET_KEY")
	if projectID == "" || region == "" || secretKey == "" {
		return errScalewayTriggerAuth
	}
	// The Jobs API authenticates using the secret key alone. The matching
	// S3 access key is read from the current CD job in case it was rotated.
	scw := scaleway.NewClient("", secretKey, projectID, region)
	down := scalewayScheduledDown{
		project: project,
		stack:   stack,
		jobName: args[0],
		etag:    args[1],
		key:     args[2],
	}
	return down.start(ctx, scw, func(accessKey, key string) ([]byte, error) {
		storage := scaleway.NewS3Client(region, accessKey, secretKey)
		return scaleway.GetObject(ctx, storage, bucket, key)
	})
}

func (down scalewayScheduledDown) start(
	ctx context.Context, scw *scaleway.Client, getProjectUpdate func(accessKey, key string) ([]byte, error),
) error {
	// CD setup can replace the shared job definition. Resolve it by name
	// when the schedule fires, using the region and project of this stack.
	definitions, err := scw.ListJobDefinitions(ctx, down.jobName)
	if err != nil {
		return err
	}
	var job *scaleway.JobDefinition
	for i := range definitions {
		def := &definitions[i]
		if def.Name != down.jobName || def.ProjectID != scw.ProjectID {
			continue
		}
		if job != nil {
			return fmt.Errorf("%w: %q", errScalewayCDJobDuplicate, down.jobName)
		}
		job = def
	}
	if job == nil {
		return fmt.Errorf("%w: %q", errScalewayCDJobMissing, down.jobName)
	}
	accessKey := job.EnvironmentVars["SCW_ACCESS_KEY"]
	if accessKey == "" {
		return fmt.Errorf("%w: %q", errScalewayCDAccessMissing, down.jobName)
	}

	// A trigger queued before a redeploy or manual down can outlive the
	// schedule. Check the current deployment before starting a down.
	data, err := getProjectUpdate(accessKey, down.key)
	current, err := scalewayDeploymentMatches(data, err, down.etag)
	if err != nil || !current {
		return err
	}

	// The shared job has the backend config and Secret Manager references.
	// Its execution checks the deployment again after leaving the queue.
	args := []string{string(cdCommandScalewayScheduledDown), down.etag, down.key}
	run, err := scw.RunJob(ctx, job.ID, []string{cdExecutable}, args, map[string]string{
		"PROJECT": down.project,
		"STACK":   down.stack,
	})
	if err != nil {
		return err
	}
	Println("Started down execution", run.ID)
	return nil
}

func checkScalewayScheduledDown(ctx context.Context, args []string) (bool, error) {
	if len(args) != 2 || args[0] == "" || args[1] == "" {
		return false, &usageError{msg: "Scaleway scheduled teardown requires a deployment ID and state key"}
	}
	bucket, err := scalewayStateBucket()
	if err != nil {
		return false, err
	}
	scw, err := scaleway.NewClientFromEnv()
	if err != nil {
		return false, err
	}
	storage := scaleway.NewS3Client(scw.Region, scw.AccessKey, scw.SecretKey)
	data, err := scaleway.GetObject(ctx, storage, bucket, args[1])
	return scalewayDeploymentMatches(data, err, args[0])
}

func scalewayDestroyGuard(ctx context.Context, args []string) pulumi.RunFunc {
	return func(*pulumi.Context) error {
		// Pulumi runs this program while holding the stack lock. A redeploy
		// can finish after our initial check but before destroy.
		current, err := checkScalewayScheduledDown(ctx, args)
		if err != nil {
			return err
		}
		if !current {
			return errScalewayStaleDeployment
		}
		return nil
	}
}

func retryScalewayDestroy(ctx context.Context, delay time.Duration, destroy func() error) error {
	// Limit the time spent waiting for the lock while leaving the running
	// destroy under the CD job's original deadline.
	waitCtx, cancel := context.WithTimeout(ctx, scalewayLockRetryLimit)
	defer cancel()

	for {
		err := destroy()
		if !auto.IsConcurrentUpdateError(err) {
			return err
		}

		Println("Scheduled Scaleway teardown is waiting for the Pulumi stack lock")
		select {
		case <-waitCtx.Done():
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		case <-time.After(delay):
		}
	}
}

func scalewayStateBucket() (string, error) {
	stateURL, err := url.Parse(os.Getenv("DEFANG_STATE_URL"))
	if err != nil || stateURL.Scheme != "s3" || stateURL.Host == "" {
		return "", errScalewayTriggerStateURL
	}
	return stateURL.Host, nil
}

func scalewayDeploymentMatches(data []byte, err error, etag string) (bool, error) {
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NoSuchKey" || apiErr.ErrorCode() == "NoSuchBucket") {
			return false, nil
		}
		return false, err
	}
	var current defangv1.ProjectUpdate
	if err := proto.Unmarshal(data, &current); err != nil {
		return false, fmt.Errorf("reading deployment ID for Scaleway teardown: %w", err)
	}
	return current.GetEtag() == etag, nil
}
