package defangscaleway

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"testing"
	"time"
)

// TestInstanceBuildLive is an opt-in integration test against a real
// Scaleway account. It is skipped unless SCW_LIVE_TEST=1, since it costs
// real money and needs SCW_SECRET_KEY/AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY
// with access to a project that has Compute Instance quota (new Scaleway
// projects commonly start at zero quota for every instance type, which
// surfaces as a clear API error rather than a silent failure).
//
// It uploads a real Dockerfile exercising the exact patterns that broke
// under the old Kaniko-in-a-Serverless-Job backend (addgroup/adduser,
// COPY --chown, USER switch, RUN as non-root) to a throwaway Object
// Storage bucket, runs the actual Create path end to end, and tears
// everything (bucket, registry namespace) down afterward.
func TestInstanceBuildLive(t *testing.T) {
	if os.Getenv("SCW_LIVE_TEST") != "1" {
		t.Skip("set SCW_LIVE_TEST=1 to run this against a real Scaleway account")
	}
	region := os.Getenv("SCW_LIVE_TEST_REGION")
	if region == "" {
		region = "fr-par"
	}
	projectID := os.Getenv("SCW_DEFAULT_PROJECT_ID")
	if projectID == "" {
		t.Fatal("SCW_DEFAULT_PROJECT_ID must be set")
	}

	ctx := context.Background()
	bucket := fmt.Sprintf("defang-build-livetest-%d", time.Now().Unix())
	key := "uploads/digest"

	workdir := t.TempDir()
	dockerfile := `FROM alpine:3.19
RUN addgroup -g 2000 appgroup && adduser -D -u 2000 -G appgroup appuser
COPY --chown=appuser:appgroup testfile.txt /home/appuser/testfile.txt
USER appuser
RUN whoami && id && touch /home/appuser/created-by-appuser.txt && ls -la /home/appuser
`
	if err := os.WriteFile(workdir+"/Dockerfile", []byte(dockerfile), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workdir+"/testfile.txt", []byte("hello-from-live-test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tarPath := workdir + "/context.tar.gz"
	//nolint:gosec // G204: test-authored command, workdir/tarPath are from t.TempDir()
	tarCmd := exec.CommandContext(ctx, "tar", "-C", workdir, "-czf", tarPath, "Dockerfile", "testfile.txt")
	if out, err := tarCmd.CombinedOutput(); err != nil {
		t.Fatalf("tar: %v: %s", err, out)
	}

	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	s3Host := fmt.Sprintf("https://s3.%s.scw.cloud", region)
	sigv4 := fmt.Sprintf("aws:amz:%s:s3", region)
	userArg := accessKey + ":" + secretKey

	run := func(args ...string) string {
		//nolint:gosec // G204: test-authored args, this test's own S3 setup/teardown calls
		out, err := exec.CommandContext(ctx, "curl", args...).CombinedOutput()
		if err != nil {
			t.Fatalf("curl %v: %v: %s", args, err, out)
		}
		return string(out)
	}
	run("-sf", "-X", "PUT", s3Host+"/"+bucket, "--aws-sigv4", sigv4, "--user", userArg)
	t.Cleanup(func() {
		run("-s", "-X", "DELETE", s3Host+"/"+bucket+"/"+key, "--aws-sigv4", sigv4, "--user", userArg)
		run("-s", "-X", "DELETE", s3Host+"/"+bucket, "--aws-sigv4", sigv4, "--user", userArg)
	})
	run("-sf", "-X", "PUT", s3Host+"/"+bucket+"/"+key,
		"--aws-sigv4", sigv4, "--user", userArg, "--data-binary", "@"+tarPath)

	regClient := &scwAPIClient{secretKey: os.Getenv("SCW_SECRET_KEY"), region: region}
	nsName := fmt.Sprintf("build-livetest-%d", time.Now().Unix())
	destination := fmt.Sprintf("rg.%s.scw.cloud/%s/app:latest", region, nsName)
	// The registry namespace must exist before `docker login`/push will work.
	if err := createRegistryNamespaceForTest(ctx, regClient, projectID, nsName); err != nil {
		t.Fatalf("creating registry namespace: %v", err)
	}
	t.Cleanup(func() {
		_ = deleteRegistryNamespaceForTest(ctx, regClient, nsName, projectID)
	})

	// runInstanceBuild no longer creates its own Cockpit data source/token
	// (see project.go: these are persistent, project-level resources now) —
	// this test stands in for project.go and creates its own throwaway pair.
	logsEndpoint, logsToken := setupCockpitForTest(ctx, t, regClient, projectID, nsName+"-logs")

	inputs := BuildInputs{
		Region:       region,
		ProjectId:    projectID,
		Source:       fmt.Sprintf("s3://%s/%s", bucket, key),
		Destination:  destination,
		LogsEndpoint: logsEndpoint,
		LogsToken:    logsToken,
	}

	assertNoLeftovers := leftoverCheckForTest(ctx, t, regClient, defaultZone(region), projectID)

	result, err := runInstanceBuild(ctx, inputs)
	if err != nil {
		t.Fatalf("runInstanceBuild: %v", err)
	}
	t.Logf("build succeeded: instance=%s logSelector=%s", result.instanceID, result.logSelector)
	// Teardown ran in runInstanceBuild's defer, so the build instance's
	// boot volume and IPs must already be gone.
	assertNoLeftovers()
}

// leftoverCheckForTest records the project's SBS volumes and Instance IPs in
// zone, and returns a check that fails the test for any that appeared since.
func leftoverCheckForTest(ctx context.Context, t *testing.T, c *scwAPIClient, zone, projectID string) func() {
	t.Helper()
	before := listLeftoversForTest(ctx, t, c, zone, projectID)
	return func() {
		t.Helper()
		for _, id := range listLeftoversForTest(ctx, t, c, zone, projectID) {
			if !slices.Contains(before, id) {
				t.Errorf("build left %s behind", id)
			}
		}
	}
}

// listLeftoversForTest lists the project's SBS volumes and Instance IPs in
// zone, as "volume <id>" / "ip <id>".
func listLeftoversForTest(ctx context.Context, t *testing.T, c *scwAPIClient, zone, projectID string) []string {
	t.Helper()
	out := make([]string, 0, 16)
	for _, l := range []struct{ kind, url, field string }{
		{"volume", c.blockBaseURL(zone) + "/volumes?page_size=100&project_id=" + projectID, "volumes"},
		{"ip", c.instanceBaseURL(zone) + "/ips?per_page=100&project=" + projectID, "ips"},
	} {
		// The list sits beside a numeric total_count, so decode it separately.
		page, err := getJSONForTest[map[string]json.RawMessage](ctx, c, l.url)
		if err != nil {
			t.Fatalf("listing %ss: %v", l.kind, err)
		}
		var items []struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(page[l.field], &items); err != nil {
			t.Fatalf("decoding %ss: %v", l.kind, err)
		}
		for _, item := range items {
			out = append(out, l.kind+" "+item.ID)
		}
	}
	return out
}

// setupCockpitForTest creates a throwaway Cockpit data source + token
// standing in for the persistent, project-level ones project.go now creates,
// registers their teardown, and returns (logsEndpoint, logsToken).
func setupCockpitForTest(
	ctx context.Context, t *testing.T, c *scwAPIClient, projectID, name string,
) (string, string) {
	t.Helper()
	dataSourceID, logsEndpoint, err := createCockpitDataSourceForTest(ctx, c, projectID, name)
	if err != nil {
		t.Fatalf("creating Cockpit data source: %v", err)
	}
	t.Cleanup(func() { _ = deleteCockpitDataSourceForTest(ctx, c, dataSourceID) })
	tokenID, logsToken, err := createCockpitTokenForTest(ctx, c, projectID, name)
	if err != nil {
		t.Fatalf("creating Cockpit token: %v", err)
	}
	t.Cleanup(func() { _ = deleteCockpitTokenForTest(ctx, c, tokenID) })
	return logsEndpoint, logsToken
}

func (c *scwAPIClient) cockpitBaseURLForTest() string {
	return "https://api.scaleway.com/cockpit/v1/regions/" + c.region
}

func createCockpitDataSourceForTest(
	ctx context.Context, c *scwAPIClient, projectID, name string,
) (string, string, error) {
	body := map[string]any{"project_id": projectID, "name": name, "type": "logs"}
	resp, err := c.doRequest(ctx, "POST", c.cockpitBaseURLForTest()+"/data-sources", body, "")
	if err != nil {
		return "", "", err
	}
	defer closeBody(resp.Body)
	var result struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}
	return result.ID, result.URL, nil
}

func deleteCockpitDataSourceForTest(ctx context.Context, c *scwAPIClient, id string) error {
	resp, err := c.doRequest(ctx, "DELETE", c.cockpitBaseURLForTest()+"/data-sources/"+id, nil, "")
	if err != nil {
		return err
	}
	defer closeBody(resp.Body)
	return nil
}

func createCockpitTokenForTest(ctx context.Context, c *scwAPIClient, projectID, name string) (string, string, error) {
	body := map[string]any{
		"project_id":   projectID,
		"name":         name,
		"token_scopes": []string{"write_only_logs", "read_only_logs"},
	}
	resp, err := c.doRequest(ctx, "POST", c.cockpitBaseURLForTest()+"/tokens", body, "")
	if err != nil {
		return "", "", err
	}
	defer closeBody(resp.Body)
	var result struct {
		ID        string `json:"id"`
		SecretKey string `json:"secret_key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}
	return result.ID, result.SecretKey, nil
}

func deleteCockpitTokenForTest(ctx context.Context, c *scwAPIClient, id string) error {
	resp, err := c.doRequest(ctx, "DELETE", c.cockpitBaseURLForTest()+"/tokens/"+id, nil, "")
	if err != nil {
		return err
	}
	defer closeBody(resp.Body)
	return nil
}

func createRegistryNamespaceForTest(ctx context.Context, c *scwAPIClient, projectID, name string) error {
	url := "https://api.scaleway.com/registry/v1/regions/" + c.region + "/namespaces"
	body := map[string]any{"project_id": projectID, "name": name}
	resp, err := c.doRequest(ctx, "POST", url, body, "")
	if err != nil {
		return err
	}
	defer closeBody(resp.Body)
	return nil
}

func deleteRegistryNamespaceForTest(ctx context.Context, c *scwAPIClient, name, projectID string) error {
	// List to find the ID (the create response is ignored above for brevity; look it up by name).
	url := "https://api.scaleway.com/registry/v1/regions/" + c.region +
		"/namespaces?project_id=" + projectID + "&name=" + name
	resp, err := c.doRequest(ctx, "GET", url, nil, "")
	if err != nil {
		return err
	}
	defer closeBody(resp.Body)
	var result struct {
		Namespaces []struct {
			ID string `json:"id"`
		} `json:"namespaces"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	for _, ns := range result.Namespaces {
		resp, err := c.doRequest(ctx, "DELETE",
			"https://api.scaleway.com/registry/v1/regions/"+c.region+"/namespaces/"+ns.ID, nil, "")
		if err != nil {
			return err
		}
		defer closeBody(resp.Body)
	}
	return nil
}

func getJSONForTest[T any](ctx context.Context, c *scwAPIClient, url string) (T, error) {
	var v T
	resp, err := c.doRequest(ctx, "GET", url, nil, "")
	if err != nil {
		return v, err
	}
	defer closeBody(resp.Body)
	err = json.NewDecoder(resp.Body).Decode(&v)
	return v, err
}
