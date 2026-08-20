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

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
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
// ContinueOnError and deletes everything it can. When the only resources left
// are the ones known to wait on that window AND the destroy failed for that
// reason, the `down` reports success and schedules a Cloud Scheduler job to run
// `down` again. Pulumi then performs the remaining deletes from live state, in
// its own dependency order: no orphaned resources and no hand-rolled teardown.
//
// One resource is exempt from that premise, because Pulumi genuinely cannot
// delete it: the Service Networking connection. The provider's delete is broken
// upstream (hashicorp/terraform-provider-google#16275, a 5.x regression from
// removePeering to deleteConnection, closed as a duplicate of #16944 whose
// resolution added an *abandon* deletion policy rather than a working delete),
// so a retry would fail for ever on any stack with managed Postgres or Redis.
// The program therefore abandons it (DeletionPolicy: ABANDON in
// createVPCPeeringInfra) and this file removes the peering it leaves behind,
// out of band, before the retry destroy — see clearAbandonedPeerings. That is
// one API call, rather than a hand-written teardown of the whole VPC.
//
// KNOWN GAP: all of this only applies to a state written by this version of the
// provider. RetainOnDelete is recorded per resource in the checkpoint and
// `pulumi destroy` does not re-run the program (see optdestroy.RunProgram), so a
// stack whose state still carries the old retains destroys "successfully" and
// leaks its VPC exactly as before, with no failure for this code to classify.
// Those stacks need one sweep by hand, or an `up` on this version first.
//
// Safety rules that follow from the schedule being a pointer at a project/stack
// rather than at a deployment:
//
//   - A scheduled run verifies the stack still holds ONLY what its job was
//     scheduled for BEFORE destroying anything. Otherwise a redeploy landing in
//     the retry window would be destroyed by the retry. The recorded URNs travel
//     in cleanupURNsEnvVar for exactly this check.
//   - A missing stack means someone else finished the job. That is success:
//     retire the job, destroy nothing.
//   - Every failure path of a scheduled run is bounded by cleanupDeadline, and
//     reaching it is reported as a failure, because the resources are then
//     abandoned.

// cleanupJobEnvVar names the scheduler job in the environment of the run it
// schedules, so a run started by a job can retire it.
const cleanupJobEnvVar = "CLEAN_UP_JOB_NAME"

// cleanupURNsEnvVar carries the URNs the job was scheduled to finish deleting,
// comma-separated. A scheduled run refuses to destroy anything outside this set.
const cleanupURNsEnvVar = "CLEAN_UP_URNS"

// cleanupJobTimeFormat stamps the job name with its creation time, which bounds
// how long the retries go on for.
const cleanupJobTimeFormat = "20060102150405"

// cleanupRetryDelay puts the first retry at the end of the window GCP needs to
// release the subnet's IP addresses. The cron then repeats every 2 hours, so a
// retry that is still too early simply tries again.
const cleanupRetryDelay = 1*time.Hour + 59*time.Minute

// cleanupDeadline bounds the retries. A delete still failing a day later is
// stuck on something the window does not explain (for instance
// hashicorp/terraform-provider-google#19908), so the job retires and the run
// fails rather than firing every 2 hours for ever.
const cleanupDeadline = 24 * time.Hour

// Pulumi type tokens, as they appear in a checkpoint.
const (
	typeNetwork          = "gcp:compute/network:Network"
	typeSubnetwork       = "gcp:compute/subnetwork:Subnetwork"
	typeSvcConnection    = "gcp:servicenetworking/connection:Connection"
	typeInstanceTemplate = "gcp:compute/instanceTemplate:InstanceTemplate"
	typeGlobalAddress    = "gcp:compute/globalAddress:GlobalAddress"
)

// peeringAddressSuffix identifies the VPC peering address by its logical name
// (see createVPCPeeringInfra). The type alone is not enough: the project's
// public IP is a GlobalAddress too, and must never be classified as pending.
const peeringAddressSuffix = "-peering-ip"

// pendingTeardownTypes are the resource types whose delete is expected to fail
// until GCP releases the subnet's IP addresses.
//
// The VPC and its subnet are what is actually blocked. The rest are caught by
// the same window because they hold a reference to it: an instance template
// races its MIG's asynchronous deletion, and the peering address is held by the
// peering the abandoned connection leaves behind (clearAbandonedPeerings
// releases it before the retry).
//
// The connection itself is listed only for a state written before the abandon
// policy shipped, where its delete cannot succeed at all: classifying it as
// pending means such a stack gets the retries and then a clear message at the
// deadline, rather than a bare provider error.
var pendingTeardownTypes = map[string]bool{
	typeNetwork:          true,
	typeSubnetwork:       true,
	typeSvcConnection:    true,
	typeInstanceTemplate: true,
}

