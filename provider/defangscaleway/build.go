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

	// Build the Kaniko executor command line
	kanikoCmd := []string{
		"/kaniko/executor",
		"--context=" + inputs.Source,
		"--destination=" + inputs.Destination,
		"--dockerfile=" + dockerfile,
		"--cache=true",
		"--snapshot-mode=redo",
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

	// Environment for Kaniko:
	// - AWS SDK env vars for S3-compatible build context access
	// - DOCKER_CONFIG_JSON for registry authentication (written by entrypoint)
	env := map[string]string{
		"AWS_ACCESS_KEY_ID":     os.Getenv("AWS_ACCESS_KEY_ID"),
		"AWS_SECRET_ACCESS_KEY": os.Getenv("AWS_SECRET_ACCESS_KEY"),
		"AWS_REGION":            os.Getenv("AWS_REGION"),
		"S3_ENDPOINT":           fmt.Sprintf("https://s3.%s.scw.cloud", inputs.Region),
		"S3_FORCE_PATH_STYLE":   "true",
		"DOCKER_CONFIG_JSON":    dockerConfig,
	}

	// Build the shell script that writes Docker config for registry auth,
	// then runs the Kaniko executor. Scaleway Serverless Jobs wraps the
	// command string in `sh -c` internally, so we must NOT add our own
	// `sh -c '...'` wrapper (which caused quoting issues).
	shellCmd := fmt.Sprintf(
		`mkdir -p /kaniko/.docker && echo "$DOCKER_CONFIG_JSON" > /kaniko/.docker/config.json && %s`,
		strings.Join(kanikoCmd, " "),
	)

	defID, err := client.createJobDefinition(ctx, inputs.ProjectId, sanitizeJobName(inputs.Destination), env, shellCmd)
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

func (c *scwAPIClient) createJobDefinition(ctx context.Context, projectID, name string, env map[string]string, command string) (string, error) {
	// Use "command" to override the image ENTRYPOINT (kaniko executor) with
	// a plain shell, and "startup_command" to set CMD with -c and the script.
	// Scaleway splits "command" by whitespace into an exec array, so shell
	// operators (&&, >, etc.) in the script would break if placed there.
	// The startup_command array preserves the script as a single argument.
	body := map[string]any{
		"name":                  name,
		"project_id":            projectID,
		"cpu_limit":             2000, // 2 vCPU for builds
		"memory_limit":          4096, // 4 GB RAM for builds
		"image_uri":             "gcr.io/kaniko-project/executor:debug",
		"command":               "sh",
		"startup_command":       []string{"-c", command},
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

	var run jobRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		return "", fmt.Errorf("decoding job run response: %w", err)
	}
	return run.ID, nil
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
