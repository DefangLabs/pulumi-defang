package aws

// Managed models on AWS, end to end through the Project component: the Compose
// payload a CLI uploads, the recipe in stack config, the ModelSelection the
// provider records (run through the real selection lifecycle), and the ECS
// task definitions that come out — the gateway command and the literal values
// applications receive.

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	awsprov "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/DefangLabs/pulumi-defang/provider/modelselect"
	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

// Resource types the AWS tests capture.
const (
	ecsServiceType        = "aws:ecs/service:Service"
	ecsTaskDefinitionType = "aws:ecs/taskDefinition:TaskDefinition"
)

type fakeBedrockCatalog struct {
	calls int
}

func (f *fakeBedrockCatalog) ListFoundationModels(
	context.Context, *bedrock.ListFoundationModelsInput, ...func(*bedrock.Options),
) (*bedrock.ListFoundationModelsOutput, error) {
	f.calls++
	text := []bedrocktypes.ModelModality{bedrocktypes.ModelModalityText}
	active := &bedrocktypes.FoundationModelLifecycle{Status: bedrocktypes.FoundationModelLifecycleStatusActive}
	return &bedrock.ListFoundationModelsOutput{ModelSummaries: []bedrocktypes.FoundationModelSummary{
		{ModelId: aws.String("zai.glm-5"), OutputModalities: text, ModelLifecycle: active},
		{ModelId: aws.String("amazon.nova-pro-v1:0"), OutputModalities: text, ModelLifecycle: active},
		{ModelId: aws.String("amazon.nova-pro-v1:1"), OutputModalities: text, ModelLifecycle: active},
	}}, nil
}

func (f *fakeBedrockCatalog) ListInferenceProfiles(
	context.Context, *bedrock.ListInferenceProfilesInput, ...func(*bedrock.Options),
) (*bedrock.ListInferenceProfilesOutput, error) {
	return &bedrock.ListInferenceProfilesOutput{}, nil
}

type awsModelRun struct {
	selections *testutil.ModelSelections
	containers map[string]awsprov.ContainerDefinition // service → main container
	catalog    *fakeBedrockCatalog
}

func (r *awsModelRun) env(service string) map[string]string {
	out := map[string]string{}
	for _, e := range r.containers[service].Environment {
		out[e.Name] = e.Value
	}
	return out
}

func constructAwsModels(t *testing.T, fixture string, recipe string) *awsModelRun {
	t.Helper()
	payload := testutil.ModelFixture(t, fixture)
	run := &awsModelRun{containers: map[string]awsprov.ContainerDefinition{}, catalog: &fakeBedrockCatalog{}}
	run.selections = &testutil.ModelSelections{Selector: modelselect.ModelSelector{
		Cloud: awsprov.ModelCloud,
		Resolver: &awsprov.BedrockResolver{NewClient: func(context.Context, string) (awsprov.BedrockCatalogAPI, error) {
			return run.catalog, nil
		}},
	}}
	var mu sync.Mutex
	mock := &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			if out, ok, err := run.selections.Handle(args); ok {
				return args.Name, out, err
			}
			if args.TypeToken == ecsTaskDefinitionType {
				var defs []awsprov.ContainerDefinition
				require.NoError(t, json.Unmarshal([]byte(args.Inputs.Get("containerDefinitions").AsString()), &defs))
				mu.Lock()
				run.containers[args.Name] = defs[0]
				mu.Unlock()
			}
			return args.Name, args.Inputs, nil
		},
	}
	config := testutil.StackConfig()
	if recipe != "" {
		config = testutil.StackConfig("defang-aws:models", recipe)
	}
	server := testutil.MakeAwsTestServer(integration.WithMocks(mock))
	_, err := server.Construct(p.ConstructRequest{
		Urn:    testutil.AwsURN("Project"),
		Config: config,
		Inputs: testutil.ComposeInputs(t, payload),
	})
	require.NoError(t, err)
	return run
}

