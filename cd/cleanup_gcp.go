package main

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/compute/metadata"
	scheduler "cloud.google.com/go/scheduler/apiv1"
	"cloud.google.com/go/scheduler/apiv1/schedulerpb"
	"cloud.google.com/go/storage"
	"github.com/DefangLabs/defang/src/pkg/cli/client"
	"github.com/DefangLabs/pulumi-defang/cd/program"
	"github.com/googleapis/gax-go/v2/apierror"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// cdCommandCleanup deletes the VPC a `down` deliberately leaves behind.
//
// Why the VPC is not deleted inline: Cloud Run attaches to the VPC with Direct
// VPC egress (see buildVpcAccess in provider/defanggcp/gcp/cloudrun.go), and
// GCP holds the subnet's IP addresses for 1-2 hours after the service is gone.
// Until they are released the subnet delete fails with
// resourceInUseByAnotherResource, and the network cannot go before its subnet.
// This is documented GCP behaviour, not a provider bug, so no amount of
// retrying inside the destroy would help:
// https://docs.cloud.google.com/run/docs/configuring/vpc-direct-vpc
//
// Both this repo and the old MVP CD hit that same constraint and retain the
// network and subnet so Pulumi removes them from state without attempting a
// delete that cannot yet succeed. MVP also retains the MIG instance templates
// and the Service Networking connection. Its delayed cleanup complements those
// retains by deleting the physical subnet and network later. This CD preserves
// that two-phase lifecycle: RetainOnDelete during the destroy, followed by the
// scheduled cleanup once GCP has released the subnet.
//
// Either way the deferred clean-up is required, and it was the piece missing
// here — so the retained networks accumulated until they exhausted the
// project's NETWORKS quota (30 per project), which is what issue 183 is about.
const cdCommandCleanup = client.CdCommand("cleanup")

// cleanupJobTimeFormat stamps the scheduler job name with its own creation
// time, so the cleanup run can tell which state-file generations predate it.
const cleanupJobTimeFormat = "20060102150405"

// cleanupJobEnvVar names the scheduler job in the environment of the build it
// schedules, so the run can delete the job that started it.
const cleanupJobEnvVar = "CLEAN_UP_JOB_NAME"

// deleteConcurrency bounds the parallel deletes. The template and router lists
// are project-wide and a matching stack can hold many, so the fan-out needs a
// ceiling to stay within the Compute API's rate limits.
const deleteConcurrency = 8

// cleanupFirstRunDelay delays the first fire to the end of the 1-2h window GCP
// needs to release the subnet's IP addresses (see cdCommandCleanup). It also
// keeps the first run from happening immediately, which the every-2-hours cron
// would otherwise do. A first fire that is still too early is not a problem:
// the delete fails, and the next fire two hours later retries.
const cleanupFirstRunDelay = 1*time.Hour + 59*time.Minute

// pulumiState is the subset of a Pulumi checkpoint the cleanup reads: the
// resource list, to recover the network's id.
type pulumiState struct {
	Checkpoint struct {
		Latest struct {
			Resources []struct {
				Type string `json:"type"`
				Id   string `json:"id"`
			} `json:"resources,omitempty"`
		} `json:"latest"`
	} `json:"checkpoint"`
}

// gcpNetworkType is the Pulumi type token of the retained VPC.
const gcpNetworkType = "gcp:compute/network:Network"

// stackDir is the folder the DIY backend keeps checkpoints in, under
// workspace.BookkeepingDir — see pulumi/pkg/backend/diy/store.go. The sdk used
// to export this as workspace.StackDir but dropped it, so it lives here now.
const stackDir = "stacks"

// cleanupJobID names the scheduler job for one stack's cleanup. The trailing
// timestamp both keeps concurrent downs from colliding and records when the
// job was created.
func cleanupJobID(projectName, stackName string, now time.Time) string {
	return "defang-cleanup-" + projectName + "-" + stackName + "-" + now.UTC().Format(cleanupJobTimeFormat)
}

