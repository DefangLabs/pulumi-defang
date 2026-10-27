package defangscaleway

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-go-provider/infer"
)

var (
	ErrBuildFailed           = errors.New("build failed")
	ErrBuildTimeout          = errors.New("build timed out")
	errScalewayAPI           = errors.New("scaleway API error")
	errMissingAWSCredentials = errors.New("AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY not set")
)

// defaultBuildInstanceType is the Compute Instance commercial type used for
// builds. It must have non-zero quota in the target Scaleway project/account:
// new Scaleway projects commonly start with zero Instance quota for every
// commercial type until identity/payment is verified, which surfaces as a
// clear "quota exceeded (0/0)" API error rather than a silent failure.
const defaultBuildInstanceType = "STANDARD3-X2C-8G"

// buildImageLabel is Scaleway's official "Docker" Marketplace InstantApp
// image. It ships Docker + buildx already installed and running, is
// maintained by Scaleway itself (not a Defang-authored artifact a BYOC
// customer would otherwise have no reason to trust), and the Instance API
// resolves this label directly without a separate Marketplace lookup.
const buildImageLabel = "docker"

// Build is a custom resource that builds a container image from a Compose
// service's build context, on an ephemeral Scaleway Compute Instance.
//
// Scaleway Serverless Jobs (the previous execution backend) run under gVisor
// with privilege escalation disabled and all root capabilities dropped. That
// is enough to break both candidate "Kaniko is deprecated" replacements:
// rootless BuildKit fails at daemon boot (needs newuidmap, which needs
// privilege escalation), and Kaniko itself fails unpacking any ordinary base
// image's rootfs (chown on files like /etc/shadow is denied) — this is not
// specific to this provider's Dockerfiles, it reproduces with a one-line
// `FROM alpine:3.19` + `RUN echo`. Neither tool is viable inside that
// sandbox, independent of which one's "dead upstream" status is weighed.
//
// A plain Compute Instance has none of those restrictions (it's a normal
// kernel, not a locked-down container sandbox), so the build step now runs
// there instead: boot the official Docker Marketplace image (no custom
// image baked by Defang — see buildImageLabel), drive the whole build via a
// single cloud-init boot script (fetch context, `docker buildx build
// --push`, report result), then terminate the instance.
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

	// The Scaleway Instance ID that ran the build (terminated by the time
	// Create/Update returns; kept for diagnostics).
	BuildId string `pulumi:"buildId"`

	// The built image URL
	Image string `pulumi:"image"`

	// LogsEndpoint is the Cockpit logs data source this build's output was
	// pushed to. It is a dedicated "custom"-origin data source created per
	// build: the project's auto-provisioned data source only accepts writes
	// from Scaleway's own managed services (confirmed: pushing to it
	// directly is rejected with a bare 403), so builds need their own.
	// Deleted by the time Create/Update returns; kept for diagnostics and
	// for whoever wires this into the CLI's build-log support next
	// (DefangLabs/defang#2105 currently stubs out Scaleway build logs
	// entirely) — that still needs a decision on how this value reaches
	// published deployment metadata, so it is not threaded any further than
	// this resource's own state yet.
	LogsEndpoint string `pulumi:"logsEndpoint,optional"`

	// LogSelector is the LogQL stream selector for this build's log lines
	// within LogsEndpoint (e.g. `{source="defang-build-<id>"}`).
	LogSelector string `pulumi:"logSelector,optional"`
}

// Create starts a build on an ephemeral Compute Instance and waits for completion.
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

	result, err := runInstanceBuild(ctx, inputs)
	if err != nil {
		return infer.CreateResponse[BuildState]{}, fmt.Errorf("build failed: %w", err)
	}

	return infer.CreateResponse[BuildState]{
		ID: req.Name,
		Output: BuildState{
			BuildInputs:  inputs,
			BuildId:      result.instanceID,
			Image:        inputs.Destination,
			LogsEndpoint: result.logsEndpoint,
			LogSelector:  result.logSelector,
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
				BuildInputs:  inputs,
				BuildId:      req.State.BuildId,
				Image:        inputs.Destination,
				LogsEndpoint: req.State.LogsEndpoint,
				LogSelector:  req.State.LogSelector,
			},
		}, nil
	}

	// Re-run the build (source reference or config changed)
	result, err := runInstanceBuild(ctx, inputs)
	if err != nil {
		return infer.UpdateResponse[BuildState]{}, fmt.Errorf("build failed: %w", err)
	}

	return infer.UpdateResponse[BuildState]{
		Output: BuildState{
			BuildInputs:  inputs,
			BuildId:      result.instanceID,
			Image:        inputs.Destination,
			LogsEndpoint: result.logsEndpoint,
			LogSelector:  result.logSelector,
		},
	}, nil
}

