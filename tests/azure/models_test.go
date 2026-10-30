package azure

// Managed models on Azure, end to end through the Project component: the
// Compose payload a CLI uploads, the recipe in stack config, the
// ModelSelection resolved against the AI Foundry account's model list, the
// Foundry deployments created from it, and the literal model names the
// Container Apps receive.

import (
	"context"
	"sync"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	azureprov "github.com/DefangLabs/pulumi-defang/provider/defangazure/azure"
	"github.com/DefangLabs/pulumi-defang/provider/modelselect"
	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

const foundryAccountID = "/subscriptions/sub-1/resourceGroups/rg-1/providers/" +
	"Microsoft.CognitiveServices/accounts/acct-1"

const (
	generallyAvailable = "GenerallyAvailable"
	foundryAccountType = "azure-native:cognitiveservices:Account"
)

var foundryCatalog = []azureprov.AvailableModel{
	{Name: "gpt-4o", Format: "OpenAI", Version: "2024-11-20", LifecycleStatus: "Deprecating", SKUs: []string{"Standard"}},
	{Name: "gpt-4.1", Format: "OpenAI", Version: "2025-04-14", LifecycleStatus: generallyAvailable,
		SKUs: []string{"Standard"}, Capabilities: map[string]string{"chatCompletion": "true"}},
	{Name: "gpt-5.1", Format: "OpenAI", Version: "2025-11-13", LifecycleStatus: "Preview",
		SKUs: []string{"GlobalStandard"}, Capabilities: map[string]string{"chatCompletion": "true"}},
	{Name: "Phi-4", Format: "Microsoft", Version: "9", LifecycleStatus: generallyAvailable,
		SKUs: []string{"GlobalStandard"}},
	{Name: "Phi-4", Format: "Microsoft", Version: "10", LifecycleStatus: generallyAvailable,
		SKUs: []string{"GlobalStandard"}},
	{Name: "text-embedding-3-small", Format: "OpenAI", Version: "1", LifecycleStatus: generallyAvailable,
		SKUs: []string{"Standard"}, Capabilities: map[string]string{"embeddings": "true"}},
}

// foundryList serves foundryCatalog for the test account.
func foundryList(t *testing.T) func(context.Context, string, string, string) ([]azureprov.AvailableModel, error) {
	t.Helper()
	return func(_ context.Context, sub, rg, acct string) ([]azureprov.AvailableModel, error) {
		assert.Equal(t, []string{"sub-1", "rg-1", "acct-1"}, []string{sub, rg, acct})
		return foundryCatalog, nil
	}
}

// deployment is the model of a Foundry deployment.
type deployment struct{ model, version, format, sku string }

type azureModelRun struct {
	selections  *testutil.ModelSelections
	deployments map[string]property.Map // deployment name → inputs
	apps        map[string]property.Map // container app → inputs
}

func (r *azureModelRun) deployed(t *testing.T, name string) deployment {
	t.Helper()
	d, ok := r.deployments[name]
	require.True(t, ok, "no Foundry deployment %q", name)
	m := d.Get("properties").AsMap().Get("model").AsMap()
	return deployment{
		model:   m.Get("name").AsString(),
		version: m.Get("version").AsString(),
		format:  m.Get("format").AsString(),
		sku:     d.Get("sku").AsMap().Get("name").AsString(),
	}
}

func (r *azureModelRun) env(t *testing.T, app string) map[string]string {
	t.Helper()
	inputs, ok := r.apps[app]
	require.True(t, ok, "no container app %q", app)
	out := map[string]string{}
	containers := inputs.Get("template").AsMap().Get("containers").AsArray()
	for _, e := range containers.Get(0).AsMap().Get("env").AsArray().All {
		m := e.AsMap()
		if v := m.Get("value"); v.IsString() {
			out[m.Get("name").AsString()] = v.AsString()
		}
	}
	return out
}

func constructAzureModels(t *testing.T, payload, recipe string) *azureModelRun {
	t.Helper()

	run := &azureModelRun{deployments: map[string]property.Map{}, apps: map[string]property.Map{}}
	run.selections = &testutil.ModelSelections{Selector: modelselect.ModelSelector{
		Cloud: azureprov.ModelCloud,
		Resolver:      &azureprov.FoundryResolver{List: foundryList(t)},
		ScopeRequired: true,
	}}
	var mu sync.Mutex
	mock := &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			if out, ok, err := run.selections.Handle(args); ok {
				return args.Name, out, err
			}
			mu.Lock()
			defer mu.Unlock()
			if args.TypeToken == foundryAccountType {
				return foundryAccountID, args.Inputs, nil
			}
			if args.TypeToken == "azure-native:cognitiveservices:Deployment" {
				run.deployments[args.Inputs.Get("deploymentName").AsString()] = args.Inputs
			}
			if args.TypeToken == "azure-native:app:ContainerApp" {
				run.apps[args.Name] = args.Inputs
			}
			return args.Name, args.Inputs, nil
		},
	}
	config := testutil.StackConfig()
	if recipe != "" {
		config = testutil.StackConfig("defang-azure:models", recipe)
	}
	server := testutil.MakeAzureTestServer(integration.WithMocks(mock))
	_, err := server.Construct(p.ConstructRequest{
		Urn:    testutil.AzureURN("Project"),
		Config: config,
		Inputs: testutil.ComposeInputs(t, payload),
	})
	require.NoError(t, err)
	return run
}

