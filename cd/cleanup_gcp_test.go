package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DefangLabs/pulumi-defang/cd/program"
)

// onlyPendingTeardown decides whether a failed destroy is reported as a success.
// Over-matching would hide a real failure behind a scheduled retry, so the
// must-not-match cases matter as much as the must-match ones.
func TestOnlyPendingTeardown(t *testing.T) {
	tests := []struct {
		name  string
		types []string
		want  bool
	}{
		{
			name:  "the VPC and subnet waiting on the IP release",
			types: []string{typeNetwork, typeSubnetwork},
			want:  true,
		},
		{
			name: "everything that can be caught by the same window",
			types: []string{
				typeNetwork,
				typeSubnetwork,
				typeSvcConnection,
				"gcp:compute/instanceTemplate:InstanceTemplate",
			},
			want: true,
		},
		{
			// A destroy that finished leaves nothing; treating that as pending
			// would schedule a retry with nothing to do.
			name:  "nothing left",
			types: nil,
			want:  false,
		},
		{
			// A live Cloud Run service means the destroy failed for a reason
			// the IP-release window does not explain.
			name:  "an unrelated resource is still standing",
			types: []string{typeNetwork, "gcp:cloudrunv2/service:Service"},
			want:  false,
		},
		{
			// A stuck database is a real failure and must surface, even though
			// it would also block the network.
			name:  "a Cloud SQL instance survived",
			types: []string{"gcp:sql/databaseInstance:DatabaseInstance", typeNetwork},
			want:  false,
		},
		{
			name:  "only an unrelated resource",
			types: []string{"gcp:artifactregistry/repository:Repository"},
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := onlyPendingTeardown(tt.types); got != tt.want {
				t.Errorf("onlyPendingTeardown(%v) = %v, want %v", tt.types, got, tt.want)
			}
		})
	}
}

// The stack and its providers are not cloud resources, so they must not make a
// finished destroy look like a pending teardown.
func TestRemainingTypesIgnoresStackAndProviders(t *testing.T) {
	// Shape and type tokens taken from a real checkpoint in the CD bucket.
	const export = `{
	  "resources": [
	    {"type": "pulumi:pulumi:Stack",            "urn": "urn:pulumi:preview::myproj::pulumi:pulumi:Stack::myproj-preview"},
	    {"type": "pulumi:providers:gcp",           "urn": "urn:pulumi:preview::myproj::pulumi:providers:gcp::default"},
	    {"type": "pulumi:providers:defang-gcp",    "urn": "urn:pulumi:preview::myproj::pulumi:providers:defang-gcp::default"},
	    {"type": "gcp:compute/network:Network",    "urn": "urn:pulumi:preview::myproj::gcp:compute/network:Network::myproj-vpc"},
	    {"type": "gcp:compute/subnetwork:Subnetwork", "urn": "urn:pulumi:preview::myproj::gcp:compute/subnetwork:Subnetwork::myproj-subnet"}
	  ]
	}`
	got, err := remainingTypes([]byte(export))
	if err != nil {
		t.Fatal(err)
	}
	want := "gcp:compute/network:Network,gcp:compute/subnetwork:Subnetwork"
	if strings.Join(got, ",") != want {
		t.Errorf("remainingTypes = %v, want %q", got, want)
	}
	if !onlyPendingTeardown(got) {
		t.Error("a state holding only the VPC and subnet must classify as a pending teardown")
	}
}

// A destroy that fully succeeded leaves only the stack and providers, and must
// not be classified as pending.
func TestRemainingTypesEmptyAfterFullDestroy(t *testing.T) {
	const export = `{"resources": [
	  {"type": "pulumi:pulumi:Stack", "urn": "urn:pulumi:preview::myproj::pulumi:pulumi:Stack::myproj-preview"},
	  {"type": "pulumi:providers:gcp", "urn": "urn:pulumi:preview::myproj::pulumi:providers:gcp::default"}
	]}`
	got, err := remainingTypes([]byte(export))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("remainingTypes = %v, want empty", got)
	}
	if onlyPendingTeardown(got) {
		t.Error("a finished destroy must not classify as a pending teardown")
	}
}

// A state that cannot be read must not be mistaken for a pending teardown.
func TestRemainingTypesRejectsGarbage(t *testing.T) {
	if _, err := remainingTypes([]byte("not json")); err == nil {
		t.Error("expected an error for unparseable state")
	}
}

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
	body, err := gcpCleanupBuild("us-docker.pkg.dev/defang/cd:v2", "my-gcp-project", "cd@my-gcp-project.iam.gserviceaccount.com", "preview", "defang-cleanup-myproj-preview-20260819143045", environ)
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
	want := "CLEAN_UP_JOB_NAME=defang-cleanup-myproj-preview-20260819143045," +
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
