package program

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	if build.ServiceAccount != "projects/my-gcp-project/serviceAccounts/cd@my-gcp-project.iam.gserviceaccount.com" {
		t.Errorf("serviceAccount = %q", build.ServiceAccount)
	}
}

func TestMetadataServiceAccountEmail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Metadata-Flavor") != "Google" {
			http.Error(w, "missing flavor", http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte("cd@proj.iam.gserviceaccount.com"))
	}))
	defer srv.Close()

	// The official metadata client honors GCE_METADATA_HOST.
	t.Setenv("GCE_METADATA_HOST", strings.TrimPrefix(srv.URL, "http://"))

	sa, err := metadataServiceAccountEmail(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sa != "cd@proj.iam.gserviceaccount.com" {
		t.Errorf("sa = %q", sa)
	}
}