func TestAzureModelRequestsFromCLIPayload(t *testing.T) {
	// The shared payload's "exact" service asks for a Bedrock ID; on Azure a
	// recipe key of that name maps it to a model the account offers.
	run := constructAzureModels(t, testutil.ModelFixture(t, "azure.yaml"), `{
		"chat-large": [{"model": "Phi-4"}],
		"embedding-default": [{"model": "text-embedding-3-small", "format": "OpenAI", "version": "1"}],
		"us.amazon.nova-2-lite-v1:0": [{"model": "gpt-4.1", "role": "chat"}]
	}`)

	// chat-default: the compatibility list, skipping the deprecating gpt-4o;
	// a preview gpt-5.1 is still eligible for the built-in default as before.
	assert.Equal(t, deployment{"gpt-5.1", "2025-11-13", "OpenAI", "GlobalStandard"}, run.deployed(t, "chat-default"))

	// chat-large has its own recipe entry; revision 10 is newer than 9.
	assert.Equal(t, deployment{"Phi-4", "10", "Microsoft", "GlobalStandard"}, run.deployed(t, "chat-large"))

	// A pinned version is deployed exactly.
	assert.Equal(t, deployment{"text-embedding-3-small", "1", "OpenAI", "Standard"}, run.deployed(t, "embedding-default"))
	assert.Equal(t, common.ModelRoleEmbedding, run.selections.Selected["embed"].SelectedRole)

	// The literal model name applications send is the deployment name.
	app := run.env(t, "app")
	assert.Equal(t, "chat-default", app["CHAT_MODEL"])
	assert.Equal(t, "embedding-default", app["EMBED_MODEL_NAME"])
	assert.Equal(t, "https://proxy.example/v1/", app["CHAT_URL"], "explicit endpoint override kept")
	worker := run.env(t, "worker")
	assert.Equal(t, "chat-default", worker["WORKER_CHAT_MODEL"])
	assert.Equal(t, "my-pinned-name", worker["LARGE_MODEL"], "explicit model override kept")
	assert.Equal(t, "us.amazon.nova-2-lite-v1:0", worker["EXACT_MODEL"])
	assert.Equal(t, "gpt-4.1", run.deployed(t, "us.amazon.nova-2-lite-v1:0").model)
}

func TestAzureUnavailableModelNamesTheFix(t *testing.T) {
	server := testutil.MakeAzureTestServer(integration.WithMocks(&integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			selections := &testutil.ModelSelections{Selector: modelselect.ModelSelector{
				Cloud: azureprov.ModelCloud, ScopeRequired: true,
				Resolver: &azureprov.FoundryResolver{List: foundryList(t)},
			}}
			if out, ok, err := selections.Handle(args); ok {
				return args.Name, out, err
			}
			if args.TypeToken == foundryAccountType {
				return foundryAccountID, args.Inputs, nil
			}
			return args.Name, args.Inputs, nil
		},
	}))
	_, err := server.Construct(p.ConstructRequest{
		Urn:    testutil.AzureURN("Project"),
		Inputs: testutil.ComposeInputs(t, testutil.ModelFixture(t, "azure.yaml")),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "us.amazon.nova-2-lite-v1:0 is not offered by this account")
	assert.Contains(t, err.Error(), "defang-azure:models.us.amazon.nova-2-lite-v1:0")
}

func TestAzureNativeModelNameIsHonored(t *testing.T) {
	// A Compose model name is deployed as that model, never replaced by a
	// role default chosen from a substring of the name.
	payload := `
services:
  app:
    image: app
    ports: [{mode: ingress, target: 8080}]
    environment: {PHI_MODEL: Phi-4, PHI_URL: "http://phi:4000/v1/"}
    models: {phi: {model_var: PHI_MODEL, endpoint_var: PHI_URL}}
  phi:
    image: litellm/litellm:v1.92.0
    ports: [{mode: host, target: 4000, protocol: tcp}]
    x-defang-llm: true
models:
  phi:
    model: Phi-4
`
	run := constructAzureModels(t, payload, "")
	assert.Equal(t, deployment{"Phi-4", "10", "Microsoft", "GlobalStandard"}, run.deployed(t, "Phi-4"))
	assert.Equal(t, "Phi-4", run.env(t, "app")["PHI_MODEL"])
	assert.Equal(t, common.ModelSourceNative, run.selections.Inputs["phi"].Source)
}

func TestAzureLegacyCLIPayload(t *testing.T) {
	// Older CLIs sent the bare alias; the alias is recovered and resolved
	// through the recipe, and the deployment name is unchanged.
	run := constructAzureModels(t, testutil.ModelFixture(t, "legacy-azure.yaml"),
		`{"chat-default": [{"model": "gpt-4.1", "format": "OpenAI"}]}`)
	assert.Equal(t, "gpt-4.1", run.deployed(t, "chat-default").model)
	assert.Equal(t, "text-embedding-3-small", run.deployed(t, "embedding-default").model,
		"embedding is never guessed from a substring")
	assert.Equal(t, "chat-default", run.env(t, "app")["CHAT_MODEL"])
}