// inUseErrorMarkers identify a delete that failed because GCP still considers
// the resource in use. The state says what survived; only the error says why,
// and without that a permission failure or an API outage would be converted
// into a "successful" down.
var inUseErrorMarkers = []string{
	// The subnet/network delete while Cloud Run still holds the IPs, and the
	// peering address while the abandoned peering still holds its range.
	"resourceInUseByAnotherResource",
	// The connection of a state that predates the abandon policy.
	"still using this connection",
	// The instance template while its MIG is still being deleted.
	"is already being used by",
}

// stateResource is the part of a checkpoint entry the classification reads.
type stateResource struct {
	URN string `json:"urn"`
	// Type is the Pulumi type token.
	Type string `json:"type"`
	// ID is the provider id, which is where the VPC's name comes from when its
	// peering has to be removed out of band.
	ID string `json:"id"`
	// Custom is false for the stack and for component resources, which are
	// bookkeeping rather than cloud resources.
	Custom bool `json:"custom"`
}

type deployment struct {
	Resources []stateResource `json:"resources"`
}

// providerTypePrefix marks a provider resource. Providers are custom:true but
// are not cloud resources, and Pulumi deletes them last — so a failed resource
// delete leaves its provider behind too.
const providerTypePrefix = "pulumi:providers:"

// remainingResources lists the cloud resources still in the stack's state.
//
// Two kinds of entry are excluded, both verified against a checkpoint in the CD
// bucket rather than assumed:
//
//   - The stack and component resources, which are custom:false. A component
//     necessarily outlives its children, because Pulumi deletes children first,
//     so defang-gcp:index:Project is still present precisely when one of its
//     children failed to delete — it is the parent of both the network and the
//     subnet. Counting it would reject the only case this feature handles.
//   - Providers, which are custom:true despite not being cloud resources.
//     Counting them would reject every partial teardown just as thoroughly.
func remainingResources(export []byte) ([]stateResource, error) {
	var d deployment
	if err := json.Unmarshal(export, &d); err != nil {
		return nil, fmt.Errorf("failed to read the stack state: %w", err)
	}
	var out []stateResource
	for _, res := range d.Resources {
		if res.Custom && !strings.HasPrefix(res.Type, providerTypePrefix) {
			out = append(out, res)
		}
	}
	return out, nil
}

// isPendingTeardown reports whether one resource is expected to be waiting on
// the IP-release window.
func isPendingTeardown(res stateResource) bool {
	if pendingTeardownTypes[res.Type] {
		return true
	}
	// The peering address, but never the project's public address.
	return res.Type == typeGlobalAddress && strings.HasSuffix(res.URN, peeringAddressSuffix)
}

// onlyPendingTeardown reports whether every cloud resource left in the state is
// one that waits on the IP-release window. An empty list means the destroy
// finished, which is not a pending teardown.
func onlyPendingTeardown(resources []stateResource) bool {
	if len(resources) == 0 {
		return false
	}
	for _, res := range resources {
		if !isPendingTeardown(res) {
			return false
		}
	}
	return true
}

