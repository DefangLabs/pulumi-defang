package azure

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cognitiveservices/armcognitiveservices"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDeployable(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"generally-available", string(armcognitiveservices.ModelLifecycleStatusGenerallyAvailable), true},
		{"stable", string(armcognitiveservices.ModelLifecycleStatusStable), true},
		{"preview", string(armcognitiveservices.ModelLifecycleStatusPreview), true},
		{"deprecating-is-blocked", string(armcognitiveservices.ModelLifecycleStatusDeprecating), false},
		{"deprecated-is-blocked", string(armcognitiveservices.ModelLifecycleStatusDeprecated), false},
		{"deprecating-blocked-case-insensitively", "deprecating", false},
		{"empty-status-stays-eligible", "", true},
		{"unknown-future-status-stays-eligible", "SomeFutureStatus", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isDeployable(tc.status))
		})
	}
}

func TestFindBest(t *testing.T) {
	models := []availableModel{
		{name: "gpt-4o", format: "OpenAI", version: "2024-05-13", lifecycleStatus: "GenerallyAvailable"},
		{name: "gpt-4o", format: "OpenAI", version: "2024-08-06", lifecycleStatus: "GenerallyAvailable"},
		{name: "gpt-4o", format: "OpenAI", version: "2024-11-20", lifecycleStatus: "Deprecating"},
		// A higher version string exists but under the wrong format — must be ignored.
		{name: "gpt-4o", format: "Microsoft", version: "9999-99-99", lifecycleStatus: "GenerallyAvailable"},
	}
	tests := []struct {
		name        string
		modelName   string
		format      string
		wantNil     bool
		wantVersion string
	}{
		// Newest version (2024-11-20) is Deprecating, so the newest *deployable*
		// version wins instead of the lexically-highest one.
		{"highest-deployable-version", "gpt-4o", "OpenAI", false, "2024-08-06"},
		{"format-must-match", "gpt-4o", "Cohere", true, ""},
		{"name-must-match", "gpt-4", "OpenAI", true, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			best := findBest(models, tc.modelName, tc.format)
			if tc.wantNil {
				assert.Nil(t, best)
				return
			}
			if assert.NotNil(t, best) {
				assert.Equal(t, tc.wantVersion, best.version)
			}
		})
	}
}

// TestPickModelSkipsDeprecating is a regression for the gpt-4o 2024-11-20
// deprecation that broke `defang up` on Azure: ListModels still returns the
// deprecating version, but deploying it fails with HTTP 400
// ServiceModelDeprecating. pickModel must never return a Deprecating or
// Deprecated version.
func TestPickModelSkipsDeprecating(t *testing.T) {
	tests := []struct {
		name        string
		models      []availableModel
		wantName    string
		wantVersion string
	}{
		{
			name: "prefers-GA-gpt-5.1-over-deprecating-gpt-4o",
			models: []availableModel{
				{name: "gpt-4o", format: "OpenAI", version: "2024-11-20", lifecycleStatus: "Deprecating"},
				{name: "gpt-5.1", format: "OpenAI", version: "2025-11-13", lifecycleStatus: "GenerallyAvailable"},
			},
			wantName:    "gpt-5.1",
			wantVersion: "2025-11-13",
		},
		{
			name: "falls-back-to-older-GA-version-of-same-model",
			models: []availableModel{
				{name: "gpt-4o", format: "OpenAI", version: "2024-11-20", lifecycleStatus: "Deprecating"},
				{name: "gpt-4o", format: "OpenAI", version: "2024-08-06", lifecycleStatus: "GenerallyAvailable"},
			},
			wantName:    "gpt-4o",
			wantVersion: "2024-08-06",
		},
		{
			name: "cascades-past-a-fully-blocked-family",
			models: []availableModel{
				{name: "gpt-4o", format: "OpenAI", version: "2024-11-20", lifecycleStatus: "Deprecating"},
				{name: "gpt-4o-mini", format: "OpenAI", version: "2024-07-18", lifecycleStatus: "Deprecated"},
				{name: "gpt-4.1", format: "OpenAI", version: "2025-04-14", lifecycleStatus: "GenerallyAvailable"},
			},
			wantName:    "gpt-4.1",
			wantVersion: "2025-04-14",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := pickModel(tc.models, ModelRoleChat)
			require.NoError(t, err)
			assert.Equal(t, tc.wantName, spec.Name)
			assert.Equal(t, tc.wantVersion, spec.Version)
			assert.Equal(t, "OpenAI", spec.Format)
		})
	}
}

func TestPickModelNoDeployableModel(t *testing.T) {
	// Only a deprecating model is available: selection must fail with a clear
	// error rather than return a version Azure will reject at deploy time.
	models := []availableModel{
		{name: "gpt-4o", format: "OpenAI", version: "2024-11-20", lifecycleStatus: "Deprecating"},
	}
	_, err := pickModel(models, ModelRoleChat)
	assert.ErrorIs(t, err, ErrNoSuitableModel)
}