// Delete cleans up any resources associated with the build. The Instance,
// its Cockpit logs data source, and its Cockpit token are all torn down
// synchronously at the end of Create/Update (see runInstanceBuild), so
// there is nothing left to clean up here.
func (*Build) Delete(ctx context.Context, req infer.DeleteRequest[BuildState]) error {
	return nil
}

// registryHost extracts the registry host from a full image destination.
// e.g., "rg.fr-par.scw.cloud/namespace/image:tag" → "rg.fr-par.scw.cloud"
func registryHost(destination string) string {
	if idx := strings.IndexByte(destination, '/'); idx > 0 {
		return destination[:idx]
	}
	return destination
}

// defaultZone picks the single zone this build runs in. BuildInputs only
// carries a region (matching the Job-based Source/Destination, which are
// also region-scoped), so this is a simplifying "first zone in the region"
// choice rather than something callers can override yet.
func defaultZone(region string) string {
	return region + "-1"
}

// shQuote single-quotes s for safe embedding in a POSIX shell command line.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

var errInvalidSource = errors.New("invalid build context source")

// parseS3Source splits an "s3://bucket/key" URL into its bucket and key.
func parseS3Source(source string) (string, string, error) {
	const prefix = "s3://"
	if !strings.HasPrefix(source, prefix) {
		return "", "", fmt.Errorf("%w: %q: expected an s3:// URL", errInvalidSource, source)
	}
	rest := source[len(prefix):]
	idx := strings.IndexByte(rest, '/')
	if idx <= 0 || idx == len(rest)-1 {
		return "", "", fmt.Errorf("%w: %q: expected s3://bucket/key", errInvalidSource, source)
	}
	return rest[:idx], rest[idx+1:], nil
}

