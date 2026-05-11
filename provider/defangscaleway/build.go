package defangscaleway

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-go-provider/infer"
)

var (
	ErrBuildFailed  = errors.New("build failed")
	ErrBuildTimeout = errors.New("build timed out")
)

// Build is a custom resource that triggers a Kaniko build via Scaleway Serverless Jobs.
type Build struct{}

type BuildInputs struct {
	// Scaleway region
	Region string `pulumi:"region"`

	// Scaleway project ID
	ProjectId string `pulumi:"projectId"`

	// S3 source URL for build context (e.g., "s3://bucket/uploads/digest")
	Source string `pulumi:"source"`

	// Destination image URL (e.g., "rg.fr-par.scw.cloud/namespace/service:tag")
	Destination string `pulumi:"destination"`

	// Dockerfile path within build context (default: "Dockerfile")
	Dockerfile *string `pulumi:"dockerfile,optional"`

	// Build target for multi-stage builds
	Target *string `pulumi:"target,optional"`

	// Build arguments
	BuildArgs map[string]string `pulumi:"buildArgs,optional"`

	// Max wait time in seconds (default: common.DefaultBuildMaxWaitTime)
	MaxWaitTime *int `pulumi:"maxWaitTime,optional"`

	// Trigger replacements when these change
	Triggers []string `pulumi:"triggers,optional"`
}

type BuildState struct {
	BuildInputs

	// The Scaleway Job Run ID
	BuildId string `pulumi:"buildId"`

	// The built image URL
	Image string `pulumi:"image"`
}

// Create starts a Kaniko build via Scaleway Serverless Jobs and waits for completion.
func (*Build) Create(
	ctx context.Context, req infer.CreateRequest[BuildInputs],
) (infer.CreateResponse[BuildState], error) {
	inputs := req.Inputs

	if req.DryRun {
		return infer.CreateResponse[BuildState]{
			ID: req.Name,
			Output: BuildState{
				BuildInputs: inputs,
			},
		}, nil
	}

	runID, err := runKanikoBuild(ctx, inputs)
	if err != nil {
		return infer.CreateResponse[BuildState]{}, fmt.Errorf("Kaniko build failed: %w", err)
	}

	return infer.CreateResponse[BuildState]{
		ID: req.Name,
		Output: BuildState{
			BuildInputs: inputs,
			BuildId:     runID,
			Image:       inputs.Destination,
		},
	}, nil
}

// Update compares inputs and re-runs the build if source or configuration changed.
func (*Build) Update(
	ctx context.Context, req infer.UpdateRequest[BuildInputs, BuildState],
) (infer.UpdateResponse[BuildState], error) {
	inputs := req.Inputs

	if req.DryRun {
		return infer.UpdateResponse[BuildState]{
			Output: BuildState{
				BuildInputs: inputs,
				BuildId:     req.State.BuildId,
				Image:       inputs.Destination,
			},
		}, nil
	}

	// Re-run the build (source reference or config changed)
	runID, err := runKanikoBuild(ctx, inputs)
	if err != nil {
		return infer.UpdateResponse[BuildState]{}, fmt.Errorf("Kaniko build failed: %w", err)
	}

	return infer.UpdateResponse[BuildState]{
		Output: BuildState{
			BuildInputs: inputs,
			BuildId:     runID,
			Image:       inputs.Destination,
		},
	}, nil
}

// Delete cleans up any resources associated with the build.
// Scaleway Serverless Jobs definitions are deleted immediately after the build
// completes (see runKanikoBuild), so there is nothing to clean up here.
func (*Build) Delete(ctx context.Context, req infer.DeleteRequest[BuildState]) error {
	return nil
}

// dockerConfigJSON generates a Docker config.json for authenticating with the
// Scaleway Container Registry. The registry uses "nologin" as the username
// and the SCW_SECRET_KEY as the password.
func dockerConfigJSON(registryHost, secretKey string) string {
	auth := base64.StdEncoding.EncodeToString([]byte("nologin:" + secretKey))
	config := map[string]any{
		"auths": map[string]any{
			registryHost: map[string]string{
				"auth": auth,
			},
		},
	}
	b, _ := json.Marshal(config)
	return string(b)
}

// registryHost extracts the registry host from a full image destination.
// e.g., "rg.fr-par.scw.cloud/namespace/image:tag" → "rg.fr-par.scw.cloud"
func registryHost(destination string) string {
	if idx := strings.IndexByte(destination, '/'); idx > 0 {
		return destination[:idx]
	}
	return destination
}

