package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	scheduler "cloud.google.com/go/scheduler/apiv1"
	"cloud.google.com/go/scheduler/apiv1/schedulerpb"
	"github.com/DefangLabs/defang/src/pkg/cli/client"
	"github.com/DefangLabs/pulumi-defang/cd/program"
	"github.com/googleapis/gax-go/v2/apierror"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
)

// A GCP `down` cannot always finish in one pass. Cloud Run attaches to the
// project VPC with Direct VPC egress (see buildVpcAccess in
// provider/defanggcp/gcp/cloudrun.go) and GCP holds the subnet's IP addresses
// for 1-2 hours after the service is gone. Until they are released the subnet
// delete fails with resourceInUseByAnotherResource, and the network cannot go
// before its subnet. That wait is documented GCP behaviour, not a provider bug,
// so retrying inside the destroy cannot help:
// https://docs.cloud.google.com/run/docs/configuring/vpc-direct-vpc
//
// Both this repo and the old MVP CD hit that same constraint. MVP retains the
// network, subnet, MIG instance templates, and Service Networking connection,
// then schedules its own delayed teardown of the physical subnet and network.
// This implementation instead removes those four retains: the destroy runs with
// ContinueOnError and deletes everything it can. If the only resources still
// standing are the ones known to wait on that window, the `down` reports
// success and schedules itself to run again. Pulumi then performs the remaining
// deletes from live state, in its own dependency order: no orphaned resources
// and no hand-rolled teardown here.
//
// The scheduled run carries cleanupJobEnvVar so that, once its destroy
// finally succeeds, it can delete the scheduler job that started it.

// cleanupJobEnvVar names the scheduler job in the environment of the run it
// schedules, so a successful retry can delete the job.
const cleanupJobEnvVar = "CLEAN_UP_JOB_NAME"

// cleanupJobTimeFormat stamps the job name with its creation time, which bounds
// how long the retries go on for.
const cleanupJobTimeFormat = "20060102150405"

// cleanupRetryDelay puts the first retry at the end of the window GCP needs to
// release the subnet's IP addresses. The cron then repeats every 2 hours, so a
// retry that is still too early simply tries again.
const cleanupRetryDelay = 1*time.Hour + 59*time.Minute

// cleanupDeadline bounds the retries. Past this the job deletes itself and says
// so, rather than firing every 2 hours for ever: a delete still failing a day
// later is stuck on something the window does not explain (for instance
// hashicorp/terraform-provider-google#19908), and needs a human.
const cleanupDeadline = 24 * time.Hour

// pendingTeardownTypes are the resource types whose delete is expected to fail
// until GCP releases the subnet's IP addresses. A destroy that leaves only
// these behind has done everything it can for now.
//
// The VPC and its subnet are the resources actually blocked. The other two hold
// a reference to the subnet and so can be caught by the same window: the
// servicenetworking connection races the Cloud SQL instance delete, and an
// instance template races its MIG's asynchronous deletion.
// Pulumi type tokens, as they appear in a checkpoint.
const (
	typeNetwork          = "gcp:compute/network:Network"
	typeSubnetwork       = "gcp:compute/subnetwork:Subnetwork"
	typeSvcConnection    = "gcp:servicenetworking/connection:Connection"
	typeInstanceTemplate = "gcp:compute/instanceTemplate:InstanceTemplate"
	typeStack            = "pulumi:pulumi:Stack"
	prefixProviders      = "pulumi:providers:"
)

var pendingTeardownTypes = map[string]bool{
	typeNetwork:          true,
	typeSubnetwork:       true,
	typeSvcConnection:    true,
	typeInstanceTemplate: true,
}

// deployment is the part of a stack export the retry logic reads.
type deployment struct {
	Resources []struct {
		Type string `json:"type"`
		URN  string `json:"urn"`
	} `json:"resources"`
}

// remainingTypes lists the resource types still in the stack's state, ignoring
// the stack itself and its providers, which are not cloud resources.
func remainingTypes(export []byte) ([]string, error) {
	var d deployment
	if err := json.Unmarshal(export, &d); err != nil {
		return nil, fmt.Errorf("failed to read the stack state: %w", err)
	}
	var types []string
	for _, res := range d.Resources {
		if res.Type == typeStack || strings.HasPrefix(res.Type, prefixProviders) {
			continue
		}
		types = append(types, res.Type)
	}
	return types, nil
}

// onlyPendingTeardown reports whether every resource left in the state is one
// that waits on the IP-release window. An empty list means the destroy finished,
// which is not a pending teardown.
//
// This reads the state rather than matching on the destroy's error text: the
// state is what Pulumi will act on when the retry runs, and an error string is
// not a contract.
func onlyPendingTeardown(types []string) bool {
	if len(types) == 0 {
		return false
	}
	for _, t := range types {
		if !pendingTeardownTypes[t] {
			return false
		}
	}
	return true
}

