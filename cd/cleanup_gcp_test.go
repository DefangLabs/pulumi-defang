package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DefangLabs/pulumi-defang/cd/program"
)

func TestCleanupJobIDRoundTrip(t *testing.T) {
	now := time.Date(2026, 8, 19, 14, 30, 45, 0, time.UTC)
	jobID := cleanupJobID("myproj", "preview", now)
	if want := "defang-cleanup-myproj-preview-20260819143045"; jobID != want {
		t.Errorf("jobID = %q, want %q", jobID, want)
	}
	got, err := cleanupJobCreatedAt(jobID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(now) {
		t.Errorf("createdAt = %s, want %s", got, now)
	}
}

// The stamp is written as UTC and must be read as UTC, or the deadline is off by
// the local offset. Asserted on the returned Location: setting TZ would not
// help, as Go resolves time.Local once per process.
func TestCleanupJobCreatedAtIsUTC(t *testing.T) {
	got, err := cleanupJobCreatedAt("defang-cleanup-p-s-20260819143045")
	if err != nil {
		t.Fatal(err)
	}
	if got.Location() != time.UTC {
		t.Errorf("location = %s, want UTC", got.Location())
	}
}

func TestCleanupJobCreatedAtRejectsGarbage(t *testing.T) {
	for _, jobID := range []string{"defang-cleanup-p-s-notatime", "nodashes", ""} {
		if _, err := cleanupJobCreatedAt(jobID); err == nil {
			t.Errorf("cleanupJobCreatedAt(%q) = nil error, want error", jobID)
		}
	}
}

// The deadline must outlast the IP-release window by a wide margin, or a retry
// gives up while it is still being told to wait.
func TestCleanupDeadlineOutlastsTheWindow(t *testing.T) {
	if cleanupDeadline <= 2*time.Hour {
		t.Errorf("cleanupDeadline = %s, must exceed the 1-2h IP-release window", cleanupDeadline)
	}
	if cleanupRetryDelay <= 0 || cleanupRetryDelay >= 2*time.Hour {
		t.Errorf("cleanupRetryDelay = %s, want inside the 2h cron period", cleanupRetryDelay)
	}
}

func TestCleanupCron(t *testing.T) {
	tests := []struct {
		now  time.Time
		want string
	}{
		// +1h59m lands at 15:29, an odd hour -> the odd 2-hourly series.
		{time.Date(2026, 8, 19, 13, 30, 0, 0, time.UTC), "29 1-23/2 * * *"},
		// +1h59m lands at 14:29, an even hour -> the even series.
		{time.Date(2026, 8, 19, 12, 30, 0, 0, time.UTC), "29 0-23/2 * * *"},
		// Crossing midnight must still yield a valid hour field.
		{time.Date(2026, 8, 19, 23, 0, 0, 0, time.UTC), "59 0-23/2 * * *"},
	}
	for _, tt := range tests {
		if got := cleanupCron(tt.now); got != tt.want {
			t.Errorf("cleanupCron(%s) = %q, want %q", tt.now, got, tt.want)
		}
	}
}

// The scheduled run must re-run `down`, not a bespoke command: the whole point
// is that Pulumi performs the remaining deletes itself.
func TestGcpCleanupBuildRerunsDown(t *testing.T) {
	environ := []string{
		"PROJECT=myproj",
		"STACK=preview",
		"GCLOUD_PROJECT=my-gcp-project",              // kept: GCLOUD_ prefix
		"DEFANG_STATES_UPLOAD_URL=https://presigned", // dropped: expires before the retry fires
		"PATH=/usr/bin",                              // dropped
	}
	body, err := gcpCleanupBuild("us-docker.pkg.dev/defang/cd:v2", "my-gcp-project", "cd@my-gcp-project.iam.gserviceaccount.com", "preview", "defang-cleanup-myproj-preview-20260819143045", nil, environ)
	if err != nil {
		t.Fatal(err)
	}

	var build struct {
		Steps []struct {
			Name string   `json:"name"`
			Args []string `json:"args"`
			Env  []string `json:"env"`
		} `json:"steps"`
		Options struct {
			Logging string `json:"logging"`
		} `json:"options"`
		Timeout        string   `json:"timeout"`
		Tags           []string `json:"tags"`
		ServiceAccount string   `json:"serviceAccount"`
	}
	if err := json.Unmarshal(body, &build); err != nil {
		t.Fatal(err)
	}
	if len(build.Steps) != 1 {
		t.Fatalf("steps = %d", len(build.Steps))
	}
	step := build.Steps[0]
	if step.Name != "us-docker.pkg.dev/defang/cd:v2" {
		t.Errorf("step name = %q", step.Name)
	}
	if strings.Join(step.Args, " ") != "down" {
		t.Errorf("args = %v, want [down]", step.Args)
	}
	// The job name must reach the retry, or a successful retry cannot retire it.
	want := "CLEAN_UP_JOB_NAME=defang-cleanup-myproj-preview-20260819143045,CLEAN_UP_URNS=," +
		"GCLOUD_PROJECT=my-gcp-project,PROJECT=myproj,STACK=preview"
	if got := strings.Join(step.Env, ","); got != want {
		t.Errorf("env = %q, want %q", got, want)
	}
	if build.Options.Logging != "CLOUD_LOGGING_ONLY" {
		t.Errorf("logging = %q", build.Options.Logging)
	}
	if want := fmt.Sprintf("%ds", int(program.CdTimeout.Seconds())); build.Timeout != want {
		t.Errorf("timeout = %q, want %q", build.Timeout, want)
	}
	if build.ServiceAccount != "projects/my-gcp-project/serviceAccounts/cd@my-gcp-project.iam.gserviceaccount.com" {
		t.Errorf("serviceAccount = %q", build.ServiceAccount)
	}
}

func TestSchedulerJobName(t *testing.T) {
	parent, name := schedulerJobName("my-gcp-project", "us-central1", "defang-cleanup-p-s-20260819143045")
	if want := "projects/my-gcp-project/locations/us-central1"; parent != want {
		t.Errorf("parent = %q, want %q", parent, want)
	}
	if want := "projects/my-gcp-project/locations/us-central1/jobs/defang-cleanup-p-s-20260819143045"; name != want {
		t.Errorf("name = %q, want %q", name, want)
	}
}

// An empty GCP project is how a non-GCP run is recognised, so the fallback to
// the old variable name must not invent one.
func TestGcpProjectFromEnv(t *testing.T) {
	t.Setenv("GCLOUD_PROJECT", "")
	t.Setenv("GCP_PROJECT", "")
	if got := gcpProjectFromEnv(); got != "" {
		t.Errorf("gcpProjectFromEnv() = %q, want empty", got)
	}
	t.Setenv("GCP_PROJECT", "legacy-name")
	if got := gcpProjectFromEnv(); got != "legacy-name" {
		t.Errorf("gcpProjectFromEnv() = %q, want the GCP_PROJECT fallback", got)
	}
	t.Setenv("GCLOUD_PROJECT", "current-name")
	if got := gcpProjectFromEnv(); got != "current-name" {
		t.Errorf("gcpProjectFromEnv() = %q, want GCLOUD_PROJECT to win", got)
	}
}

// --- classification -------------------------------------------------------

// realCheckpoint mirrors the shape of a checkpoint in the CD bucket: the stack
// and the Project component are custom:false, and the VPC/subnet are children of
// the component. Taken from gs://defang-cd-.../html-css-js/newprovidergcp.json.bak
const realCheckpoint = `{"resources": [
  {"type": "pulumi:pulumi:Stack",        "custom": false, "urn": "urn:pulumi:np::html-css-js::pulumi:pulumi:Stack::html-css-js-np"},
  {"type": "pulumi:providers:gcp",       "custom": true,  "urn": "urn:pulumi:np::html-css-js::pulumi:providers:gcp::default"},
  {"type": "defang-gcp:index:Project",   "custom": false, "urn": "urn:pulumi:np::html-css-js::defang-gcp:index:Project::html-css-js"},
  {"type": "gcp:compute/network:Network","custom": true,  "urn": "urn:pulumi:np::html-css-js::defang-gcp:index:Project$gcp:compute/network:Network::html-css-js-vpc"},
  {"type": "gcp:compute/subnetwork:Subnetwork","custom": true,"urn": "urn:pulumi:np::html-css-js::defang-gcp:index:Project$gcp:compute/subnetwork:Subnetwork::html-css-js-subnet"}
]}`

// The Project component is custom:false and is the PARENT of the VPC, so it is
// still in state exactly when a child failed to delete. Counting it would reject
// the only case this feature exists for — the bug that made the first version of
// this inert.
func TestRemainingResourcesIgnoresComponentsAndStack(t *testing.T) {
	got, err := remainingResources([]byte(realCheckpoint))
	if err != nil {
		t.Fatal(err)
	}
	// Providers are custom:true but are not cloud resources; an earlier version
	// of this test stripped them with a helper, which hid that the production
	// path did not, leaving the feature inert.
	for _, res := range got {
		switch {
		case res.Type == "defang-gcp:index:Project", res.Type == typeStackForTest:
			t.Errorf("non-custom resource %q must be ignored", res.Type)
		case strings.HasPrefix(res.Type, "pulumi:providers:"):
			t.Errorf("provider %q must be ignored", res.Type)
		}
	}
	if want := 2; len(got) != want {
		t.Errorf("remainingResources returned %d resources, want %d: %+v", len(got), want, got)
	}
	if !onlyPendingTeardown(got) {
		t.Error("a state holding only the VPC and subnet must classify as a pending teardown")
	}
}

const typeStackForTest = "pulumi:pulumi:Stack"

func res(t, urn string) stateResource { return stateResource{Type: t, URN: urn, Custom: true} }

func TestOnlyPendingTeardown(t *testing.T) {
	tests := []struct {
		name string
		in   []stateResource
		want bool
	}{
		{"the VPC and subnet", []stateResource{res(typeNetwork, "urn::a-vpc"), res(typeSubnetwork, "urn::a-subnet")}, true},
		{"everything the window can catch", []stateResource{
			res(typeNetwork, "urn::a-vpc"),
			res(typeSubnetwork, "urn::a-subnet"),
			res(typeSvcConnection, "urn::a-svc-conn"),
			res(typeInstanceTemplate, "urn::a-instance-template"),
			// The connection's dependency, skipped when the connection fails.
			res(typeGlobalAddress, "urn::a-peering-ip"),
		}, true},
		// A finished destroy leaves nothing; a retry would have nothing to do.
		{"nothing left", nil, false},
		// The project's public address is the same type as the peering address
		// and must never be treated as pending.
		{"the public IP address", []stateResource{res(typeNetwork, "urn::a-vpc"), res(typeGlobalAddress, "urn::a-ip")}, false},
		// Real failures that must still fail the down, even though they would
		// also block the network.
		{"a live Cloud Run service", []stateResource{res(typeNetwork, "urn::a-vpc"), res("gcp:cloudrunv2/service:Service", "urn::a-svc")}, false},
		{"a surviving Cloud SQL instance", []stateResource{res("gcp:sql/databaseInstance:DatabaseInstance", "urn::a-db"), res(typeNetwork, "urn::a-vpc")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := onlyPendingTeardown(tt.in); got != tt.want {
				t.Errorf("onlyPendingTeardown = %v, want %v", got, tt.want)
			}
		})
	}
}

// State says WHAT survived; only the error says WHY. Without this a permission
// failure or an API outage would be converted into a successful down.
func TestIsInUseFailure(t *testing.T) {
	yes := []string{
		"error: deleting urn:...: googleapi: Error 400: resourceInUseByAnotherResource",
		"Failed to delete connection; Producer services (e.g. CloudSQL) are still using this connection.",
		"The instance_template resource 'x' is already being used by 'y'",
	}
	for _, msg := range yes {
		if !isInUseFailure(errors.New(msg)) {
			t.Errorf("isInUseFailure(%q) = false, want true", msg)
		}
	}
	no := []string{
		"error: googleapi: Error 403: Permission denied on resource",
		"error: Quota exceeded for quota metric 'Queries'",
		"error: rpc error: code = Unavailable desc = the service is currently unavailable",
	}
	for _, msg := range no {
		if isInUseFailure(errors.New(msg)) {
			t.Errorf("isInUseFailure(%q) = true, want false", msg)
		}
	}
	if isInUseFailure(nil) {
		t.Error("isInUseFailure(nil) = true, want false")
	}
}

// --- the replacement-deployment guard ------------------------------------

// The schedule points at a project/stack, not at a deployment. If a redeploy
// lands in the retry window, the retry must NOT destroy it.
func TestUnexpectedURNsDetectsARedeploy(t *testing.T) {
	const vpc = "urn:pulumi:np::p::defang-gcp:index:Project$gcp:compute/network:Network::p-vpc"
	const subnet = "urn:pulumi:np::p::defang-gcp:index:Project$gcp:compute/subnetwork:Subnetwork::p-subnet"
	expected := urnSet(vpc + "," + subnet)

	// The leftovers the job was scheduled for: nothing unexpected.
	same := []stateResource{res(typeNetwork, vpc), res(typeSubnetwork, subnet)}
	if got := unexpectedURNs(same, expected); len(got) != 0 {
		t.Errorf("unexpectedURNs = %v, want none", got)
	}

	// A redeploy adds a service; the retry must stand down.
	redeployed := append(same, res("gcp:cloudrunv2/service:Service",
		"urn:pulumi:np::p::defang-gcp:index:Project$gcp:cloudrunv2/service:Service::p-web"))
	got := unexpectedURNs(redeployed, expected)
	if len(got) != 1 || !strings.HasSuffix(got[0], "p-web") {
		t.Errorf("unexpectedURNs = %v, want the new service", got)
	}

	// A fresh deployment reusing the same names still differs by URN only if
	// the stack differs; the same-URN case is covered by the type check above.
	// An empty recorded set must treat everything as unexpected, so a job
	// scheduled without URNs can never destroy anything.
	if got := unexpectedURNs(same, urnSet("")); len(got) != 2 {
		t.Errorf("unexpectedURNs with no recorded URNs = %v, want all resources", got)
	}
}

func TestJoinURNsIsSortedAndRoundTrips(t *testing.T) {
	in := []stateResource{res(typeSubnetwork, "urn::b"), res(typeNetwork, "urn::a")}
	csv := joinURNs(in)
	if csv != "urn::a,urn::b" {
		t.Errorf("joinURNs = %q, want sorted", csv)
	}
	set := urnSet(csv)
	if !set["urn::a"] || !set["urn::b"] || len(set) != 2 {
		t.Errorf("urnSet(%q) = %v", csv, set)
	}
}

// The recorded URNs must reach the retry, or its guard has nothing to compare
// against and it refuses to destroy anything.
func TestGcpCleanupBuildCarriesJobAndURNs(t *testing.T) {
	remaining := []stateResource{res(typeNetwork, "urn::p-vpc"), res(typeSubnetwork, "urn::p-subnet")}
	body, err := gcpCleanupBuild("img", "proj", "sa@proj.iam.gserviceaccount.com", "preview",
		"defang-cleanup-p-preview-20260819143045", remaining,
		[]string{"PROJECT=p", "STACK=preview", "GCLOUD_PROJECT=proj", "PATH=/usr/bin"})
	if err != nil {
		t.Fatal(err)
	}
	var build struct {
		Steps []struct {
			Args []string `json:"args"`
			Env  []string `json:"env"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(body, &build); err != nil {
		t.Fatal(err)
	}
	if strings.Join(build.Steps[0].Args, " ") != "down" {
		t.Errorf("args = %v, want [down]", build.Steps[0].Args)
	}
	env := strings.Join(build.Steps[0].Env, "\n")
	if !strings.Contains(env, "CLEAN_UP_JOB_NAME=defang-cleanup-p-preview-20260819143045") {
		t.Errorf("job name missing from env:\n%s", env)
	}
	if !strings.Contains(env, "CLEAN_UP_URNS=urn::p-subnet,urn::p-vpc") {
		t.Errorf("recorded URNs missing from env:\n%s", env)
	}
}