func TestAwsModelRequestsFromCLIPayload(t *testing.T) {
	// A custom recipe: chat-default is overridden with a versionless family,
	// the other aliases are absent and use the compatibility defaults.
	run := constructAwsModels(t, "aws.yaml", `{"chat-default": [{"model": "amazon.nova-pro-v1"}]}`)

	// Selections: the request as written (ai/ normalized), only its recipe entry.
	chat := run.selections.Inputs["chat"]
	assert.Equal(t, "chat-default", chat.Request)
	assert.Equal(t, common.ModelSourceRecipe, chat.Source)
	assert.Equal(t, "amazon.nova-pro-v1", chat.Candidates[0].Model)
	assert.Equal(t, common.ModelSourceDefault, run.selections.Inputs["embed"].Source)
	assert.Equal(t, common.ModelSourceNative, run.selections.Inputs["exact"].Source)
	assert.Equal(t, "amazon.nova-pro-v1:1", run.selections.Selected["chat"].Model, "highest revision of the family")

	// Gateway commands serve the resolved target under the application-facing name.
	assert.Equal(t, []string{"--drop_params", "--model", "bedrock/amazon.nova-pro-v1:1", "--alias", "chat-default"},
		run.containers["chat"].Command)
	assert.Equal(t, []string{"--drop_params", "--model", "bedrock/amazon.titan-embed-text-v2:0", "--alias",
		"embedding-default"}, run.containers["embed"].Command)
	assert.Equal(t, []string{"--drop_params", "--model", "bedrock/zai.glm-5", "--alias", "chat-large"},
		run.containers["large"].Command)
	assert.Equal(t, []string{"--drop_params", "--model", "bedrock/us.amazon.nova-2-lite-v1:0", "--alias",
		"us.amazon.nova-2-lite-v1:0"}, run.containers["exact"].Command)
	assert.Equal(t, "us-west-2", run.env("chat")["AWS_REGION"], "the CLI's region hint is kept")
	assert.Equal(t, "networkisalreadyprivate", run.env("chat")["LITELLM_MASTER_KEY"])

	// The literal values applications receive.
	app := run.env("app")
	assert.Equal(t, "chat-default", app["CHAT_MODEL"])
	assert.Equal(t, "https://proxy.example/v1/", app["CHAT_URL"], "explicit endpoint override kept")
	assert.Equal(t, "embedding-default", app["EMBED_MODEL_NAME"], "custom model_var")
	assert.Contains(t, app["EMBED_BASE_URL"], ":4000/v1/")
	worker := run.env("worker")
	assert.Equal(t, "chat-default", worker["WORKER_CHAT_MODEL"])
	assert.Equal(t, "us.amazon.nova-2-lite-v1:0", worker["EXACT_MODEL"])
	assert.Equal(t, "my-pinned-name", worker["LARGE_MODEL"], "explicit model override kept")
	assert.NotContains(t, worker, "CHAT_MODEL")

	// Only the colon-less chat-large default and the family needed the catalog, once.
	assert.Equal(t, 1, run.catalog.calls)
}

func TestAwsModelRequestsOlderRecipe(t *testing.T) {
	// No defang-aws:models at all: today's defaults, no catalog for exact IDs.
	run := constructAwsModels(t, "aws.yaml", "")
	assert.Equal(t, []string{"--drop_params", "--model", "bedrock/us.amazon.nova-2-lite-v1:0", "--alias", "chat-default"},
		run.containers["chat"].Command)
	assert.Equal(t, common.ModelSourceDefault, run.selections.Inputs["chat"].Source)
}

func TestAwsLegacyCLIPayloadUnchanged(t *testing.T) {
	// A recipe change must not alter what an older CLI deployed.
	run := constructAwsModels(t, "legacy-aws.yaml", `{"chat-default": [{"model": "amazon.nova-pro-v1:0"}]}`)

	assert.Equal(t, []string{"--drop_params", "--model", "bedrock/us.amazon.nova-2-lite-v1:0", "--alias", "chat-default"},
		run.containers["chat"].Command, "exact routing preserved")
	assert.Equal(t, "bedrock/us.amazon.nova-2-lite-v1:0", run.env("app")["CHAT_MODEL"])
	assert.Equal(t, "bedrock/amazon.titan-embed-text-v2:0", run.env("app")["EMBED_MODEL_NAME"])

	// The deployed targets are recorded, so a later deployment knows them.
	assert.Equal(t, common.ModelSourceLegacyPayload, run.selections.Inputs["chat"].Source)
	assert.Equal(t, "us.amazon.nova-2-lite-v1:0", run.selections.Selected["chat"].Model)
	assert.Equal(t, common.ModelRoleEmbedding, run.selections.Selected["embed"].SelectedRole)
	assert.Equal(t, "amazon.titan-embed-text-v2:0", run.selections.Selected["embed"].Model)

	// A user-managed gateway is neither rewritten nor recorded.
	assert.Equal(t, []string{"--config", "/app/litellm.yaml"}, run.containers["custom"].Command)
	assert.NotContains(t, run.selections.Inputs, "custom")
	assert.Zero(t, run.catalog.calls)
}

func TestAwsModelRequestRejectsInvalidRecipe(t *testing.T) {
	server := testutil.MakeAwsTestServer()
	_, err := server.Construct(p.ConstructRequest{
		Urn:    testutil.AwsURN("Project"),
		Config: testutil.StackConfig("defang-aws:models", `{"chat-default": [{"model": "x", "location": "global"}]}`),
		Inputs: testutil.ComposeInputs(t, testutil.ModelFixture(t, "aws.yaml")),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `defang-aws:models.chat-default[0]: field "location" is not supported on aws`)
}