// cleanupJobCreatedAt recovers the creation time cleanupJobID encoded.
func cleanupJobCreatedAt(jobID string) (time.Time, error) {
	stamp := jobID[strings.LastIndex(jobID, "-")+1:]
	t, err := time.ParseInLocation(cleanupJobTimeFormat, stamp, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("cannot read the creation time from job name %q: %w", jobID, err)
	}
	return t, nil
}

// cleanupCron renders a 5-field cron expression firing every 2 hours, first at
// now+cleanupFirstRunDelay. Unlike the self-destruct trigger this repeats on
// purpose: the network delete fails while anything still references the
// network, so each fire is a retry. A successful run deletes the job.
func cleanupCron(now time.Time) string {
	first := now.Add(cleanupFirstRunDelay).UTC()
	return fmt.Sprintf("%d %d-23/2 * * *", first.Minute(), first.Hour()%2)
}

// gcpCleanupBuild renders the builds.create request body that re-runs this CD
// image with args ["cleanup"] — the same shape as gcpSelfDestructBuild in
// cd/program, plus the job name so the run can delete its own scheduler job.
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
			"args": []string{string(cdCommandCleanup)},
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

// scheduleGcpCleanup creates the Cloud Scheduler job that runs the cleanup.
// It is called after a full destroy; the caller only warns on failure, because
// a missing cleanup job must not fail the down itself.
func scheduleGcpCleanup(ctx context.Context, projectName, stackName string) error {
	gcpProject := gcpProjectFromEnv()
	if gcpProject == "" {
		return errors.New("missing required environment variable: GCLOUD_PROJECT")
	}
	region := getenv("GCLOUD_REGION", os.Getenv("REGION"))
	if region == "" {
		return errors.New("missing required environment variable: GCLOUD_REGION")
	}
	cdImage := os.Getenv("DEFANG_CD_IMAGE")
	if cdImage == "" {
		return errors.New("missing required environment variable: DEFANG_CD_IMAGE")
	}

	// The scheduled build must run as the service account this run uses; Cloud
	// Build exposes it through the metadata server.
	saCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	saEmail, err := metadata.EmailWithContext(saCtx, "default")
	if err != nil {
		return fmt.Errorf("cleanup needs the CD to run in Cloud Build: %w", err)
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
			Description: "Defang cleanup of the retained VPC; deletes itself once it succeeds",
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
		return fmt.Errorf("failed to create cleanup job: %w", err)
	}
	warn("Scheduled cleanup of the retained VPC as", jobID)
	return nil
}

// cleanupGCP is the `cleanup` command. It deletes the retained VPC of the
// stack it was scheduled for, then deletes its own scheduler job.
func cleanupGCP(ctx context.Context, projectName, stackName string) error {
	jobID := os.Getenv(cleanupJobEnvVar)
	if jobID == "" {
		return &usageError{msg: "missing required environment variable: " + cleanupJobEnvVar}
	}
	gcpProject := gcpProjectFromEnv()
	if gcpProject == "" {
		return errors.New("missing required environment variable: GCLOUD_PROJECT")
	}
	region := getenv("GCLOUD_REGION", os.Getenv("REGION"))
	if region == "" {
		return errors.New("missing required environment variable: GCLOUD_REGION")
	}

	if err := cleanupGcpNetwork(ctx, gcpProject, region, projectName, stackName, jobID); err != nil {
		// A state backup with no network will never grow one, so retrying
		// cannot help: drop through and delete the job, or it fires every two
		// hours for ever. Any other failure (a blocked delete, an I/O error)
		// may well succeed later, so leave the job in place to retry.
		if !errors.Is(err, errNoNetworkInState) {
			return err
		}
		warn(" **", err, "- giving up and removing the cleanup job")
	}

	return deleteSchedulerJob(ctx, gcpProject, region, jobID)
}

// errNoNetworkInState marks the one failure that is not worth retrying.
var errNoNetworkInState = errors.New("no retained VPC found in the stack state backup")