// buildScript renders the plain shell script (not a #cloud-config YAML
// document — a bare "#!/bin/bash" is cloud-init's "user-data script"
// format, and executes exactly as written with no YAML indentation/quoting
// concerns) that the build Instance runs at boot:
//  1. fetch the build context from Object Storage (S3-compatible, signed
//     the same way curl's own --aws-sigv4 support signs any S3 request —
//     no AWS SDK or CLI needed on the instance),
//  2. `docker buildx build --push` it to the destination registry,
//  3. relay the build's output, line by line, to a dedicated Cockpit logs
//     data source (the project's auto-provisioned one rejects writes from
//     anything but Scaleway's own managed services), finishing with a
//     RESULT:SUCCESS/RESULT:FAILURE line that doubles as the completion
//     signal for waitForInstanceBuild,
//  4. power off.
func buildScript(
	inputs BuildInputs,
	bucket, key string,
	registrySecret string,
	logsEndpoint, logsToken, logSource string,
) (string, error) {
	dockerfile := "Dockerfile"
	if inputs.Dockerfile != nil && *inputs.Dockerfile != "" {
		dockerfile = *inputs.Dockerfile
	}

	var buildArgs strings.Builder
	for k, v := range common.Sorted(inputs.BuildArgs) {
		buildArgs.WriteString(" --build-arg " + shQuote(k+"="+v))
	}

	target := ""
	if inputs.Target != nil && *inputs.Target != "" {
		target = " --target " + shQuote(*inputs.Target)
	}

	awsAccessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	awsSecretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if awsAccessKeyID == "" || awsSecretAccessKey == "" {
		return "", errMissingAWSCredentials
	}
	s3URL := fmt.Sprintf("https://s3.%s.scw.cloud/%s/%s", inputs.Region, bucket, key)

	var b strings.Builder
	b.WriteString("#!/bin/bash\n")
	b.WriteString("set -u\n\n")

	// A per-line Loki push, not a batched/continuous shipper: good enough to
	// get the full build log and a completion signal into Cockpit without
	// the complexity of a real log-tailing agent. See the Build doc comment
	// for the tradeoffs recorded around this.
	//
	// The -d argument is built by concatenating single- and double-quoted
	// bash segments (JSON literal text in single quotes, $ts/$esc expansion
	// in double quotes) rather than escaping embedded double quotes inside
	// one double-quoted string: much easier to get right than a wall of
	// backslashes, since single-quoted text needs no escaping at all here.
	const lokiPushTemplate = `push_log() {
  local ts; ts=$(date +%%s%%N)
  local esc; esc=$(python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$1")
  curl -s -o /dev/null -u %s -H 'Content-Type: application/json' \
    -d '{"streams":[{"stream":{"source":"%s"},"values":[["'"$ts"'",'"$esc"']]}]}' \
    %s/loki/api/v1/push >/dev/null 2>&1
}

`
	fmt.Fprintf(&b, lokiPushTemplate, shQuote("api_key:"+logsToken), logSource, shQuote(logsEndpoint))

	b.WriteString("push_log 'defang-build: fetching build context'\n")
	b.WriteString("for i in 1 2 3 4 5; do\n")
	// -H x-amz-content-sha256 is required explicitly: curl versions differ on
	// whether --aws-sigv4 sends it automatically for a bodyless request, and
	// Scaleway's S3-compatible endpoint rejects the request outright without
	// it ("Required header missing: X-Amz-Content-Sha256", confirmed live
	// against the older curl on Scaleway's official Docker Marketplace
	// image). The AWS "UNSIGNED-PAYLOAD" sentinel is not honored here (it
	// produced "SignatureDoesNotMatch", also confirmed live) — curl's own
	// SigV4 signing still signs the real (empty, for a bodyless GET) payload
	// hash, so the header must carry that same value: the well-known SHA-256
	// of the empty string.
	// SHA-256 of the empty string, per RFC: the well-known constant value.
	const emptyPayloadSHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	const curlFetchTemplate = "  CODE=$(curl -s --aws-sigv4 %s --user %s" +
		" -H 'x-amz-content-sha256: " + emptyPayloadSHA256 + "'" +
		" %s -o /root/context.tar.gz -w '%%{http_code}' 2>/root/context-fetch.log)\n"
	fmt.Fprintf(&b, curlFetchTemplate,
		shQuote("aws:amz:"+inputs.Region+":s3"),
		shQuote(awsAccessKeyID+":"+awsSecretAccessKey),
		shQuote(s3URL),
	)
	b.WriteString("  if [ \"$CODE\" = \"200\" ]; then break; fi\n")
	const contextFetchFailureLog = "  push_log \"defang-build: context fetch attempt $i got HTTP $CODE: " +
		"$(tr '\\n' ' ' < /root/context.tar.gz | head -c 300) $(tr '\\n' ' ' < /root/context-fetch.log)\"\n"
	b.WriteString(contextFetchFailureLog)
	b.WriteString("  sleep 3\n")
	b.WriteString("done\n")
	b.WriteString("if [ \"$CODE\" != \"200\" ]; then push_log 'RESULT:FAILURE'; poweroff; exit 1; fi\n")
	b.WriteString("mkdir -p /root/buildctx\n")
	b.WriteString("tar xzf /root/context.tar.gz -C /root/buildctx\n\n")

	fmt.Fprintf(&b, "docker login %s -u nologin -p %s >/root/login.log 2>&1\n",
		shQuote(registryHost(inputs.Destination)), shQuote(registrySecret))
	b.WriteString("push_log \"docker login: $(tr '\\n' ' ' < /root/login.log)\"\n\n")

	b.WriteString("cd /root/buildctx\n")
	fmt.Fprintf(&b, "docker buildx build --progress=plain -f %s%s%s -t %s --push . > /root/build.log 2>&1\n",
		shQuote(dockerfile), target, buildArgs.String(), shQuote(inputs.Destination))
	b.WriteString("echo \"BUILD_EXIT=$?\" >> /root/build.log\n\n")

	b.WriteString("while IFS= read -r line; do push_log \"$line\"; done < /root/build.log\n")
	b.WriteString("if grep -q '^BUILD_EXIT=0$' /root/build.log; then\n")
	b.WriteString("  push_log 'RESULT:SUCCESS'\n")
	b.WriteString("else\n")
	b.WriteString("  push_log 'RESULT:FAILURE'\n")
	b.WriteString("fi\n")
	b.WriteString("poweroff\n")

	return b.String(), nil
}

type instanceBuildResult struct {
	instanceID   string
	logsEndpoint string
	logSelector  string
}

