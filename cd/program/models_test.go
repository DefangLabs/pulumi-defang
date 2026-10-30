package program

import (
	"encoding/json"
	"io/fs"
	"os"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	awscompose "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws/compose"
	scalewaycompose "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-scaleway/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The payloads the provider tests deploy; parsed here with the CD's own
// decoder, the path every real deployment takes.
func modelPayload(t *testing.T, name string) []byte {
	t.Helper()
	data, err := fs.ReadFile(os.DirFS("../../tests/testdata/model-requests"), name)
	require.NoError(t, err)
	return data
}

func TestParseModelPayloads(t *testing.T) {
	for _, fixture := range []string{"aws.yaml", "gcp.yaml", "azure.yaml", "scaleway.yaml"} {
		t.Run(fixture, func(t *testing.T) {
			project, err := parseCompose(modelPayload(t, fixture), "modeldemo")
			require.NoError(t, err)
			assert.Equal(t, "ai/chat-default", project.Models["chat"].Model)
			assert.Equal(t, "chat-large", project.Models["large"].Model)
			assert.Equal(t, &compose.ServiceModelConfig{ModelVar: "EMBED_MODEL_NAME", EndpointVar: "EMBED_BASE_URL"},
				project.Services["app"].Models["embed"])
			if fixture != "scaleway.yaml" {
				assert.NotNil(t, project.Services["chat"].LLM, "scalar x-defang-llm")
				assert.Empty(t, project.Services["chat"].Command)
			}
		})
	}
	legacy, err := parseCompose(modelPayload(t, "legacy-aws.yaml"), "modeldemo")
	require.NoError(t, err)
	assert.Equal(t, "--model", legacy.Services["chat"].Command[1])
	assert.NotNil(t, legacy.Services["custom"].LLM, "an object x-defang-llm still decodes")
}

func TestToArgsCarryModels(t *testing.T) {
	project, err := parseCompose(modelPayload(t, "aws.yaml"), "modeldemo")
	require.NoError(t, err)
	applySeeds(project, ModelSeeds{"embed": {Model: "amazon.titan-embed-text-v2:0", Existed: true}})

	args := toAWSArgs(project, "etag")
	models, ok := args.Models.(awscompose.ModelConfigMap)
	require.True(t, ok)
	require.Len(t, models, 4)
	chat := models["chat"].(awscompose.ModelConfigArgs)
	assert.Equal(t, pulumi.String("ai/chat-default"), chat.Model)
	assert.Nil(t, chat.Seed)
	seed := models["embed"].(awscompose.ModelConfigArgs).Seed.(awscompose.ModelTargetArgs)
	assert.Equal(t, pulumi.String("amazon.titan-embed-text-v2:0"), seed.Model)
	assert.Equal(t, pulumi.Bool(true), seed.Existed)

	app := args.Services.(awscompose.ServiceConfigMap)["app"].(awscompose.ServiceConfigArgs)
	binding := app.Models.(awscompose.ServiceModelConfigMap)["chat"].(awscompose.ServiceModelConfigArgs)
	assert.Equal(t, pulumi.String("CHAT_MODEL"), binding.ModelVar)
	assert.Equal(t, pulumi.String("CHAT_URL"), binding.EndpointVar)

	scw, err := parseCompose(modelPayload(t, "scaleway.yaml"), "modeldemo")
	require.NoError(t, err)
	scwArgs := toScalewayArgs(scw, "etag")
	assert.Len(t, scwArgs.Models.(scalewaycompose.ModelConfigMap), 4)
	worker := scwArgs.Services.(scalewaycompose.ServiceConfigMap)["worker"].(scalewaycompose.ServiceConfigArgs)
	assert.Len(t, worker.Models.(scalewaycompose.ServiceModelConfigMap), 3)
}

// checkpoint builds the deployment object of `pulumi stack export`.
func checkpoint(t *testing.T, resources ...map[string]any) []byte {
	t.Helper()
	data, err := json.Marshal(map[string]any{"resources": resources})
	require.NoError(t, err)
	return data
}

const urnPrefix = "urn:pulumi:prod::modeldemo::"

func TestSeedsFromState(t *testing.T) {
	project := "urn:pulumi:prod::modeldemo::defang-aws:index:Project::modeldemo"

	t.Run("aws-task-definition", func(t *testing.T) {
		chatSvc := urnPrefix + "defang-aws:index:Project$defang-aws:index:Service::chat"
		containerDefs, err := json.Marshal([]map[string]any{{
			"name": "chat", "command": []string{"--drop_params", "--model", "bedrock/amazon.titan-embed-text-v2:0",
				"--alias", "embedding-default"},
		}})
		require.NoError(t, err)
		state := checkpoint(t,
			map[string]any{"urn": project, "type": "defang-aws:index:Project"},
			map[string]any{"urn": chatSvc, "type": "defang-aws:index:Service", "parent": project},
			map[string]any{
				"urn": chatSvc + "$aws:ecs/taskDefinition:TaskDefinition::chat", "type": "aws:ecs/taskDefinition:TaskDefinition",
				"parent": chatSvc, "inputs": map[string]any{"containerDefinitions": string(containerDefs)},
			},
		)
		seeds, err := SeedsFromState(state, modelPayload(t, "aws.yaml"), "modeldemo")
		require.NoError(t, err)
		assert.Equal(t, &compose.ModelTarget{Model: "amazon.titan-embed-text-v2:0", Existed: true}, seeds["chat"])
		assert.NotContains(t, seeds, "embed", "a model never deployed has no seed")
	})

	t.Run("gcp-cloud-init", func(t *testing.T) {
		svc := urnPrefix + "defang-gcp:index:Project$defang-gcp:index:Service::embed"
		userData := "ExecStart=/usr/bin/docker run --pull=always --rm --name=embed -p 4000:4000/tcp " +
			"-e VERTEXAI_LOCATION=us-central1 -e VERTEXAI_PROJECT=p litellm/litellm:v1.92.0 --drop_params " +
			"--model vertex_ai/gemini-embedding-001 --alias embedding-default"
		state := checkpoint(t,
			map[string]any{"urn": svc, "type": "defang-gcp:index:Service"},
			map[string]any{
				"urn": svc + "$gcp:compute/instanceTemplate:InstanceTemplate::embed", "parent": svc,
				"type":   "gcp:compute/instanceTemplate:InstanceTemplate",
				"inputs": map[string]any{"metadata": map[string]any{"user-data": userData}},
			},
		)
		seeds, err := SeedsFromState(state, modelPayload(t, "gcp.yaml"), "modeldemo")
		require.NoError(t, err)
		assert.Equal(t, &compose.ModelTarget{Model: "gemini-embedding-001", Location: "us-central1", Existed: true},
			seeds["embed"])
	})

	t.Run("azure-foundry-deployment", func(t *testing.T) {
		llm := urnPrefix + "defang-azure:index:Project$defang-azure:index:LLM::embed"
		state := checkpoint(t,
			map[string]any{"urn": llm, "type": "defang-azure:index:LLM"},
			map[string]any{
				"urn": llm + "$azure-native:cognitiveservices:Deployment::embedding-default", "parent": llm,
				"type": "azure-native:cognitiveservices:Deployment",
				"outputs": map[string]any{
					"properties": map[string]any{"model": map[string]any{
						"name": "text-embedding-3-small", "version": "1", "format": "OpenAI",
					}},
					"sku": map[string]any{"name": "Standard"},
				},
			},
		)
		seeds, err := SeedsFromState(state, modelPayload(t, "azure.yaml"), "modeldemo")
		require.NoError(t, err)
		assert.Equal(t, &compose.ModelTarget{
			Model: "text-embedding-3-small", Version: "1", Format: "OpenAI", Sku: "Standard", Existed: true,
		}, seeds["embed"])
	})

	t.Run("scaleway-consumer-env", func(t *testing.T) {
		app := urnPrefix + "defang-scaleway:index:Project$defang-scaleway:index:Service::app"
		state := checkpoint(t,
			map[string]any{"urn": app, "type": "defang-scaleway:index:Service"},
			map[string]any{
				"urn": app + "$scaleway:containers/container:Container::app", "parent": app,
				"type": "scaleway:containers/container:Container",
				"inputs": map[string]any{"environmentVariables": map[string]any{
					"EMBED_MODEL_NAME": "bge-multilingual-gemma2", "CHAT_MODEL": "qwen3-235b-a22b-instruct-2507",
				}},
			},
		)
		seeds, err := SeedsFromState(state, modelPayload(t, "scaleway.yaml"), "modeldemo")
		require.NoError(t, err)
		assert.Equal(t, &compose.ModelTarget{Model: "bge-multilingual-gemma2", Existed: true}, seeds["embed"])
		assert.Equal(t, &compose.ModelTarget{Model: "qwen3-235b-a22b-instruct-2507", Existed: true}, seeds["chat"])
	})
}

func TestSeedsFromStateWithoutTarget(t *testing.T) {
	t.Run("persisted-selection-needs-no-seed", func(t *testing.T) {
		svc := urnPrefix + "defang-aws:index:Service::chat"
		state := checkpoint(t,
			map[string]any{"urn": svc, "type": "defang-aws:index:Service"},
			map[string]any{
				"urn": urnPrefix + "defang-aws:index:ModelSelection::chat", "type": "defang-aws:index:ModelSelection",
			},
		)
		seeds, err := SeedsFromState(state, modelPayload(t, "aws.yaml"), "modeldemo")
		require.NoError(t, err)
		assert.NotContains(t, seeds, "chat")
	})

	t.Run("existing-service-without-known-target", func(t *testing.T) {
		svc := urnPrefix + "defang-aws:index:Service::embed"
		state := checkpoint(t, map[string]any{"urn": svc, "type": "defang-aws:index:Service"})
		seeds, err := SeedsFromState(state, modelPayload(t, "aws.yaml"), "modeldemo")
		require.NoError(t, err)
		assert.Equal(t, &compose.ModelTarget{Existed: true}, seeds["embed"],
			"existence without a target stops embedding migration")
	})

	t.Run("new-stack", func(t *testing.T) {
		seeds, err := SeedsFromState(nil, modelPayload(t, "aws.yaml"), "modeldemo")
		require.NoError(t, err)
		assert.Empty(t, seeds)
	})
}