// cleanupGcpNetwork finds this stack's network in the state backup and tears it
// down. A state file that is present again means the stack was redeployed, so
// there is nothing to clean up.
func cleanupGcpNetwork(ctx context.Context, gcpProject, region, projectName, stackName, jobID string) error {
	createdAt, err := cleanupJobCreatedAt(jobID)
	if err != nil {
		return err
	}

	bucket, object, err := stackStateObject(projectName, stackName)
	if err != nil {
		return err
	}

	gcs, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %w", err)
	}
	defer gcs.Close()

	exists, err := objectExists(ctx, gcs, bucket, object)
	if err != nil {
		return fmt.Errorf("failed to check the state file gs://%s/%s: %w", bucket, object, err)
	}
	if exists {
		warn("Stack", stackName, "exists again; skipping the VPC cleanup")
		return nil
	}

	// Pulumi keeps the previous checkpoint alongside the live one. Once the
	// stack is removed only the backup is left, and it still records the
	// retained network.
	networkID, err := networkFromStateBackup(ctx, gcs, bucket, object+".bak", createdAt)
	if err != nil {
		return err
	}

	warn("Deleting the retained VPC", networkID)
	return deleteGcpNetwork(ctx, gcpProject, region, networkID)
}

// stackStateObject derives the bucket and object of the stack's checkpoint in
// the DIY (GCS) backend, matching pulumi/pkg/backend/diy.
func stackStateObject(projectName, stackName string) (string, string, error) {
	backendURL := getenv("PULUMI_BACKEND_URL", os.Getenv("DEFANG_STATE_URL"))
	if backendURL == "" {
		return "", "", errors.New("missing required environment variable: PULUMI_BACKEND_URL")
	}
	u, err := url.Parse(backendURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid PULUMI_BACKEND_URL %q: %w", backendURL, err)
	}
	u.Path = path.Join(u.Path, workspace.BookkeepingDir, stackDir, projectName, stackName+".json")
	return parseGCSURL(u.String())
}

// parseGCSURL splits a gs:// URL into its bucket and object.
func parseGCSURL(raw string) (string, string, error) {
	rest, ok := strings.CutPrefix(raw, "gs://")
	if !ok {
		return "", "", fmt.Errorf("expected a gs:// URL, got %q", raw)
	}
	bucket, object, ok := strings.Cut(rest, "/")
	if !ok || bucket == "" || object == "" {
		return "", "", fmt.Errorf("expected a bucket and an object in %q", raw)
	}
	// The stack name may be base64, which can carry escaped characters.
	object, err := url.PathUnescape(object)
	if err != nil {
		return "", "", fmt.Errorf("invalid object path in %q: %w", raw, err)
	}
	return bucket, object, nil
}

// networkFromStateBackup reads the newest generation of the state backup that
// predates the cleanup job and returns the network's id. Generations created
// after the job belong to a later deploy and are ignored. The bucket is
// versioned by the CLI (EnsureBucketExists with versioning), so the walk back
// through generations reaches a checkpoint that still holds the network even
// when the last one no longer does.
func networkFromStateBackup(ctx context.Context, gcs *storage.Client, bucket, object string, before time.Time) (string, error) {
	generations, err := objectGenerations(ctx, gcs, bucket, object)
	if err != nil {
		return "", fmt.Errorf("failed to list the generations of gs://%s/%s: %w", bucket, object, err)
	}
	for _, gen := range candidateGenerations(generations, before) {
		state, err := readStateFile(ctx, gcs, bucket, object, gen.generation)
		if err != nil {
			return "", err
		}
		for _, res := range state.Checkpoint.Latest.Resources {
			if res.Type == gcpNetworkType {
				return res.Id, nil
			}
		}
	}
	return "", fmt.Errorf("%w: looked for %s in gs://%s/%s", errNoNetworkInState, gcpNetworkType, bucket, object)
}

type objectGeneration struct {
	generation int64
	created    time.Time
}