func runInstanceBuild(ctx context.Context, inputs BuildInputs) (instanceBuildResult, error) {
	secretKey := os.Getenv("SCW_SECRET_KEY")
	client := &scwAPIClient{secretKey: secretKey, region: inputs.Region}
	zone := defaultZone(inputs.Region)

	bucket, key, err := parseS3Source(inputs.Source)
	if err != nil {
		return instanceBuildResult{}, err
	}

	name := sanitizeResourceName(inputs.Destination)
	logSource := "defang-build-" + name

	dataSourceID, logsEndpoint, err := client.createLogsDataSource(ctx, inputs.ProjectId, name+"-logs")
	if err != nil {
		return instanceBuildResult{}, fmt.Errorf("creating Cockpit logs data source: %w", err)
	}
	// Retried, not best-effort-once: deleting a Cockpit resource immediately
	// after it saw recent write activity occasionally fails transiently
	// (observed live — a manual retry moments later always succeeded), and a
	// silently-failed cleanup means a leftover data source/token/instance in
	// the customer's own account on every such build.
	defer retryCleanup(ctx, func(ctx context.Context) error { return client.deleteLogsDataSource(ctx, dataSourceID) })

	tokenID, logsToken, err := client.createLogsToken(ctx, inputs.ProjectId, name+"-logs")
	if err != nil {
		return instanceBuildResult{}, fmt.Errorf("creating Cockpit logs token: %w", err)
	}
	defer retryCleanup(ctx, func(ctx context.Context) error { return client.deleteLogsToken(ctx, tokenID) })

	script, err := buildScript(inputs, bucket, key, secretKey, logsEndpoint, logsToken, logSource)
	if err != nil {
		return instanceBuildResult{}, err
	}

	serverID, err := client.createServer(ctx, zone, inputs.ProjectId, name)
	if err != nil {
		return instanceBuildResult{}, fmt.Errorf("creating build instance: %w", err)
	}
	defer retryCleanup(ctx, func(ctx context.Context) error { return client.terminateServer(ctx, zone, serverID) })

	if err := client.setCloudInit(ctx, zone, serverID, script); err != nil {
		return instanceBuildResult{}, fmt.Errorf("setting build instance cloud-init: %w", err)
	}
	if err := client.powerOnServer(ctx, zone, serverID); err != nil {
		return instanceBuildResult{}, fmt.Errorf("starting build instance: %w", err)
	}

	logSelector := fmt.Sprintf("{source=%q}", logSource)

	maxWait := common.DefaultBuildMaxWaitTime
	if inputs.MaxWaitTime != nil {
		maxWait = *inputs.MaxWaitTime
	}
	if err := waitForInstanceBuild(ctx, logsEndpoint, logsToken, logSelector, maxWait); err != nil {
		return instanceBuildResult{}, err
	}

	return instanceBuildResult{instanceID: serverID, logsEndpoint: logsEndpoint, logSelector: logSelector}, nil
}

// retryCleanup retries a best-effort teardown call a few times with a short
// backoff before giving up, since a delete immediately after recent write
// activity on the same resource has been observed to fail transiently (see
// the callers in runInstanceBuild). It never returns an error: a cleanup
// call runs in a defer, and the resource it fails to remove is a cost/tidiness
// concern, not a build-correctness one.
func retryCleanup(ctx context.Context, fn func(context.Context) error) {
	const attempts = 3
	for i := range attempts {
		if err := fn(ctx); err == nil {
			return
		}
		if i < attempts-1 {
			_ = common.SleepWithContext(ctx, 2*time.Second)
		}
	}
}

// waitForInstanceBuild polls the build's Cockpit log stream for the
// RESULT:SUCCESS/RESULT:FAILURE line buildScript writes as its last action,
// which doubles as the completion signal: no separate Object Storage
// marker or job-status API is needed, since the logs are already being
// queried for the raw build output.
func waitForInstanceBuild(ctx context.Context, logsEndpoint, logsToken, logSelector string, maxWait int) error {
	deadline := time.Now().Add(time.Duration(maxWait) * time.Second)
	pollInterval := 3 * time.Second
	start := time.Now().Add(-time.Minute) // tolerate clock skew between here and the instance

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("build %w after %ds", ErrBuildTimeout, maxWait)
		}
		if err := common.SleepWithContext(ctx, pollInterval); err != nil {
			return err
		}

		lines, err := queryLoki(ctx, logsEndpoint, logsToken, logSelector, start, time.Now())
		if err != nil {
			return fmt.Errorf("polling build logs: %w", err)
		}
		for _, line := range lines {
			switch {
			case strings.Contains(line, "RESULT:SUCCESS"):
				return nil
			case strings.Contains(line, "RESULT:FAILURE"):
				return fmt.Errorf("%w: see build logs at %s (selector %s)", ErrBuildFailed, logsEndpoint, logSelector)
			}
		}
	}
}