// handleGcpPendingTeardown decides what a GCP destroy failure means.
//
// It returns nil when the failure is only the IP-release window, having made
// sure a retry is scheduled — the `down` then reports success, because
// everything that could be deleted was. It returns destroyErr unchanged for
// any other failure, and for a non-GCP run.
func handleGcpPendingTeardown(ctx context.Context, stack auto.Stack, projectName, stackName string, destroyErr error) error {
	if destroyErr == nil || gcpProjectFromEnv() == "" {
		return destroyErr
	}

	export, err := stack.Export(ctx)
	if err != nil {
		warn(" ** Could not read the stack state to classify the destroy failure:", err)
		return destroyErr
	}
	types, err := remainingTypes(export.Deployment)
	if err != nil {
		warn(" **", err)
		return destroyErr
	}
	if !onlyPendingTeardown(types) {
		return destroyErr
	}

	warn(" ** The VPC cannot be deleted yet: GCP holds the subnet's IP addresses for 1-2 hours after Cloud Run releases them.")
	if err := ensureGcpCleanupScheduled(ctx, projectName, stackName); err != nil {
		// Without a retry the VPC would leak silently, so this is the failure
		// worth surfacing — not the destroy error it replaces.
		warn(" ** Failed to schedule the retry:", err)
		return destroyErr
	}
	return nil
}

// ensureGcpCleanupScheduled schedules the retry, unless this run is already a
// scheduled retry — in which case its own cron fires again, so long as the
// deadline has not passed.
func ensureGcpCleanupScheduled(ctx context.Context, projectName, stackName string) error {
	jobID := os.Getenv(cleanupJobEnvVar)
	if jobID == "" {
		return scheduleGcpCleanup(ctx, projectName, stackName)
	}

	createdAt, err := cleanupJobCreatedAt(jobID)
	if err != nil {
		return err
	}
	if age := time.Since(createdAt); age > cleanupDeadline {
		warn(fmt.Sprintf(" ** The VPC has resisted deletion for %s, which the IP-release window does not explain.", age.Round(time.Hour)))
		warn(" ** Giving up and removing the retry job; the VPC needs deleting by hand in the GCP console.")
		return deleteGcpCleanupJob(ctx, jobID)
	}
	warn(" ** Leaving the retry job in place; it will try again within 2 hours.")
	return nil
}

// finishGcpCleanup runs after a destroy that succeeded. When this run was a
// scheduled retry, the job that started it has done its work and is removed.
func finishGcpCleanup(ctx context.Context) {
	jobID := os.Getenv(cleanupJobEnvVar)
	if jobID == "" || gcpProjectFromEnv() == "" {
		return
	}
	if err := deleteGcpCleanupJob(ctx, jobID); err != nil {
		// The job is idempotent, so a leftover only costs one more no-op run.
		warn(" ** Failed to remove the retry job", jobID, "-", err)
		return
	}
	warn(" ** VPC deleted; removed the retry job", jobID)
}

// scheduleGcpCleanup creates the Cloud Scheduler job that re-runs this `down`.
func scheduleGcpCleanup(ctx context.Context, projectName, stackName string) error {
	gcpProject := gcpProjectFromEnv()
	region := gcpRegionFromEnv()
	if region == "" {
		return errors.New("missing required environment variable: GCLOUD_REGION")
	}
	cdImage := os.Getenv("DEFANG_CD_IMAGE")
	if cdImage == "" {
		return errors.New("missing required environment variable: DEFANG_CD_IMAGE")
	}

	// The scheduled build runs as the service account this run uses; Cloud
	// Build exposes it through the metadata server.
	saCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	saEmail, err := metadata.EmailWithContext(saCtx, "default")
	if err != nil {
		return fmt.Errorf("the retry needs the CD to run in Cloud Build: %w", err)
	}

	now := time.Now()
	jobID := cleanupJobID(projectName, stackName, now)
	body, err := gcpCleanupBuild(cdImage, gcpProject, saEmail, stackName, jobID, os.Environ())
	if err != nil {
		return err
	}

	client, err := scheduler.NewCloudSchedulerClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create scheduler client: %w", err)
	}
	defer client.Close()

	parent, name := schedulerJobName(gcpProject, region, jobID)
	if _, err := client.CreateJob(ctx, &schedulerpb.CreateJobRequest{
		Parent: parent,
		Job: &schedulerpb.Job{
			Name:        name,
			Description: "Defang retry of `cd down` to delete the VPC once GCP releases the subnet IPs; deletes itself once it succeeds",
			Schedule:    cleanupCron(now),
			TimeZone:    "Etc/UTC",
			Target: &schedulerpb.Job_HttpTarget{
				HttpTarget: &schedulerpb.HttpTarget{
					Uri:        fmt.Sprintf("https://cloudbuild.googleapis.com/v1/projects/%s/builds", gcpProject),
					HttpMethod: schedulerpb.HttpMethod_POST,
					Body:       body,
					Headers:    map[string]string{"Content-Type": "application/json"},
					AuthorizationHeader: &schedulerpb.HttpTarget_OauthToken{
						OauthToken: &schedulerpb.OAuthToken{
							ServiceAccountEmail: saEmail,
							Scope:               "https://www.googleapis.com/auth/cloud-platform",
						},
					},
				},
			},
		},
	}); err != nil {
		return fmt.Errorf("failed to create the retry job: %w", err)
	}
	warn(" ** Scheduled a retry in", cleanupRetryDelay.Round(time.Minute).String(), "as", jobID)
	return nil
}

