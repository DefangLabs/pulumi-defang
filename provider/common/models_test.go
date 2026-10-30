package common

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testCloud = ModelCloud{
	Name:      "testcloud",
	Namespace: "defang-test",
	Fields:    []string{ModelFieldLocation},
	Defaults: map[string][]compose.ModelCandidate{
		"chat-default":      {{Model: "default-chat:1"}},
		"embedding-default": {{Model: "default-embed:1"}},
	},
	NativeExact: func(m string) bool { return len(m) > 2 && m[len(m)-2] == ':' },
}

func TestParseModelRecipe(t *testing.T) {
	recipe, err := ParseModelRecipe(testCloud, `{
		"chat-default": [{"model": "a:1", "location": "global"}, {"model": "b:1"}],
		"my-embed": [{"model": "e:2", "role": "embedding"}]
	}`)
	require.NoError(t, err)
	assert.Equal(t, []compose.ModelCandidate{{Model: "a:1", Location: "global"}, {Model: "b:1"}}, recipe["chat-default"])
	assert.Equal(t, "embedding", recipe["my-embed"][0].Role)

	// YAML `version: 1` arrives as a number and means revision "1".
	azureLike := ModelCloud{
		Name: "azure", Namespace: "defang-azure", Fields: []string{ModelFieldVersion, ModelFieldAllowPreview},
	}
	recipe, err = ParseModelRecipe(azureLike,
		`{"pinned": [{"model": "m", "version": 1, "role": "chat", "allow-preview": true}]}`)
	require.NoError(t, err)
	assert.Equal(t, compose.ModelCandidate{Model: "m", Version: "1", Role: "chat", AllowPreview: true},
		recipe["pinned"][0])
}