// candidateGenerations orders the generations newest first and drops any created
// at or after `before` (the cleanup job's own creation time).
//
// The filter is what keeps this safe rather than merely correct: a generation
// written after the job was scheduled belongs to a LATER deploy, whose network
// is live. Reading one would hand back a network still in use and the teardown
// would delete it. Only older checkpoints describe the stack this job was
// scheduled to clean up.
func candidateGenerations(generations []objectGeneration, before time.Time) []objectGeneration {
	out := make([]objectGeneration, 0, len(generations))
	for _, gen := range generations {
		// Not After: a generation stamped exactly at the cut-off is not older
		// than the job, so it cannot be assumed to predate it.
		if gen.created.Before(before) {
			out = append(out, gen)
		}
	}
	slices.SortFunc(out, func(a, b objectGeneration) int { return cmp.Compare(b.generation, a.generation) })
	return out
}

func objectGenerations(ctx context.Context, gcs *storage.Client, bucket, object string) ([]objectGeneration, error) {
	var out []objectGeneration
	it := gcs.Bucket(bucket).Objects(ctx, &storage.Query{Prefix: object, Versions: true})
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		// The prefix query also matches longer names.
		if attrs.Name == object {
			out = append(out, objectGeneration{generation: attrs.Generation, created: attrs.Created})
		}
	}
}

func readStateFile(ctx context.Context, gcs *storage.Client, bucket, object string, generation int64) (*pulumiState, error) {
	r, err := gcs.Bucket(bucket).Object(object).Generation(generation).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read gs://%s/%s generation %d: %w", bucket, object, generation, err)
	}
	defer r.Close()
	var state pulumiState
	if err := json.NewDecoder(r).Decode(&state); err != nil {
		return nil, fmt.Errorf("failed to decode gs://%s/%s generation %d: %w", bucket, object, generation, err)
	}
	return &state, nil
}

func objectExists(ctx context.Context, gcs *storage.Client, bucket, object string) (bool, error) {
	_, err := gcs.Bucket(bucket).Object(object).Attrs(ctx)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, storage.ErrObjectNotExist) || isNotFound(err) {
		return false, nil
	}
	return false, err
}

// deleteGcpNetwork removes the network and everything still referencing it, in
// the order GCP requires. Each step tolerates an already-deleted resource, so
// a retry after a partial run makes progress.
//
// The referencing resources exist because the GCP provider retains them on
// delete: the subnet (gcp.go), the servicenetworking peering (vpc_peering.go)
// and the MIG instance templates (compute.go). Cloud Routers are not retained,
// but a destroy that continued on error can leave one behind, and it blocks
// the network delete just the same.
func deleteGcpNetwork(ctx context.Context, gcpProject, region, networkID string) error {
	networks, err := compute.NewNetworksRESTClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create networks client: %w", err)
	}
	defer networks.Close()

	name := path.Base(networkID)

	// List the subnets up front: the instance templates are matched on their
	// subnetwork, not their network (see deleteInstanceTemplates), and the same
	// list is what gets deleted below.
	subnets, err := listSubnetworks(ctx, gcpProject, region, networkID)
	if err != nil {
		return err
	}

	if err := deleteInstanceTemplates(ctx, gcpProject, networkID, subnets); err != nil {
		return err
	}
	if err := removeNetworkPeerings(ctx, networks, gcpProject, name); err != nil {
		return err
	}
	if err := deleteRouters(ctx, gcpProject, region, networkID); err != nil {
		return err
	}
	if err := deleteSubnetworks(ctx, gcpProject, region, subnets); err != nil {
		return err
	}

	op, err := networks.Delete(ctx, &computepb.DeleteNetworkRequest{
		Project: gcpProject,
		Network: name,
	})
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete network %s: %w", name, err)
	}
	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for the delete of network %s: %w", name, err)
	}
	return nil
}