func runKanikoBuild(ctx context.Context, inputs BuildInputs) (string, error) {
	secretKey := os.Getenv("SCW_SECRET_KEY")
	client := &scwAPIClient{
		secretKey: secretKey,
		region:    inputs.Region,
	}

	dockerfile := "Dockerfile"
	if inputs.Dockerfile != nil && *inputs.Dockerfile != "" {
		dockerfile = *inputs.Dockerfile
	}

	// Build the Kaniko executor command line.
	// --force is required because Scaleway Serverless Jobs uses a sandboxed
	// runtime (like gVisor) that doesn't support chown on all files.
	kanikoCmd := []string{
		"/kaniko/executor",
		"--context=" + inputs.Source,
		"--destination=" + inputs.Destination,
		"--dockerfile=" + dockerfile,
		"--cache=true",
		"--snapshot-mode=redo",
		"--force",
		"--verbosity=info",
	}
	if inputs.Target != nil && *inputs.Target != "" {
		kanikoCmd = append(kanikoCmd, "--target="+*inputs.Target)
	}
	for k, v := range inputs.BuildArgs {
		kanikoCmd = append(kanikoCmd, fmt.Sprintf("--build-arg=%s=%s", k, v))
	}

	// Generate Docker config for Scaleway Container Registry auth
	host := registryHost(inputs.Destination)
	dockerConfig := dockerConfigJSON(host, secretKey)

	// Build the shell script that writes Docker config for registry auth,
	// then runs the Kaniko executor.
	// Write Docker config for registry auth at both KANIKO_DIR (/workspace)
	// and the default /kaniko/.docker/ location.
	shellCmd := fmt.Sprintf(
		`mkdir -p /workspace/.docker /kaniko/.docker && echo "$DOCKER_CONFIG_JSON" > /workspace/.docker/config.json && cp /workspace/.docker/config.json /kaniko/.docker/config.json && %s`,
		strings.Join(kanikoCmd, " "),
	)

	// WARNING: Scaleway Serverless Jobs API does not support secret injection.
	// These credentials are passed as plain-text environment variables and stored
	// in the job definition body. This is a known security limitation.
	// TODO: Use Scaleway Secret Manager integration when/if Scaleway adds native
	// secret injection support for Serverless Jobs.
	//
	// Environment for Kaniko:
	// - AWS SDK env vars for S3-compatible build context access
	// - DOCKER_CONFIG_JSON for registry authentication (written by script)
	// - KANIKO_SCRIPT holds the full build script, executed via eval
	env := map[string]string{
		"AWS_ACCESS_KEY_ID":          os.Getenv("AWS_ACCESS_KEY_ID"),
		"AWS_SECRET_ACCESS_KEY":      os.Getenv("AWS_SECRET_ACCESS_KEY"),
		"AWS_REGION":                 os.Getenv("AWS_REGION"),
		"AWS_EC2_METADATA_DISABLED":  "true",          // Prevent SDK from falling through to IMDS
		"S3_ENDPOINT":                fmt.Sprintf("https://s3.%s.scw.cloud", inputs.Region),
		"S3_FORCE_PATH_STYLE":        "true",
		"KANIKO_DIR":                 "/workspace",    // Use writable dir; /kaniko is read-only in sandbox
		"DOCKER_CONFIG_JSON":         dockerConfig,
		"KANIKO_SCRIPT":              shellCmd,
	}

	defID, err := client.createJobDefinition(ctx, inputs.ProjectId, sanitizeJobName(inputs.Destination), env)
	if err != nil {
		return "", fmt.Errorf("creating Kaniko job definition: %w", err)
	}
	defer func() {
		_ = client.deleteJobDefinition(ctx, defID)
	}()

	runID, err := client.runJob(ctx, defID)
	if err != nil {
		return "", fmt.Errorf("starting Kaniko job: %w", err)
	}

	maxWait := common.DefaultBuildMaxWaitTime
	if inputs.MaxWaitTime != nil {
		maxWait = *inputs.MaxWaitTime
	}
	deadline := time.Now().Add(time.Duration(maxWait) * time.Second)
	pollInterval := 5 * time.Second

	for {
		if time.Now().After(deadline) {
			return runID, fmt.Errorf("build %s: %w after %ds", runID, ErrBuildTimeout, maxWait)
		}
		if err := common.SleepWithContext(ctx, pollInterval); err != nil {
			return runID, err
		}

		state, errMsg, err := client.getJobRunStatus(ctx, runID)
		if err != nil {
			return runID, fmt.Errorf("polling build status: %w", err)
		}

		switch state {
		case "succeeded":
			return runID, nil
		case "failed", "canceled":
			return runID, fmt.Errorf("build %s: %w: %s", runID, ErrBuildFailed, errMsg)
		case "queued", "running", "pending":
			continue
		default:
			continue
		}
	}
}

