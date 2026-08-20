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

// The timestamp must be read as UTC: parsing it in the local zone would shift
// the cut-off and skip state generations that predate the job.
func TestCleanupJobCreatedAtIsUTC(t *testing.T) {
	t.Setenv("TZ", "America/Vancouver")
	got, err := cleanupJobCreatedAt("defang-cleanup-p-s-20260819143045")
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 8, 19, 14, 30, 45, 0, time.UTC); !got.Equal(want) {
		t.Errorf("createdAt = %s, want %s", got, want)
	}
}

func TestCleanupJobCreatedAtRejectsGarbage(t *testing.T) {
	for _, jobID := range []string{"defang-cleanup-p-s-notatime", "nodashes", ""} {
		if _, err := cleanupJobCreatedAt(jobID); err == nil {
			t.Errorf("cleanupJobCreatedAt(%q) = nil error, want error", jobID)
		}
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

// The first fire must not be immediate, or the job runs before the resources
// referencing the network are gone, and it must stay inside the 2h period.
func TestCleanupFirstRunDelay(t *testing.T) {
	if cleanupFirstRunDelay <= 0 || cleanupFirstRunDelay >= 2*time.Hour {
		t.Errorf("cleanupFirstRunDelay = %s, want between 0 and 2h", cleanupFirstRunDelay)
	}
}

func TestGcpCleanupBuild(t *testing.T) {
	environ := []string{
		"PROJECT=myproj",
		"STACK=preview",
		"GCLOUD_PROJECT=my-gcp-project",              // kept: GCLOUD_ prefix
		"PULUMI_BACKEND_URL=gs://defang-cd",          // kept: PULUMI_ prefix, needed to find the state
		"DEFANG_STATES_UPLOAD_URL=https://presigned", // dropped: expires before the job fires
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
	if strings.Join(step.Args, " ") != "cleanup" {
		t.Errorf("args = %v, want [cleanup]", step.Args)
	}
	want := "CLEAN_UP_JOB_NAME=defang-cleanup-myproj-preview-20260819143045," +
		"GCLOUD_PROJECT=my-gcp-project,PROJECT=myproj,PULUMI_BACKEND_URL=gs://defang-cd,STACK=preview"
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
	if strings.Join(build.Tags, ",") != "defang-cd,defang-cleanup,preview" {
		t.Errorf("tags = %v", build.Tags)
	}
}

func TestStackStateObject(t *testing.T) {
	t.Setenv("PULUMI_BACKEND_URL", "gs://defang-cd-bucket")
	t.Setenv("DEFANG_STATE_URL", "")
	bucket, object, err := stackStateObject("myproj", "preview")
	if err != nil {
		t.Fatal(err)
	}
	if bucket != "defang-cd-bucket" {
		t.Errorf("bucket = %q", bucket)
	}
	if want := ".pulumi/stacks/myproj/preview.json"; object != want {
		t.Errorf("object = %q, want %q", object, want)
	}
}

// DEFANG_STATE_URL is the fallback the CLI always sets; PULUMI_BACKEND_URL wins
// because it is what Pulumi itself wrote the checkpoint to.
func TestStackStateObjectFallsBackToStateURL(t *testing.T) {
	t.Setenv("PULUMI_BACKEND_URL", "")
	t.Setenv("DEFANG_STATE_URL", "gs://defang-cd-bucket")
	bucket, object, err := stackStateObject("myproj", "preview")
	if err != nil {
		t.Fatal(err)
	}
	if bucket != "defang-cd-bucket" || object != ".pulumi/stacks/myproj/preview.json" {
		t.Errorf("bucket = %q, object = %q", bucket, object)
	}
}

func TestStackStateObjectRequiresBackend(t *testing.T) {
	t.Setenv("PULUMI_BACKEND_URL", "")
	t.Setenv("DEFANG_STATE_URL", "")
	if _, _, err := stackStateObject("myproj", "preview"); err == nil {
		t.Error("expected an error when no backend URL is set")
	}
}

func TestParseGCSURL(t *testing.T) {
	tests := []struct {
		raw     string
		bucket  string
		object  string
		wantErr bool
	}{
		{raw: "gs://bucket/a/b.json", bucket: "bucket", object: "a/b.json"},
		{raw: "gs://bucket/a/b%3D%3D.json", bucket: "bucket", object: "a/b==.json"},
		{raw: "s3://bucket/a", wantErr: true},
		{raw: "gs://bucket", wantErr: true},
		{raw: "gs:///object", wantErr: true},
		{raw: "gs://bucket/", wantErr: true},
	}
	for _, tt := range tests {
		bucket, object, err := parseGCSURL(tt.raw)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseGCSURL(%q) = nil error, want error", tt.raw)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseGCSURL(%q): %v", tt.raw, err)
			continue
		}
		if bucket != tt.bucket || object != tt.object {
			t.Errorf("parseGCSURL(%q) = %q, %q; want %q, %q", tt.raw, bucket, object, tt.bucket, tt.object)
		}
	}
}

// The network id from Pulumi state is a partial path, while GCP returns full
// self-links. A prefix-sharing network name must not match.
func TestReferencesNetwork(t *testing.T) {
	const networkID = "projects/my-gcp-project/global/networks/myproj-vpc"
	tests := []struct {
		selfLink string
		want     bool
	}{
		{"https://www.googleapis.com/compute/v1/projects/my-gcp-project/global/networks/myproj-vpc", true},
		{"https://www.googleapis.com/compute/v1/projects/my-gcp-project/global/networks/other-vpc", false},
		// Same suffix, different network: must not match.
		{"https://www.googleapis.com/compute/v1/projects/my-gcp-project/global/networks/notmyproj-vpc", false},
		// Same name in a different project: must not match.
		{"https://www.googleapis.com/compute/v1/projects/other-project/global/networks/myproj-vpc", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := referencesNetwork(tt.selfLink, networkID); got != tt.want {
			t.Errorf("referencesNetwork(%q) = %v, want %v", tt.selfLink, got, tt.want)
		}
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

// A state backup with no network is terminal: the job must be able to
// recognise it and delete itself instead of retrying every two hours for ever.
func TestNoNetworkInStateIsIdentifiable(t *testing.T) {
	err := fmt.Errorf("%w: looked for %s in gs://b/o", errNoNetworkInState, gcpNetworkType)
	if !errors.Is(err, errNoNetworkInState) {
		t.Error("wrapped errNoNetworkInState is not identifiable with errors.Is")
	}
	if errors.Is(errors.New("delete blocked by a dependent resource"), errNoNetworkInState) {
		t.Error("an unrelated error must not match errNoNetworkInState")
	}
}

func TestNetworkFromStateDecode(t *testing.T) {
	const checkpoint = `{
	  "version": 3,
	  "checkpoint": {
	    "stack": "organization/myproj/preview",
	    "latest": {
	      "resources": [
	        {"type": "pulumi:pulumi:Stack", "id": ""},
	        {"type": "gcp:compute/subnetwork:Subnetwork", "id": "projects/p/regions/us-central1/subnetworks/s"},
	        {"type": "gcp:compute/network:Network", "id": "projects/p/global/networks/myproj-vpc"}
	      ]
	    }
	  }
	}`
	var state pulumiState
	if err := json.Unmarshal([]byte(checkpoint), &state); err != nil {
		t.Fatal(err)
	}
	var found string
	for _, res := range state.Checkpoint.Latest.Resources {
		if res.Type == gcpNetworkType {
			found = res.Id
		}
	}
	if want := "projects/p/global/networks/myproj-vpc"; found != want {
		t.Errorf("network id = %q, want %q", found, want)
	}
}
