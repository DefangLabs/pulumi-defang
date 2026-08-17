package program

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestGcpSelfDestructBuild(t *testing.T) {
	environ := []string{
		"PROJECT=myproj",
		"STACK=preview",
		"GCLOUD_PROJECT=my-gcp-project", // kept: GCLOUD_ prefix
		"PATH=/usr/bin",                 // dropped
	}
	body, err := gcpSelfDestructBuild("us-docker.pkg.dev/defang/cd:v2", "my-gcp-project", "cd@my-gcp-project.iam.gserviceaccount.com", "preview", environ)
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
	if step.Name != "us-docker.pkg.dev/defang/cd:v2" || strings.Join(step.Args, " ") != "down" {
		t.Errorf("step = %+v", step)
	}
	if strings.Join(step.Env, ",") != "GCLOUD_PROJECT=my-gcp-project,PROJECT=myproj,STACK=preview" {
		t.Errorf("env = %v", step.Env)
	}
	if build.Options.Logging != "CLOUD_LOGGING_ONLY" {
		t.Errorf("logging = %q", build.Options.Logging)
	}
	if want := fmt.Sprintf("%ds", int(CdTimeout.Seconds())); build.Timeout != want {
		t.Errorf("timeout = %q, want %q", build.Timeout, want)
	}
	if build.ServiceAccount != "projects/my-gcp-project/serviceAccounts/cd@my-gcp-project.iam.gserviceaccount.com" {
		t.Errorf("serviceAccount = %q", build.ServiceAccount)
	}
}