// gcpCleanupBuild renders the builds.create request body that re-runs this CD
// image with `down` — the same shape as gcpSelfDestructBuild in cd/program,
// plus the job name so the retry can delete the job that started it.
func gcpCleanupBuild(cdImage, gcpProject, saEmail, stackName, jobID string, environ []string) ([]byte, error) {
	env := program.SelfDestructEnv(environ)
	env[cleanupJobEnvVar] = jobID
	envList := make([]string, 0, len(env))
	for _, k := range slices.Sorted(maps.Keys(env)) {
		envList = append(envList, k+"="+env[k])
	}
	build := map[string]any{
		"steps": []map[string]any{{
			"name": cdImage,
			"args": []string{string(client.CdCommandDown)},
			"env":  envList,
		}},
		"options": map[string]any{
			// Custom-service-account builds require an explicit logging mode.
			"logging":                 "CLOUD_LOGGING_ONLY",
			"enableStructuredLogging": true,
		},
		"timeout":        fmt.Sprintf("%ds", int(program.CdTimeout.Seconds())),
		"tags":           []string{"defang-cd", "defang-cleanup", stackName},
		"serviceAccount": fmt.Sprintf("projects/%s/serviceAccounts/%s", gcpProject, saEmail),
	}
	return json.Marshal(build)
}

// cleanupJobID names the retry job for one stack. The trailing timestamp keeps
// concurrent downs from colliding and records when the retries started.
func cleanupJobID(projectName, stackName string, now time.Time) string {
	return "defang-cleanup-" + projectName + "-" + stackName + "-" + now.UTC().Format(cleanupJobTimeFormat)
}

// cleanupJobCreatedAt recovers the time cleanupJobID encoded.
func cleanupJobCreatedAt(jobID string) (time.Time, error) {
	stamp := jobID[strings.LastIndex(jobID, "-")+1:]
	t, err := time.ParseInLocation(cleanupJobTimeFormat, stamp, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("cannot read the creation time from job name %q: %w", jobID, err)
	}
	return t, nil
}

// cleanupCron renders a 5-field cron firing every 2 hours, first at
// now+cleanupRetryDelay. Cloud Scheduler has no one-shot schedule, and the
// repetition is wanted here: each fire is another attempt.
func cleanupCron(now time.Time) string {
	first := now.Add(cleanupRetryDelay).UTC()
	return fmt.Sprintf("%d %d-23/2 * * *", first.Minute(), first.Hour()%2)
}

func deleteGcpCleanupJob(ctx context.Context, jobID string) error {
	region := gcpRegionFromEnv()
	if region == "" {
		return errors.New("missing required environment variable: GCLOUD_REGION")
	}
	client, err := scheduler.NewCloudSchedulerClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create scheduler client: %w", err)
	}
	defer client.Close()

	_, name := schedulerJobName(gcpProjectFromEnv(), region, jobID)
	if err := client.DeleteJob(ctx, &schedulerpb.DeleteJobRequest{Name: name}); err != nil {
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete the retry job %s: %w", jobID, err)
	}
	return nil
}

func schedulerJobName(gcpProject, region, jobID string) (string, string) {
	parent := fmt.Sprintf("projects/%s/locations/%s", gcpProject, region)
	return parent, parent + "/jobs/" + jobID
}

// gcpProjectFromEnv mirrors cd/config.go: GCP_PROJECT is kept for old CLIs. An
// empty result means this is not a GCP run.
func gcpProjectFromEnv() string {
	return getenv("GCLOUD_PROJECT", os.Getenv("GCP_PROJECT"))
}

func gcpRegionFromEnv() string {
	return getenv("GCLOUD_REGION", os.Getenv("REGION"))
}

// isNotFound reports whether err is a 404 from either transport, in which case
// the job is already gone.
func isNotFound(err error) bool {
	var gerr *googleapi.Error
	if errors.As(err, &gerr) && gerr.Code == 404 {
		return true
	}
	var aerr *apierror.APIError
	if errors.As(err, &aerr) {
		if aerr.HTTPCode() == 404 {
			return true
		}
		if s := aerr.GRPCStatus(); s != nil && s.Code() == codes.NotFound {
			return true
		}
	}
	return false
}
