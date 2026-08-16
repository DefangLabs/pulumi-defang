package program

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudscheduler"
	gcpconfig "github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/config"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// On GCP the CD runs as a Cloud Build (see ByocGcp.runCdCommand in the CLI),
// so the self-destruct trigger is a Cloud Scheduler job that POSTs the same
// builds.create request the CLI submits, with args ["down"]. The CD service
// account already carries everything this needs: cloudscheduler.admin ("for
// scheduling clean up jobs"), cloudbuild.builds.editor, and project-level
// serviceAccountUser — and the CLI enables cloudscheduler.googleapis.com at
// setup.
const metadataSAEmailURL = "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/email"

// createGCPSelfDestruct schedules this stack's own `defang cd down`. Like the
// AWS variant it does not depend on the project component, so even a deploy
// that fails halfway still cleans itself up at the TTL.
func createGCPSelfDestruct(pctx *pulumi.Context, cf *compose.Project, ttl time.Duration, cdImage string, opts ...pulumi.ResourceOption) error {
	if cdImage == "" {
		return fmt.Errorf("defang:ttl requires defang:cdImage (the DEFANG_CD_IMAGE env var) to re-run this image for the down")
	}
	projectID := gcpconfig.GetProject(pctx)
	if projectID == "" {
		return fmt.Errorf("defang:ttl requires gcp:project in Pulumi config")
	}

	// The build must run as the same service account as this run; Cloud Build
	// exposes it via the metadata server. Local debug runs have no metadata
	// server — nothing to clone a down run from.
	saEmail, err := metadataServiceAccountEmail(pctx.Context())
	if err != nil {
		return fmt.Errorf("defang:ttl requires the CD to run in Cloud Build: %w", err)
	}

	fireAt := time.Now().Add(ttl)
	body, err := gcpSelfDestructBuild(cdImage, projectID, saEmail, pctx.Stack(), os.Environ())
	if err != nil {
		return err
	}

	_ = pctx.Log.Info(fmt.Sprintf("self-destruct: this stack will run `defang cd down` on itself at %s (ttl %s); redeploying extends it",
		fireAt.UTC().Format(time.RFC3339), ttl), nil)

	_, err = cloudscheduler.NewJob(pctx, "self-destruct", &cloudscheduler.JobArgs{
		Description: pulumi.Sprintf("defang self-destruct for %s/%s", cf.Name, pctx.Stack()),
		Region:      pulumi.String(gcpconfig.GetRegion(pctx)),
		Schedule:    pulumi.String(selfDestructCron(fireAt)),
		TimeZone:    pulumi.String("Etc/UTC"),
		HttpTarget: cloudscheduler.JobHttpTargetArgs{
			HttpMethod: pulumi.String("POST"),
			Uri:        pulumi.String(fmt.Sprintf("https://cloudbuild.googleapis.com/v1/projects/%s/builds", projectID)),
			Headers:    pulumi.StringMap{"Content-Type": pulumi.String("application/json")},
			Body:       pulumi.String(base64.StdEncoding.EncodeToString(body)),
			OauthToken: cloudscheduler.JobHttpTargetOauthTokenArgs{
				ServiceAccountEmail: pulumi.String(saEmail),
				Scope:               pulumi.String("https://www.googleapis.com/auth/cloud-platform"),
			},
		},
	}, opts...)
	return err
}

// gcpSelfDestructBuild renders the builds.create request body — the same
// build the CLI submits (Gcp.RunCloudBuild), with args ["down"].
func gcpSelfDestructBuild(cdImage, projectID, saEmail, stack string, environ []string) ([]byte, error) {
	env := SelfDestructEnv(environ)
	envList := make([]string, 0, len(env))
	for _, k := range slices.Sorted(maps.Keys(env)) {
		envList = append(envList, k+"="+env[k])
	}
	build := map[string]any{
		"steps": []map[string]any{{
			"name": cdImage,
			"args": []string{"down"},
			"env":  envList,
		}},
		"options": map[string]any{
			// Custom-service-account builds require an explicit logging mode;
			// mirrors RunCloudBuild in the CLI.
			"logging":                 "CLOUD_LOGGING_ONLY",
			"enableStructuredLogging": true,
		},
		"timeout":        "3600s",
		"tags":           []string{"defang-cd", "defang-self-destruct", stack},
		"serviceAccount": fmt.Sprintf("projects/%s/serviceAccounts/%s", projectID, saEmail),
	}
	return json.Marshal(build)
}

// metadataServiceAccountEmail asks the GCE metadata server which service
// account this workload runs as.
func metadataServiceAccountEmail(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataSAEmailURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metadata server returned %s", resp.Status)
	}
	email, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return "", err
	}
	sa := strings.TrimSpace(string(email))
	if sa == "" || !strings.Contains(sa, "@") {
		return "", fmt.Errorf("metadata server returned no service account email")
	}
	return sa, nil
}