func TestParseModelRecipeErrors(t *testing.T) {
	tests := []struct {
		name, raw, want string
	}{
		{"not-a-map", `["chat-default"]`, "must map each selector"},
		{"not-a-list", `{"chat-default": {"model": "x"}}`, "defang-test:models.chat-default must be a non-empty list"},
		{"empty-list", `{"chat-default": []}`, "non-empty list"},
		{"missing-model", `{"chat-default": [{"location": "x"}]}`,
			"defang-test:models.chat-default[0]: `model` is required"},
		{"unsupported-field", `{"chat-default": [{"model": "x", "format": "OpenAI"}]}`,
			`field "format" is not supported on testcloud`},
		{"bad-role", `{"k": [{"model": "x", "role": "image"}]}`, `role "image" must be`},
		{"builtin-role-mismatch", `{"embedding-default": [{"model": "x", "role": "chat"}]}`, "reserved for embedding models"},
		{"mixed-roles", `{"k": [{"model": "x", "role": "chat"}, {"model": "y", "role": "embedding"}]}`, "must share a role"},
		{"interpolation", `{"k": [{"model": "${MODEL}", "role": "chat"}]}`, "unresolved interpolation"},
		{"non-string", `{"k": [{"model": 5, "role": "chat"}, {"model": true}]}`, "must be a string"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseModelRecipe(testCloud, tc.raw)
			require.ErrorIs(t, err, ErrModelRecipe)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestNormalizeModelRequest(t *testing.T) {
	assert.Equal(t, "chat-default", NormalizeModelRequest("ai/chat-default"))
	assert.Equal(t, "chat-large", NormalizeModelRequest("ai/chat-large"))
	assert.Equal(t, "embedding-default", NormalizeModelRequest(" ai/embedding-default "))
	// Arbitrary artifact identifiers keep their prefix.
	assert.Equal(t, "ai/gemma3", NormalizeModelRequest("ai/gemma3"))
	assert.Equal(t, "us.amazon.nova-2-lite-v1:0", NormalizeModelRequest("us.amazon.nova-2-lite-v1:0"))
}

func TestPlanModelRequest(t *testing.T) {
	recipe := ModelRecipe{
		"chat-default": {{Model: "recipe-chat:3"}},
		"family":       {{Model: "fam", Role: ModelRoleChat}},
		"x:1":          {{Model: "should-not-be-used:9"}},
		"no-role":      {{Model: "nr"}},
	}
	tests := []struct {
		name, request  string
		recipe         ModelRecipe
		wantSource     string
		wantRole       string
		wantCandidates []compose.ModelCandidate
	}{
		{"recipe-alias", "chat-default", recipe, ModelSourceRecipe, ModelRoleChat, recipe["chat-default"]},
		{"ai-spelling", "ai/chat-default", recipe, ModelSourceRecipe, ModelRoleChat, recipe["chat-default"]},
		{"custom-family-key", "family", recipe, ModelSourceRecipe, ModelRoleChat, recipe["family"]},
		// A native exact ID is never rewritten by a recipe key of the same name.
		{"native-exact-wins", "x:1", recipe, ModelSourceNative, "", []compose.ModelCandidate{{Model: "x:1"}}},
		// Missing built-ins fall back to compatibility defaults; the recipe is not merged.
		{"default-for-missing-builtin", "embedding-default", recipe, ModelSourceDefault, ModelRoleEmbedding,
			testCloud.Defaults["embedding-default"]},
		{"older-recipe-without-models", "chat-default", nil, ModelSourceDefault, ModelRoleChat,
			testCloud.Defaults["chat-default"]},
		{"native-name", "gemini-pro", recipe, ModelSourceNative, "", []compose.ModelCandidate{{Model: "gemini-pro"}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := PlanModelRequest(testCloud, tc.request, tc.recipe)
			require.NoError(t, err)
			assert.Equal(t, tc.wantSource, plan.Source)
			assert.Equal(t, tc.wantRole, plan.Role)
			assert.Equal(t, tc.wantCandidates, plan.Candidates)
		})
	}

	t.Run("custom-key-needs-role-without-catalog", func(t *testing.T) {
		_, err := PlanModelRequest(testCloud, "no-role", recipe)
		require.ErrorIs(t, err, ErrModelRecipe)
		assert.Contains(t, err.Error(), "role: chat")
	})
	t.Run("custom-key-role-from-catalog", func(t *testing.T) {
		withCatalog := testCloud
		withCatalog.RoleFromCatalog = true
		plan, err := PlanModelRequest(withCatalog, "no-role", recipe)
		require.NoError(t, err)
		assert.Empty(t, plan.Role)
	})
	t.Run("missing-default", func(t *testing.T) {
		_, err := PlanModelRequest(testCloud, "chat-large", nil)
		require.ErrorIs(t, err, ErrModelRequest)
		assert.Contains(t, err.Error(), "defang-test:models.chat-large")
	})
	t.Run("unresolved-interpolation", func(t *testing.T) {
		_, err := PlanModelRequest(testCloud, "${MODEL_NAME}", nil)
		require.ErrorIs(t, err, ErrModelRequest)
	})
}

func TestCompareRevisions(t *testing.T) {
	assert.Positive(t, CompareRevisions("10", "9"))
	assert.Negative(t, CompareRevisions("9", "10"))
	assert.Positive(t, CompareRevisions("2025-04-14", "2024-11-20"))
	assert.Positive(t, CompareRevisions("2024-11-20", "2024-08-06"))
	assert.Zero(t, CompareRevisions("0613", "613"))
	assert.Positive(t, CompareRevisions("1.10", "1.9"))
	assert.Positive(t, CompareRevisions("2-preview", "2"))
	assert.Zero(t, CompareRevisions("", ""))
}

func TestGatewayCommand(t *testing.T) {
	assert.Equal(t, []string{"--drop_params", "--model", "bedrock/us.amazon.nova-2-lite-v1:0", "--alias", "chat-default"},
		GatewayCommand("bedrock", "us.amazon.nova-2-lite-v1:0", "chat-default"))
	// Never doubled; ARNs (which contain '/') still get the provider prefix.
	assert.Equal(t, "bedrock/x:0", GatewayCommand("bedrock", "bedrock/x:0", "x")[2])
	assert.Equal(t, "bedrock/arn:aws:bedrock:us-east-1:1:application-inference-profile/abc",
		GatewayCommand("bedrock", "arn:aws:bedrock:us-east-1:1:application-inference-profile/abc", "a")[2])
}

func TestLegacyGateway(t *testing.T) {
	svc := compose.ServiceConfig{
		Image:   pulumi.String("litellm/litellm:v1.92.0"),
		Command: []string{"--drop_params", "--model", "vertex_ai/gemini-embedding-001", "--alias", "embedding-default"},
	}
	target, alias, ok := LegacyGateway(svc, "vertex_ai")
	require.True(t, ok)
	assert.Equal(t, "gemini-embedding-001", target)
	assert.Equal(t, "embedding-default", alias)

	userManaged := svc
	userManaged.Command = []string{"--config", "/app/config.yaml"}
	_, _, ok = LegacyGateway(userManaged, "vertex_ai")
	assert.False(t, ok, "a user-managed LiteLLM command is never recognized")

	otherImage := svc
	otherImage.Image = pulumi.String("ghcr.io/me/litellm:1")
	_, _, ok = LegacyGateway(otherImage, "vertex_ai")
	assert.False(t, ok)
}

func TestInjectModelVar(t *testing.T) {
	services := compose.Services{
		"app": {
			Environment: compose.Environment{"LLM_URL": pulumi.String("https://api.scaleway.ai/v1/")},
			Models:      map[string]*compose.ServiceModelConfig{"llm": {ModelVar: "LLM_MODEL", EndpointVar: "LLM_URL"}},
		},
		"worker": {
			Environment: compose.Environment{"MODEL": pulumi.String("user-choice")},
			Models:      map[string]*compose.ServiceModelConfig{"llm": {ModelVar: "MODEL", EndpointVar: "BASE_URL"}},
		},
		"other": {Environment: compose.Environment{"LLM_MODEL": pulumi.String("unrelated")}},
	}
	original := services["app"].Environment
	require.NoError(t, InjectModelVar(services, "llm", pulumi.String("qwen3-235b-a22b-instruct-2507")))
	assert.Equal(t, pulumi.String("qwen3-235b-a22b-instruct-2507"), services["app"].Environment["LLM_MODEL"])
	assert.Equal(t, pulumi.String("user-choice"), services["worker"].Environment["MODEL"], "a set variable is the user's")
	assert.Equal(t, pulumi.String("unrelated"), services["other"].Environment["LLM_MODEL"], "unbound services untouched")
	assert.NotContains(t, original, "LLM_MODEL", "the input environment is not mutated")

	// Variable names are never derived: a binding must name its model variable.
	services["app"] = compose.ServiceConfig{Models: map[string]*compose.ServiceModelConfig{"llm": nil}}
	require.ErrorIs(t, InjectModelVar(services, "llm", pulumi.String("x")), ErrModelRequest)
}

func TestIsManagedGateway(t *testing.T) {
	models := compose.Models{"chat": {Model: "chat-default"}}
	gateway := compose.ServiceConfig{LLM: &compose.LlmConfig{}, Image: pulumi.String("litellm/litellm:v1.92.0")}
	assert.True(t, IsManagedGateway("chat", gateway, models))

	legacy := gateway
	legacy.Command = []string{"--drop_params", "--model", "bedrock/x:0", "--alias", "chat-default"}
	assert.False(t, IsManagedGateway("chat", legacy, models), "an older CLI's resolved gateway")
	assert.False(t, IsManagedGateway("custom", gateway, models), "not named after a model: user-managed")
	assert.False(t, IsManagedGateway("chat", compose.ServiceConfig{Image: pulumi.String("x")}, models))
}