// deleteInstanceTemplates deletes the retained MIG instance templates that hold
// the network, either directly or through one of its subnets. Templates are a
// global resource, so the whole project is listed and filtered.
//
// A template is matched on either field. The code only ever sets Subnetwork
// (see the network interface built in provider/defanggcp/gcp/compute.go), but
// GCP fills Network in from it, so reading either one back normally works —
// verified against live templates in defang-playground-dev, which were created
// with Subnetwork alone and report both. Matching only Network would leave the
// whole teardown resting on that server-side normalisation, and a template we
// fail to spot blocks the network delete for ever, so match both.
func deleteInstanceTemplates(ctx context.Context, gcpProject, networkID string, subnets []subnetwork) error {
	templates, err := compute.NewInstanceTemplatesRESTClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create instance templates client: %w", err)
	}
	defer templates.Close()

	grp, grpCtx := errgroup.WithContext(ctx)
	grp.SetLimit(deleteConcurrency)
	var listErr error
	it := templates.List(ctx, &computepb.ListInstanceTemplatesRequest{Project: gcpProject})
	for {
		tmpl, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			// Break rather than return: the deletes already started must still
			// be waited on, or they run unsupervised while the process exits.
			listErr = fmt.Errorf("failed to list instance templates: %w", err)
			break
		}
		if tmpl.GetProperties() == nil || !usesNetwork(tmpl.GetProperties().GetNetworkInterfaces(), networkID, subnets) {
			continue
		}
		tmplName := tmpl.GetName()
		grp.Go(func() error {
			op, err := templates.Delete(grpCtx, &computepb.DeleteInstanceTemplateRequest{
				Project:          gcpProject,
				InstanceTemplate: tmplName,
			})
			if err != nil {
				if isNotFound(err) {
					return nil
				}
				return fmt.Errorf("failed to delete instance template %s: %w", tmplName, err)
			}
			if err := op.Wait(grpCtx); err != nil {
				return fmt.Errorf("failed to wait for the delete of instance template %s: %w", tmplName, err)
			}
			return nil
		})
	}
	return errors.Join(grp.Wait(), listErr)
}

// usesNetwork reports whether any interface holds the network directly or sits
// in one of its subnets.
func usesNetwork(interfaces []*computepb.NetworkInterface, networkID string, subnets []subnetwork) bool {
	for _, ni := range interfaces {
		if referencesNetwork(ni.GetNetwork(), networkID) {
			return true
		}
		for _, subnet := range subnets {
			if referencesNetwork(ni.GetSubnetwork(), subnet.id) {
				return true
			}
		}
	}
	return false
}

// removeNetworkPeerings drops every peering on the network, including the
// retained servicenetworking connection. A network with a peering cannot be
// deleted, and the peering outlives the address range it was created from.
func removeNetworkPeerings(ctx context.Context, networks *compute.NetworksClient, gcpProject, name string) error {
	network, err := networks.Get(ctx, &computepb.GetNetworkRequest{Project: gcpProject, Network: name})
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get network %s: %w", name, err)
	}
	for _, peering := range network.GetPeerings() {
		peeringName := peering.GetName()
		op, err := networks.RemovePeering(ctx, &computepb.RemovePeeringNetworkRequest{
			Project: gcpProject,
			Network: name,
			NetworksRemovePeeringRequestResource: &computepb.NetworksRemovePeeringRequest{
				Name: &peeringName,
			},
		})
		if err != nil {
			if isNotFound(err) {
				continue
			}
			return fmt.Errorf("failed to remove peering %s from network %s: %w", peeringName, name, err)
		}
		if err := op.Wait(ctx); err != nil {
			return fmt.Errorf("failed to wait for the removal of peering %s: %w", peeringName, err)
		}
	}
	return nil
}

// deleteRouters deletes the Cloud Routers on the network, which carry the NAT
// configuration and hold a reference to the network.
func deleteRouters(ctx context.Context, gcpProject, region, networkID string) error {
	routers, err := compute.NewRoutersRESTClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create routers client: %w", err)
	}
	defer routers.Close()

	grp, grpCtx := errgroup.WithContext(ctx)
	grp.SetLimit(deleteConcurrency)
	var listErr error
	it := routers.List(ctx, &computepb.ListRoutersRequest{Project: gcpProject, Region: region})
	for {
		router, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			listErr = fmt.Errorf("failed to list routers: %w", err)
			break
		}
		if !referencesNetwork(router.GetNetwork(), networkID) {
			continue
		}
		routerName := router.GetName()
		grp.Go(func() error {
			op, err := routers.Delete(grpCtx, &computepb.DeleteRouterRequest{
				Project: gcpProject,
				Region:  region,
				Router:  routerName,
			})
			if err != nil {
				if isNotFound(err) {
					return nil
				}
				return fmt.Errorf("failed to delete router %s: %w", routerName, err)
			}
			if err := op.Wait(grpCtx); err != nil {
				return fmt.Errorf("failed to wait for the delete of router %s: %w", routerName, err)
			}
			return nil
		})
	}
	return errors.Join(grp.Wait(), listErr)
}

