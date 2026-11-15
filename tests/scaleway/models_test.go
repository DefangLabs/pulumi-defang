package scaleway

// Managed models on Scaleway, end to end through the Project component:
// direct inference has no gateway, so the provider consumes the virtual model
// services of a new CLI's payload, records each selection, and injects the
// resolved Generative APIs model ID into the bound containers.

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
	scwprov "github.com/DefangLabs/pulumi-defang/provider/defangscaleway/scaleway"
	"github.com/DefangLabs/pulumi-defang/provider/modelselect"
	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

type scalewayModelRun struct {
	selections *testutil.ModelSelections
	containers map[string]property.Map // container logical name → inputs
}

// env merges a container's plain and secret environment variables.
func (r *scalewayModelRun) env(t *testing.T, name string) map[string]string {
	t.Helper()
	inputs, ok := r.containers[name]
	require.True(t, ok, "no container %q", name)
	out := map[string]string{}
	for _, key := range []string{"environmentVariables", "secretEnvironmentVariables"} {
		v := inputs.Get(key)
		if !v.IsMap() {
			continue
		}
		for k, val := range v.AsMap().All {
			if val.IsString() { // secret or not: a secret is a flag on the value
				out[k] = val.AsString()
			}
		}
	}
	return out
}

func constructScalewayModels(t *testing.T, fixture, recipe string) (*scalewayModelRun, error) {
	t.Helper()
	// The payload's `OPENAI_API_KEY: null` is a Secret Manager reference, which
	// the Scaleway config provider reads over the real API; that path is
	// unchanged here, so the tests give the key a literal value.
	payload := strings.ReplaceAll(testutil.ModelFixture(t, fixture), "OPENAI_API_KEY: null", "OPENAI_API_KEY: test-key")

	run := &scalewayModelRun{containers: map[string]property.Map{}}
	run.selections = &testutil.ModelSelections{Selector: modelselect.ModelSelector{
		Cloud: scwprov.ModelCloud, Resolver: scwprov.GenerativeAPIsResolver{},
	}}
	var mu sync.Mutex
	mock := &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			if out, ok, err := run.selections.Handle(args); ok {
				return args.Name, out, err
			}
			if args.TypeToken == scalewayContainerType {
				mu.Lock()
				run.containers[args.Name] = args.Inputs
				mu.Unlock()
			}
			return args.Name, args.Inputs, nil
		},
	}
	config := testutil.StackConfig()
	if recipe != "" {
		config = testutil.StackConfig("defang-scaleway:models", recipe)
	}
	server := testutil.MakeScalewayTestServer(integration.WithMocks(mock))
	_, err := server.Construct(p.ConstructRequest{
		Urn:    testutil.ScalewayURN("Project"),
		Config: config,
		Inputs: testutil.ComposeInputs(t, payload),
	})
	return run, err
}

func TestScalewayModelRequestsFromCLIPayload(t *testing.T) {
	run, err := constructScalewayModels(t, "scaleway.yaml", `{
		"chat-default": [{"model": "mistral-small-3.2-24b-instruct-2506"}],
		"us.amazon.nova-2-lite-v1:0": [{"model": "llama-3.3-70b-instruct", "role": "chat"}]
	}`)
	require.NoError(t, err)

	// Only the two application containers exist: model services are virtual.
	assert.Len(t, run.containers, 2)

	app := run.env(t, "app")
	assert.Equal(t, "mistral-small-3.2-24b-instruct-2506", app["CHAT_MODEL"], "recipe target, not the alias")
	assert.Equal(t, "bge-multilingual-gemma2", app["EMBED_MODEL_NAME"], "compatibility default")
	assert.Equal(t, "https://proxy.example/v1/", app["CHAT_URL"], "explicit endpoint override kept")
	assert.Equal(t, "https://api.scaleway.ai/v1/", app["EMBED_BASE_URL"], "direct endpoint kept")

	worker := run.env(t, "worker")
	assert.Equal(t, "mistral-small-3.2-24b-instruct-2506", worker["WORKER_CHAT_MODEL"])
	assert.Equal(t, "llama-3.3-70b-instruct", worker["EXACT_MODEL"], "recipe key for a custom name")
	assert.Equal(t, "my-pinned-name", worker["LARGE_MODEL"], "explicit model override kept")

	assert.Equal(t, "chat-default", run.selections.Inputs["chat"].Request, "ai/ spelling normalized")
	assert.Equal(t, common.ModelSourceDefault, run.selections.Inputs["large"].Source)
	assert.Equal(t, "qwen3-235b-a22b-instruct-2507", run.selections.Selected["large"].Model,
		"chat-large defaults to the chat-default target")
}

func TestScalewayLegacyCLIPayloadUnchanged(t *testing.T) {
	// An older CLI already erased the model services; its concrete values are
	// deployed as they are, whatever the recipe now says.
	run, err := constructScalewayModels(t, "legacy-scaleway.yaml",
		`{"chat-default": [{"model": "mistral-small-3.2-24b-instruct-2506"}]}`)
	require.NoError(t, err)
	app := run.env(t, "app")
	assert.Equal(t, "qwen3-235b-a22b-instruct-2507", app["CHAT_MODEL"])
	assert.Equal(t, "bge-multilingual-gemma2", app["EMBED_MODEL_NAME"])
	assert.Empty(t, run.selections.Inputs)
}

func TestScalewayLLMContainerStillRejected(t *testing.T) {
	// A model service that brings its own container is not direct inference;
	// it keeps the existing unsupported error.
	payload := `
services:
  llm:
    image: litellm/litellm:v1.92.0
    ports: [{mode: ingress, target: 4000}]
    x-defang-llm: true
`
	server := testutil.MakeScalewayTestServer()
	_, err := server.Construct(p.ConstructRequest{
		Urn:    testutil.ScalewayURN("Project"),
		Inputs: testutil.ComposeInputs(t, payload),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LLM services are not implemented for Scaleway")
}