// sanitizeResourceName creates a safe, bounded, unique resource name from a
// destination string, for use as the build's Instance/data-source/token
// name. Bounded to 40 runes (not e.g. Cockpit's 50-rune data source name
// limit) so callers have room to append a short suffix like "-logs". Two
// destinations sharing a long common prefix (e.g. the same registry host)
// would otherwise truncate to the same name — Cockpit rejects a duplicate
// data source name outright (confirmed live: "resource already exists") —
// so, like sanitizeRepoName in the GCP provider, a long name is truncated
// with a hash of the full original name appended rather than cut blind.
func sanitizeResourceName(dest string) string {
	original := dest
	s := strings.NewReplacer("/", "-", ":", "-", ".", "-").Replace(dest)
	if len(s) > 40 {
		suffix := "-" + resourceNameHash(original)
		s = strings.TrimRight(s[:40-len(suffix)], "-") + suffix
	}
	return strings.TrimRight(s, "-")
}

func resourceNameHash(name string) string {
	digest := sha256.Sum256([]byte(name))
	return hex.EncodeToString(digest[:])[:8]
}

// scwAPIClient is a minimal Scaleway API client for the Instance and
// Cockpit APIs. It lives in the provider package (not the CLI) to avoid
// importing the full Scaleway Go SDK; the provider binary runs inside the
// CD task container.
type scwAPIClient struct {
	secretKey string
	region    string
}

func (c *scwAPIClient) instanceBaseURL(zone string) string {
	return "https://api.scaleway.com/instance/v1/zones/" + zone
}

func (c *scwAPIClient) cockpitBaseURL() string {
	return "https://api.scaleway.com/cockpit/v1/regions/" + c.region
}

func (c *scwAPIClient) doRequest(
	ctx context.Context, method, url string, body any, contentType string,
) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		switch v := body.(type) {
		case string:
			reqBody = strings.NewReader(v)
		default:
			b, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("marshaling request: %w", err)
			}
			reqBody = bytes.NewReader(b)
			contentType = "application/json"
		}
	}

	//nolint:gosec // G704: url is always built from this client's own fixed Scaleway API paths, never external input
	r, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	r.Header.Set("X-Auth-Token", c.secretKey)
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}

	resp, err := http.DefaultClient.Do(r) //nolint:gosec // G704: see above
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		defer closeBody(resp.Body)
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d: %s", errScalewayAPI, resp.StatusCode, string(respBody))
	}
	return resp, nil
}