// isInUseFailure reports whether the destroy failed because GCP still considers
// something in use, rather than for an unrelated reason.
func isInUseFailure(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, marker := range inUseErrorMarkers {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

// urnSet parses the recorded URN list.
func urnSet(csv string) map[string]bool {
	out := map[string]bool{}
	for _, urn := range strings.Split(csv, ",") {
		if urn = strings.TrimSpace(urn); urn != "" {
			out[urn] = true
		}
	}
	return out
}

// unexpectedURNs returns the resources present in state that the job was not
// scheduled to delete. A non-empty result means the stack no longer holds only
// the leftovers this job was created for — most likely a new deployment — and
// the retry must not touch it.
func unexpectedURNs(resources []stateResource, expected map[string]bool) []string {
	var out []string
	for _, res := range resources {
		if !expected[res.URN] {
			out = append(out, res.URN)
		}
	}
	return out
}

// scheduledCleanupJob returns the job that started this run, or "" for an
// ordinary run.
func scheduledCleanupJob() string { return os.Getenv(cleanupJobEnvVar) }

// networkNameFromState finds the VPC's name in the live state.
//
// The id of a gcp:compute/network:Network is
// projects/<project>/global/networks/<name>; a bare name is accepted too, so a
// different id shape cannot silently turn this into a no-op.
func networkNameFromState(resources []stateResource) string {
	for _, res := range resources {
		if res.Type == typeNetwork && res.ID != "" {
			return res.ID[strings.LastIndex(res.ID, "/")+1:]
		}
	}
	return ""
}

// clearAbandonedPeerings removes the network peerings that the destroy left
// behind, so the retry's deletes can get past them.
//
// The Service Networking connection is abandoned rather than deleted, because
// the provider cannot delete one (see createVPCPeeringInfra). Abandoning it
// leaves its peering on the VPC, and that peering holds the reserved range — so
// both the peering address and the network itself would keep failing to delete
// for ever. Nothing else ever peers with this VPC, so every peering on it is one
// to remove.
//
// Failures here are reported and not returned: the retry destroy then fails the
// way it did before, and the job fires again.
func clearAbandonedPeerings(ctx context.Context, resources []stateResource) {
	network := networkNameFromState(resources)
	if network == "" {
		return // no network left in state, so no peering of ours survives
	}
	gcpProject := gcpProjectFromEnv()

	client, err := compute.NewNetworksRESTClient(ctx)
	if err != nil {
		warn(" ** Could not reach the compute API to remove the VPC peering:", err)
		return
	}
	defer client.Close()

	net, err := client.Get(ctx, &computepb.GetNetworkRequest{Project: gcpProject, Network: network})
	if err != nil {
		if !isNotFound(err) {
			warn(" ** Could not read the VPC", network, "to remove its peerings:", err)
		}
		return
	}
	for _, peering := range net.GetPeerings() {
		name := peering.GetName()
		op, err := client.RemovePeering(ctx, &computepb.RemovePeeringNetworkRequest{
			Project: gcpProject,
			Network: network,
			NetworksRemovePeeringRequestResource: &computepb.NetworksRemovePeeringRequest{
				Name: &name,
			},
		})
		if err == nil {
			err = op.Wait(ctx)
		}
		if err != nil {
			if !isNotFound(err) {
				warn(" ** Could not remove the peering", name, "from the VPC", network, ":", err)
			}
			continue
		}
		warn(" ** Removed the abandoned peering", name, "from the VPC", network)
	}
}

// prepareScheduledCleanup runs before a scheduled retry destroys anything: it
// decides whether the destroy may proceed at all, and then clears what Pulumi
// cannot delete itself.
//
// When the destroy may not proceed, the job has been retired and the run is
// finished: either someone else completed the teardown, or a new deployment now
// occupies this project/stack and must not be destroyed.
func prepareScheduledCleanup(ctx context.Context, stack auto.Stack) (proceed bool, err error) {
	jobID := scheduledCleanupJob()
	if jobID == "" || gcpProjectFromEnv() == "" {
		return true, nil // an ordinary `down`, not a retry
	}

	export, err := stack.Export(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to read the stack state before the retry: %w", err)
	}
	resources, err := remainingResources(export.Deployment)
	if err != nil {
		return false, err
	}

	if len(resources) == 0 {
		warn(" ** Nothing left to delete; retiring the retry job", jobID)
		return false, retireCleanupJob(ctx, jobID)
	}

	expected := urnSet(os.Getenv(cleanupURNsEnvVar))
	if unexpected := unexpectedURNs(resources, expected); len(unexpected) > 0 {
		// Destroying here would delete a deployment this job knows nothing
		// about. Stand down instead, and let its own lifecycle manage it.
		warn(" ** This stack now holds resources the retry was not scheduled for, so it has been redeployed:")
		for _, urn := range unexpected {
			warn("      ", urn)
		}
		warn(" ** Not destroying anything; retiring the retry job", jobID)
		return false, retireCleanupJob(ctx, jobID)
	}

	// Only now that the state is known to hold nothing but this job's leftovers
	// is it safe to touch the VPC's peerings.
	clearAbandonedPeerings(ctx, resources)
	return true, nil
}

// handleGcpPendingTeardown decides what a GCP destroy failure means.
//
// It returns nil when the failure is only the IP-release window, having made
// sure a retry is scheduled — the `down` then reports success, because
// everything that could be deleted was. Any other failure, and any failure past
// the deadline, is returned so the caller fails.
func handleGcpPendingTeardown(ctx context.Context, stack auto.Stack, projectName, stackName string, destroyErr error) error {
	if destroyErr == nil || gcpProjectFromEnv() == "" {
		return destroyErr
	}
	// Whatever the failure is, a scheduled run past its deadline must stop
	// rerunning; otherwise its cron fires every 2 hours for ever.
	if expired, err := retireExpiredCleanupJob(ctx); expired {
		return errors.Join(destroyErr, err)
	}

	if !isInUseFailure(destroyErr) {
		return destroyErr
	}

	export, err := stack.Export(ctx)
	if err != nil {
		warn(" ** Could not read the stack state to classify the destroy failure:", err)
		return destroyErr
	}
	resources, err := remainingResources(export.Deployment)
	if err != nil {
		warn(" **", err)
		return destroyErr
	}
	if !onlyPendingTeardown(resources) {
		return destroyErr
	}

	warn(" ** The VPC cannot be deleted yet: GCP holds the subnet's IP addresses for 1-2 hours after Cloud Run releases them.")
	if scheduledCleanupJob() != "" {
		warn(" ** Leaving the retry job in place; it will try again within 2 hours.")
		return nil
	}
	if err := scheduleGcpCleanup(ctx, projectName, stackName, resources); err != nil {
		// Without a retry the VPC would leak silently, which is the whole bug
		// this exists to fix — so surface the original failure.
		warn(" ** Failed to schedule the retry:", err)
		return destroyErr
	}
	return nil
}

// retireExpiredCleanupJob deletes the job of the current run when it is past
// cleanupDeadline. It reports whether the deadline had passed, so the caller
// fails the run: the resources are abandoned at that point and only the exit
// status will show it.
func retireExpiredCleanupJob(ctx context.Context) (bool, error) {
	jobID := scheduledCleanupJob()
	if jobID == "" {
		return false, nil
	}
	createdAt, err := cleanupJobCreatedAt(jobID)
	if err != nil {
		return false, err
	}
	age := time.Since(createdAt)
	if age <= cleanupDeadline {
		return false, nil
	}
	warn(fmt.Sprintf(" ** These resources have resisted deletion for %s, which the 1-2 hour IP-release window does not explain.", age.Round(time.Hour)))
	warn(" ** Giving up: retiring the retry job. They need deleting by hand in the GCP console.")
	return true, retireCleanupJob(ctx, jobID)
}

// finishGcpCleanup runs after a destroy that succeeded, retiring the job that
// started it.
func finishGcpCleanup(ctx context.Context) {
	jobID := scheduledCleanupJob()
	if jobID == "" || gcpProjectFromEnv() == "" {
		return
	}
	if err := retireCleanupJob(ctx, jobID); err != nil {
		// The stack is gone now, so a surviving job cannot be retired by a
		// later run of this code path — say so loudly rather than implying it
		// will sort itself out.
		warn(" ** The VPC is deleted, but the retry job", jobID, "could not be removed:", err)
		warn(" ** It will keep firing every 2 hours until deleted by hand:", jobID)
		return
	}
	warn(" ** VPC deleted; retired the retry job", jobID)
}

// finishGcpCleanupForMissingStack retires the job when the stack it was
// scheduled against no longer exists, which means the teardown is done. Without
// this the run would fail at stack selection before any retirement logic and the
// cron would fire for ever.
func finishGcpCleanupForMissingStack(ctx context.Context) error {
	jobID := scheduledCleanupJob()
	if jobID == "" || gcpProjectFromEnv() == "" {
		return nil
	}
	warn(" ** The stack no longer exists, so the teardown is complete; retiring the retry job", jobID)
	return retireCleanupJob(ctx, jobID)
}

// scheduleGcpCleanup creates the Cloud Scheduler job that re-runs this `down`,
// recording the URNs it is expected to finish deleting.
func scheduleGcpCleanup(ctx context.Context, projectName, stackName string, remaining []stateResource) error {
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
	body, err := gcpCleanupBuild(cdImage, gcpProject, saEmail, stackName, jobID, remaining, os.Environ())
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
			Description: "Defang retry of `cd down` to delete the VPC once GCP releases the subnet IPs; retires itself once it succeeds",
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
// plus the job name and the URNs the retry is allowed to delete.
func gcpCleanupBuild(cdImage, gcpProject, saEmail, stackName, jobID string, remaining []stateResource, environ []string) ([]byte, error) {
	env := program.SelfDestructEnv(environ)
	env[cleanupJobEnvVar] = jobID
	env[cleanupURNsEnvVar] = joinURNs(remaining)
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

// joinURNs renders the URNs for cleanupURNsEnvVar, sorted so the value is
// stable for a given state.
func joinURNs(resources []stateResource) string {
	urns := make([]string, 0, len(resources))
	for _, res := range resources {
		urns = append(urns, res.URN)
	}
	slices.Sort(urns)
	return strings.Join(urns, ",")
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

func retireCleanupJob(ctx context.Context, jobID string) error {
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