// subnetwork is one of the network's subnets: its name, to delete it, and its
// id in the "projects/P/regions/R/subnetworks/N" form, to match references to
// it.
type subnetwork struct {
	name string
	id   string
}

// listSubnetworks returns the subnets of the network in the region. This
// includes the retained shared subnet and the load balancer's proxy-only
// subnet.
func listSubnetworks(ctx context.Context, gcpProject, region, networkID string) ([]subnetwork, error) {
	subnets, err := compute.NewSubnetworksRESTClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create subnetworks client: %w", err)
	}
	defer subnets.Close()

	var out []subnetwork
	it := subnets.List(ctx, &computepb.ListSubnetworksRequest{Project: gcpProject, Region: region})
	for {
		subnet, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list subnetworks: %w", err)
		}
		if !referencesNetwork(subnet.GetNetwork(), networkID) {
			continue
		}
		out = append(out, subnetwork{
			name: subnet.GetName(),
			id:   fmt.Sprintf("projects/%s/regions/%s/subnetworks/%s", gcpProject, region, subnet.GetName()),
		})
	}
}

// deleteSubnetworks deletes the network's subnets.
func deleteSubnetworks(ctx context.Context, gcpProject, region string, list []subnetwork) error {
	subnets, err := compute.NewSubnetworksRESTClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create subnetworks client: %w", err)
	}
	defer subnets.Close()

	grp, grpCtx := errgroup.WithContext(ctx)
	grp.SetLimit(deleteConcurrency)
	for _, subnet := range list {
		subnetName := subnet.name
		grp.Go(func() error {
			op, err := subnets.Delete(grpCtx, &computepb.DeleteSubnetworkRequest{
				Project:    gcpProject,
				Region:     region,
				Subnetwork: subnetName,
			})
			if err != nil {
				if isNotFound(err) {
					return nil
				}
				return fmt.Errorf("failed to delete subnet %s: %w", subnetName, err)
			}
			if err := op.Wait(grpCtx); err != nil {
				return fmt.Errorf("failed to wait for the delete of subnet %s: %w", subnetName, err)
			}
			return nil
		})
	}
	return grp.Wait()
}

// referencesNetwork reports whether a resource's network field points at the
// network. The field is a full self-link, while the id from Pulumi state is
// the "projects/P/global/networks/N" form, so compare on the trailing path.
func referencesNetwork(selfLink, networkID string) bool {
	if selfLink == "" {
		return false
	}
	return strings.HasSuffix(selfLink, "/"+strings.TrimPrefix(networkID, "/"))
}

// deleteSchedulerJob deletes the cleanup job itself, so a successful run does
// not fire again.
func deleteSchedulerJob(ctx context.Context, gcpProject, region, jobID string) error {
	client, err := scheduler.NewCloudSchedulerClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create scheduler client: %w", err)
	}
	defer client.Close()

	_, name := schedulerJobName(gcpProject, region, jobID)
	if err := client.DeleteJob(ctx, &schedulerpb.DeleteJobRequest{Name: name}); err != nil {
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete the cleanup job %s: %w", jobID, err)
	}
	return nil
}

func schedulerJobName(gcpProject, region, jobID string) (string, string) {
	parent := fmt.Sprintf("projects/%s/locations/%s", gcpProject, region)
	return parent, parent + "/jobs/" + jobID
}

// gcpProjectFromEnv mirrors cd/config.go: GCP_PROJECT is kept for old CLIs.
func gcpProjectFromEnv() string {
	return getenv("GCLOUD_PROJECT", os.Getenv("GCP_PROJECT"))
}

// isNotFound reports whether err is a 404 from either the REST or the gRPC
// transport, in which case the resource is already gone.
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
