package defangazure

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// TestLlmModelAlias covers how the Azure AI Foundry deployment gets its name.
// Dependent services send that name as the "model" field, so a wrong answer here
// surfaces as DeploymentNotFound at runtime, after a deploy that looked green.
func TestLlmModelAlias(t *testing.T) {
	litellmCmd := []string{"--drop_params", "--model", "chat-default", "--alias", "chat-default"}

	for _, tt := range []struct {
		name     string
		services compose.Services
		want     string
	}{
		{
			// The regression: compose lets a dependent rename the injected variable
			// (models: {llm: {model_var: MODEL}}). The old env-var scan looked only for
			// LLM_MODEL, missed it, and silently deployed under the service name while
			// the CLI told the app to ask for "chat-default".
			name: "alias comes from the command even when model_var is renamed",
			services: compose.Services{
				"llm": {Command: litellmCmd, LLM: &compose.LlmConfig{}},
				"app": {Environment: compose.Environment{"MODEL": pulumi.String("chat-default")}},
			},
			want: "chat-default",
		},
		{
			// Precedence: the command must win over the env var. The two carry
			// different values here on purpose — identical ones would pass whichever
			// source were consulted and prove nothing.
			name: "the command wins over a conflicting env var",
			services: compose.Services{
				"llm": {Command: litellmCmd, LLM: &compose.LlmConfig{}},
				"app": {Environment: compose.Environment{"LLM_MODEL": pulumi.String("stale-env-alias")}},
			},
			want: "chat-default",
		},
		{
			// Older CLIs emit no --alias; the env-var scan is still the fallback.
			name: "falls back to the injected env var when the command has no alias",
			services: compose.Services{
				"llm": {LLM: &compose.LlmConfig{}},
				"app": {Environment: compose.Environment{"LLM_MODEL": pulumi.String("chat-default")}},
			},
			want: "chat-default",
		},
		{
			name: "falls back to the service name when nothing else is available",
			services: compose.Services{
				"llm": {LLM: &compose.LlmConfig{}},
				"app": {},
			},
			want: "llm",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := llmModelAlias("llm", tt.services); got != tt.want {
				t.Errorf("llmModelAlias() = %q, want %q", got, tt.want)
			}
		})
	}
}
