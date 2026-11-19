package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/DefangLabs/defang/src/pkg/clouds/scaleway"
	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"github.com/DefangLabs/pulumi-defang/cd/program"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

const testScalewayProjectUpdateKey = "projects/app/dev/project.pb"

func TestScalewayTTLCommandDispatch(t *testing.T) {
	t.Setenv("PROJECT", "app")
	t.Setenv("STACK", "dev")
	t.Setenv("DEFANG_TTL", "")
	err := cdMain(t.Context(), cdExecutable, "scaleway-ttl", "up", "invalid-payload")
	require.ErrorContains(t, err, "requires DEFANG_TTL")

	t.Setenv("DEFANG_TTL", "2h")
	for _, command := range []string{"up", "preview"} {
		err = cdMain(t.Context(), cdExecutable, "scaleway-ttl", command, "invalid-payload")
		require.ErrorContains(t, err, "failed to fetch payload")
	}
}

func TestScalewayScheduledDownBeforeDeadline(t *testing.T) {
	t.Setenv("PROJECT", "app")
	t.Setenv("STACK", "dev")
	t.Setenv("SCW_SECRET_KEY", "")
	deadline := time.Now().AddDate(1, 0, 0).UTC().Format(time.RFC3339)
	args := []string{cdExecutable, program.ScalewayTriggerDownCommand, "defang-cd-dev",
		deadline, "deployment", testScalewayProjectUpdateKey}
	require.NoError(t, cdMain(t.Context(), args...))
}

type scalewayTriggerTransport func(*http.Request) (*http.Response, error)

func (f scalewayTriggerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestScalewayScheduledDownStartsCurrentJob(t *testing.T) {
	const scheduledETag = "scheduled"
	for _, tc := range []struct {
		name     string
		etag     string
		readErr  error
		wantDown bool
	}{
		{name: "current deployment", etag: scheduledETag, wantDown: true},
		{name: "redeployed", etag: "newer"},
		{name: "removed", readErr: &types.NoSuchKey{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scw := scaleway.NewClient("", "token", "project-id", "fr-par")
			started := false
			scw.HTTPClient = &http.Client{Transport: scalewayTriggerTransport(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "token", req.Header.Get("X-Auth-Token"))
				var body string
				switch req.Method {
				case http.MethodGet:
					assert.Equal(t, "/serverless-jobs/v1alpha2/regions/fr-par/job-definitions", req.URL.Path)
					assert.Equal(t, "project-id", req.URL.Query().Get("project_id"))
					body = `{"total_count":2,"job_definitions":[
						{"id":"other","name":"defang-cd-other","project_id":"project-id"},
						{"id":"replacement","name":"defang-cd-dev","project_id":"project-id",
						 "environment_variables":{"SCW_ACCESS_KEY":"current-access-key"}}
					]}`
				case http.MethodPost:
					started = true
					assert.Equal(t, "/serverless-jobs/v1alpha2/regions/fr-par/job-definitions/replacement/start", req.URL.Path)
					var start struct {
						Command []string          `json:"startup_command"`
						Args    []string          `json:"args"`
						Env     map[string]string `json:"environment_variables"`
					}
					require.NoError(t, json.NewDecoder(req.Body).Decode(&start))
					assert.Equal(t, []string{cdExecutable}, start.Command)
					assert.Equal(t, []string{
						string(cdCommandScalewayScheduledDown), scheduledETag, testScalewayProjectUpdateKey,
					}, start.Args)
					assert.Equal(t, map[string]string{
						"PROJECT": "app", "STACK": "dev",
					}, start.Env)
					body = `{"job_runs":[{"id":"down-run"}]}`
				default:
					t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
			})}
			down := scalewayScheduledDown{
				project: "app", stack: "dev", jobName: "defang-cd-dev", etag: scheduledETag, key: testScalewayProjectUpdateKey,
			}
			require.NoError(t, down.start(t.Context(), scw, func(accessKey, key string) ([]byte, error) {
				assert.Equal(t, "current-access-key", accessKey)
				assert.Equal(t, testScalewayProjectUpdateKey, key)
				if tc.readErr != nil {
					return nil, tc.readErr
				}
				return proto.Marshal(&defangv1.ProjectUpdate{Etag: tc.etag})
			}))
			assert.Equal(t, tc.wantDown, started)
		})
	}
}