func (c *scwAPIClient) createServer(ctx context.Context, zone, projectID, name string) (string, error) {
	body := map[string]any{
		"name":                name,
		"commercial_type":     defaultBuildInstanceType,
		"image":               buildImageLabel,
		"project":             projectID,
		"dynamic_ip_required": true,
	}
	resp, err := c.doRequest(ctx, "POST", c.instanceBaseURL(zone)+"/servers", body, "")
	if err != nil {
		// "quotas_exceeded" here means the Scaleway account/project itself has
		// zero Instance quota, not that defaultBuildInstanceType was a bad
		// choice: every commercial type returns the same (0/0) until the
		// account's payment method and identity are verified (confirmed
		// against Scaleway's own docs) — there is no quota-exempt type to
		// fall back to, so the fix has to be pointing the customer at
		// verification, not picking a different instance type.
		if strings.Contains(err.Error(), "quotas_exceeded") {
			return "", fmt.Errorf(
				"%w: this Scaleway project has no Compute Instance quota yet — "+
					"verify the account's payment method and identity in the Scaleway console, "+
					"then contact Scaleway Support if quota is still 0 afterward",
				err,
			)
		}
		return "", err
	}
	defer closeBody(resp.Body)

	var result struct {
		Server struct {
			ID string `json:"id"`
		} `json:"server"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding server response: %w", err)
	}
	return result.Server.ID, nil
}

// setCloudInit must be called before powerOnServer: cloud-init only runs
// the user-data it finds at first boot.
func (c *scwAPIClient) setCloudInit(ctx context.Context, zone, serverID, script string) error {
	resp, err := c.doRequest(ctx, "PATCH",
		c.instanceBaseURL(zone)+"/servers/"+serverID+"/user_data/cloud-init", script, "text/plain")
	if err != nil {
		return err
	}
	defer closeBody(resp.Body)
	return nil
}

func (c *scwAPIClient) serverAction(ctx context.Context, zone, serverID, action string) error {
	resp, err := c.doRequest(ctx, "POST",
		c.instanceBaseURL(zone)+"/servers/"+serverID+"/action", map[string]string{"action": action}, "")
	if err != nil {
		return err
	}
	defer closeBody(resp.Body)
	return nil
}

func (c *scwAPIClient) powerOnServer(ctx context.Context, zone, serverID string) error {
	return c.serverAction(ctx, zone, serverID, "poweron")
}

func (c *scwAPIClient) terminateServer(ctx context.Context, zone, serverID string) error {
	return c.serverAction(ctx, zone, serverID, "terminate")
}

func (c *scwAPIClient) createLogsDataSource(
	ctx context.Context, projectID, name string,
) (string, string, error) {
	body := map[string]any{"project_id": projectID, "name": name, "type": "logs"}
	resp, reqErr := c.doRequest(ctx, "POST", c.cockpitBaseURL()+"/data-sources", body, "")
	if reqErr != nil {
		return "", "", reqErr
	}
	defer closeBody(resp.Body)

	var result struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if decErr := json.NewDecoder(resp.Body).Decode(&result); decErr != nil {
		return "", "", fmt.Errorf("decoding data source response: %w", decErr)
	}
	return result.ID, result.URL, nil
}

func (c *scwAPIClient) deleteLogsDataSource(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, "DELETE", c.cockpitBaseURL()+"/data-sources/"+id, nil, "")
	if err != nil {
		return err
	}
	defer closeBody(resp.Body)
	return nil
}

func (c *scwAPIClient) createLogsToken(ctx context.Context, projectID, name string) (string, string, error) {
	body := map[string]any{
		"project_id":   projectID,
		"name":         name,
		"token_scopes": []string{"write_only_logs", "read_only_logs"},
	}
	resp, reqErr := c.doRequest(ctx, "POST", c.cockpitBaseURL()+"/tokens", body, "")
	if reqErr != nil {
		return "", "", reqErr
	}
	defer closeBody(resp.Body)

	var result struct {
		ID        string `json:"id"`
		SecretKey string `json:"secret_key"`
	}
	if decErr := json.NewDecoder(resp.Body).Decode(&result); decErr != nil {
		return "", "", fmt.Errorf("decoding token response: %w", decErr)
	}
	return result.ID, result.SecretKey, nil
}

func (c *scwAPIClient) deleteLogsToken(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, "DELETE", c.cockpitBaseURL()+"/tokens/"+id, nil, "")
	if err != nil {
		return err
	}
	defer closeBody(resp.Body)
	return nil
}

type lokiQueryRangeResponse struct {
	Data struct {
		Result []struct {
			Values [][]string `json:"values"` // each value is [timestamp_ns_string, log_line]
		} `json:"result"`
	} `json:"data"`
}

// queryLoki queries a Cockpit logs data source's Loki-compatible API for
// log lines matching selector within [start, end).
func queryLoki(ctx context.Context, endpoint, token, selector string, start, end time.Time) ([]string, error) {
	params := url.Values{
		"query":     {selector},
		"start":     {strconv.FormatInt(start.UnixNano(), 10)},
		"end":       {strconv.FormatInt(end.UnixNano(), 10)},
		"direction": {"forward"},
		"limit":     {"1000"},
	}
	reqURL := endpoint + "/loki/api/v1/query_range?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth("api_key", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp.Body)
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d: %s", errScalewayAPI, resp.StatusCode, string(respBody))
	}

	var parsed lokiQueryRangeResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decoding Loki response: %w", err)
	}

	var lines []string
	for _, stream := range parsed.Data.Result {
		for _, v := range stream.Values {
			if len(v) >= 2 {
				lines = append(lines, v[1])
			}
		}
	}
	return lines, nil
}

func closeBody(body io.Closer) {
	_ = body.Close()
}
