package gcp

// Managed models on GCP, end to end through the Project component: the Compose
// payload a CLI uploads, the recipe in stack config, the ModelSelection the
// provider records, and the Compute Engine cloud-init that runs the gateway and
// its consumers — the gateway command, its Vertex AI location and the literal
// values applications receive.

import (
	"strings"
	"sync"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	gcpprov "github.com/DefangLabs/pulumi-defang/provider/defanggcp/gcp"
	"github.com/DefangLabs/pulumi-defang/provider/modelselect"
	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

type gcpModelRun struct {
	selections *testutil.ModelSelections
	userData   map[string]string // container name → cloud-init user-data
}

// unit returns the systemd unit text that runs the named container.
func (r *gcpModelRun) execStart(t *testing.T, container string) string {
	t.Helper()
	data, ok := r.userData[container]
	require.True(t, ok, "no instance template runs %s", container)
	for _, line := range strings.Split(data, "\n") {
		if strings.Contains(line, "ExecStart=/usr/bin/docker run") && strings.Contains(line, "--name="+container+" ") {
			return line
		}
	}
	t.Fatalf("no docker run line for %s", container)
	return ""
}

func constructGcpModels(t *testing.T, fixture, recipe string) *gcpModelRun {
	t.Helper()
	payload := testutil.ModelFixture(t, fixture)
	run := &gcpModelRun{userData: map[string]string{}}
	run.selections = &testutil.ModelSelections{Selector: modelselect.ModelSelector{
		Cloud: gcpprov.ModelCloud, Resolver: gcpprov.VertexResolver{},
	}}
	var mu sync.Mutex
	mock := &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			if out, ok, err := run.selections.Handle(args); ok {
				return args.Name, out, err
			}
			if args.TypeToken == gcpInstanceTemplateType {
				data := args.Inputs.Get("metadata").AsMap().Get("user-data").AsString()
				for _, field := range strings.Fields(data) {
					if name, ok := strings.CutPrefix(field, "--name="); ok {
						mu.Lock()
						run.userData[name] = data
						mu.Unlock()
						break
					}
				}
			}
			return args.Name, args.Inputs, nil
		},
	}
	kv := []string{"gcp:project", "my-project", "gcp:region", "us-central1"}
	if recipe != "" {
		kv = append(kv, "defang-gcp:models", recipe)
	}
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))
	_, err := server.Construct(p.ConstructRequest{
		Urn:    testutil.GcpURN("Project"),
		Config: testutil.StackConfig(kv...),
		Inputs: testutil.ComposeInputs(t, payload),
	})
	require.NoError(t, err)
	return run
}

func TestGcpModelRequestsFromCLIPayload(t *testing.T) {
	run := constructGcpModels(t, "gcp.yaml",
		`{"chat-default": [{"model": "gemini-3.5-flash-lite", "location": "europe-west4"}]}`)

	chat := run.execStart(t, "chat")
	assert.Contains(t, chat, "--drop_params --model vertex_ai/gemini-3.5-flash-lite --alias chat-default")
	assert.Contains(t, chat, "VERTEXAI_LOCATION=europe-west4", "the recipe location routes the gateway")
	assert.Contains(t, chat, "VERTEXAI_PROJECT=my-project")

	// Compatibility defaults keep today's choices, including global routing.
	large := run.execStart(t, "large")
	assert.Contains(t, large, "--model vertex_ai/gemini-3.1-pro-preview --alias chat-large")
	assert.Contains(t, large, "VERTEXAI_LOCATION=global")
	embed := run.execStart(t, "embed")
	assert.Contains(t, embed, "--model vertex_ai/gemini-embedding-001 --alias embedding-default")
	assert.Contains(t, embed, "VERTEXAI_LOCATION=us-central1", "embedding default stays in the stack region")
	assert.Equal(t, common.ModelSourceDefault, run.selections.Inputs["embed"].Source)

	app := run.execStart(t, "app")
	assert.Contains(t, app, "CHAT_MODEL=chat-default")
	assert.Contains(t, app, "CHAT_URL=https://proxy.example/v1/", "explicit endpoint override kept")
	assert.Contains(t, app, "EMBED_MODEL_NAME=embedding-default")
	worker := run.execStart(t, "worker")
	assert.Contains(t, worker, "WORKER_CHAT_MODEL=chat-default")
	assert.Contains(t, worker, "EXACT_MODEL=us.amazon.nova-2-lite-v1:0")
	assert.Contains(t, worker, "LARGE_MODEL=my-pinned-name", "explicit model override kept")
}

func TestGcpLegacyCLIPayloadUnchanged(t *testing.T) {
	run := constructGcpModels(t, "legacy-gcp.yaml", `{"chat-default": [{"model": "other"}]}`)

	chat := run.execStart(t, "chat")
	assert.Contains(t, chat, "--drop_params --model vertex_ai/gemini-3.5-flash --alias chat-default")
	assert.Contains(t, chat, "VERTEXAI_LOCATION=global")
	assert.Contains(t, run.execStart(t, "app"), "CHAT_MODEL=vertex_ai/gemini-3.5-flash")

	assert.Equal(t, common.ModelSourceLegacyPayload, run.selections.Inputs["chat"].Source)
	assert.Equal(t, "gemini-3.5-flash", run.selections.Selected["chat"].Model)
	assert.Equal(t, "global", run.selections.Selected["chat"].Location)
	assert.Equal(t, "us-central1", run.selections.Selected["embed"].Location)
}

func TestGcpCustomKeyNeedsRole(t *testing.T) {
	payload := `
services:
  llm:
    image: litellm/litellm:v1.92.0
    ports: [{mode: host, target: 4000, protocol: tcp}]
    x-defang-llm: true
models:
  llm:
    model: fast
`
	server := testutil.MakeGcpTestServer()
	_, err := server.Construct(p.ConstructRequest{
		Urn:    testutil.GcpURN("Project"),
		Config: testutil.StackConfig("defang-gcp:models", `{"fast": [{"model": "gemini-3.5-flash"}]}`),
		Inputs: testutil.ComposeInputs(t, payload),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "defang-gcp:models.fast needs `role: chat` or `role: embedding`")
}