// sanitizeJobName creates a safe job name from a destination string.
func sanitizeJobName(dest string) string {
	s := strings.NewReplacer("/", "-", ":", "-", ".", "-").Replace(dest)
	if len(s) > 50 {
		s = s[:50]
	}
	return strings.TrimRight(s, "-")
}

// scwAPIClient is a minimal Scaleway API client for the Serverless Jobs API.
// It lives in the provider package (not the CLI) to avoid importing the full
// Scaleway Go SDK; the provider binary runs inside the CD task container.
type scwAPIClient struct {
	secretKey string
	region    string
}

type jobDefinitionResponse struct {
	ID string `json:"id"`
}

type jobRunResponse struct {
	ID           string `json:"id"`
	State        string `json:"state"`
	ErrorMessage string `json:"error_message"`
}

func (c *scwAPIClient) baseURL() string {
	return fmt.Sprintf("https://api.scaleway.com/serverless-jobs/v1alpha1/regions/%s", c.region)
}

func (c *scwAPIClient) doRequest(ctx context.Context, method, url string, body any) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	r, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	r.Header.Set("X-Auth-Token", c.secretKey)
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Scaleway API error %d: %s", resp.StatusCode, string(respBody))
	}
	return resp, nil
}

func (c *scwAPIClient) createJobDefinition(ctx context.Context, projectID, name string, env map[string]string) (string, error) {
	// Scaleway Serverless Jobs splits the "command" string by whitespace into
	// an exec array. Only "command" and "environment_variables" are inherited
	// by job runs; "args" and "startup_command" are NOT.
	//
	// To run a multi-command shell script we use an env-var bootstrap:
	//   command: "sh -c eval$IFS$KANIKO_SCRIPT"
	// After whitespace split → exec["sh", "-c", "eval$IFS$KANIKO_SCRIPT"]
	// sh parses the script text "eval$IFS$KANIKO_SCRIPT":
	//   - $IFS expands to space/tab/newline (no literal whitespace in token)
	//   - $KANIKO_SCRIPT expands to the full build script
	//   - eval re-parses the expansion as shell code (with &&, >, etc.)
	body := map[string]any{
		"name":                  name,
		"project_id":            projectID,
		"cpu_limit":             2000,  // 2 vCPU for builds
		"memory_limit":          4096,  // 4 GB RAM for builds
		"local_storage_capacity": 10000, // 10 GB local storage for builds
		"image_uri":             "rg." + c.region + ".scw.cloud/defang-cd/kaniko-executor:patched",
		"command":               "sh -c eval$IFS$KANIKO_SCRIPT",
		"environment_variables": env,
	}

	resp, err := c.doRequest(ctx, "POST", c.baseURL()+"/job-definitions", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var def jobDefinitionResponse
	if err := json.NewDecoder(resp.Body).Decode(&def); err != nil {
		return "", fmt.Errorf("decoding job definition response: %w", err)
	}
	return def.ID, nil
}

func (c *scwAPIClient) runJob(ctx context.Context, definitionID string) (string, error) {
	resp, err := c.doRequest(ctx, "POST", c.baseURL()+"/job-definitions/"+definitionID+"/start", map[string]any{})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// The start endpoint returns {"job_runs": [...]} not a flat object
	var result struct {
		JobRuns []jobRunResponse `json:"job_runs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding job run response: %w", err)
	}
	if len(result.JobRuns) == 0 {
		return "", fmt.Errorf("no job runs returned from start")
	}
	return result.JobRuns[0].ID, nil
}

func (c *scwAPIClient) getJobRunStatus(ctx context.Context, runID string) (state, errMsg string, err error) {
	resp, err := c.doRequest(ctx, "GET", c.baseURL()+"/job-runs/"+runID, nil)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var run jobRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		return "", "", fmt.Errorf("decoding job run status: %w", err)
	}
	return run.State, run.ErrorMessage, nil
}

func (c *scwAPIClient) deleteJobDefinition(ctx context.Context, definitionID string) error {
	resp, err := c.doRequest(ctx, "DELETE", c.baseURL()+"/job-definitions/"+definitionID, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
