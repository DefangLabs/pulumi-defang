package azure

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cognitiveservices/armcognitiveservices"
	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/DefangLabs/pulumi-defang/provider/modelselect"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAccountID = "/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.CognitiveServices/accounts/acct-1"

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

func TestPickRevision(t *testing.T) {
	models := []AvailableModel{
		{Name: "gpt-4o", Format: "OpenAI", Version: "2024-05-13", LifecycleStatus: "GenerallyAvailable"},
		{Name: "gpt-4o", Format: "OpenAI", Version: "2024-08-06", LifecycleStatus: "GenerallyAvailable"},
		{Name: "gpt-4o", Format: "OpenAI", Version: "2024-11-20", LifecycleStatus: "Deprecating"},
		{Name: "gpt-4o-mini", Format: "OpenAI", Version: "2025-01-01", LifecycleStatus: "GenerallyAvailable"},
		{Name: "Phi-4", Format: "Microsoft", Version: "9", LifecycleStatus: "GenerallyAvailable"},
		{Name: "Phi-4", Format: "Microsoft", Version: "10", LifecycleStatus: "GenerallyAvailable"},
		{Name: "gpt-5.2", Format: "OpenAI", Version: "2026-01-01", LifecycleStatus: "Preview"},
		{Name: "gpt-5.2", Format: "OpenAI", Version: "2025-12-01", LifecycleStatus: "GenerallyAvailable"},
		{Name: "dual", Format: "OpenAI", Version: "1", LifecycleStatus: "GenerallyAvailable"},
		{Name: "dual", Format: "Microsoft", Version: "1", LifecycleStatus: "GenerallyAvailable"},
	}
	tests := []struct {
		name        string
		candidate   compose.ModelCandidate
		wantVersion string
		wantErr     error
	}{
		// The newest version (2024-11-20) is Deprecating, so the newest
		// *deployable* version wins.
		{"highest-deployable-version", compose.ModelCandidate{Model: "gpt-4o", Format: "OpenAI"}, "2024-08-06", nil},
		{"format-must-match", compose.ModelCandidate{Model: "gpt-4o", Format: "Cohere"}, "", common.ErrModelUnavailable},
		// "gpt-4" must never select gpt-4o or gpt-4o-mini: exact names only.
		{"name-is-exact-not-a-prefix", compose.ModelCandidate{Model: "gpt-4", Format: "OpenAI"}, "",
			common.ErrModelUnavailable},
		{"name-is-case-insensitive", compose.ModelCandidate{Model: "GPT-4O"}, "2024-08-06", nil},
		// Revision 10 is newer than 9 even though "9" > "10" as strings.
		{"numeric-revision-order", compose.ModelCandidate{Model: "Phi-4"}, "10", nil},
		{"pinned-version", compose.ModelCandidate{Model: "Phi-4", Version: "9"}, "9", nil},
		{"pinned-version-missing", compose.ModelCandidate{Model: "Phi-4", Version: "11"}, "", common.ErrModelUnavailable},
		{"pinned-deprecating-version", compose.ModelCandidate{Model: "gpt-4o", Version: "2024-11-20"}, "",
			common.ErrModelUnavailable},
		{"preview-skipped-by-default", compose.ModelCandidate{Model: "gpt-5.2"}, "2025-12-01", nil},
		{"preview-allowed", compose.ModelCandidate{Model: "gpt-5.2", AllowPreview: true}, "2026-01-01", nil},
		{"pinned-preview-is-explicit", compose.ModelCandidate{Model: "gpt-5.2", Version: "2026-01-01"}, "2026-01-01", nil},
		{"ambiguous-format", compose.ModelCandidate{Model: "dual"}, "", common.ErrModelAmbiguous},
		{"format-resolves-ambiguity", compose.ModelCandidate{Model: "dual", Format: "Microsoft"}, "1", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, err := pickRevision(models, tc.candidate)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantVersion, m.Version)
		})
	}
}

// selectWith resolves a planned request against a fake account catalog
// through the same ModelSelection lifecycle the provider registers.
func selectWith(
	t *testing.T, models []AvailableModel, request string, recipe common.ModelRecipe,
) (common.ModelSelectionState, error) {
	t.Helper()
	plan, err := common.PlanModelRequest(ModelCloud, request, recipe)
	require.NoError(t, err)
	listed := 0
	resolver := &FoundryResolver{List: func(_ context.Context, sub, rg, account string) ([]AvailableModel, error) {
		listed++
		assert.Equal(t, []string{"sub-1", "rg-1", "acct-1"}, []string{sub, rg, account})
		return models, nil
	}}
	selector := modelselect.ModelSelector{Cloud: ModelCloud, Resolver: resolver, ScopeRequired: true}
	resp, err := selector.Create(context.Background(), infer.CreateRequest[common.ModelSelectionInputs]{
		Inputs: common.ModelSelectionInputs{
			Request: plan.Request, Role: plan.Role, Source: plan.Source, Candidates: plan.Candidates, Scope: testAccountID,
		},
	})
	assert.LessOrEqual(t, listed, 1, "the account catalog is listed at most once")
	return resp.Output, err
}

// TestDefaultSkipsDeprecating is a regression for the gpt-4o 2024-11-20
// deprecation that broke `defang up` on Azure: ListModels still returns the
// deprecating version, but deploying it fails with HTTP 400
// ServiceModelDeprecating.
func TestDefaultSkipsDeprecating(t *testing.T) {
	tests := []struct {
		name        string
		models      []AvailableModel
		wantName    string
		wantVersion string
	}{
		{
			name: "prefers-GA-gpt-5.1-over-deprecating-gpt-4o",
			models: []AvailableModel{
				{Name: "gpt-4o", Format: "OpenAI", Version: "2024-11-20", LifecycleStatus: "Deprecating"},
				{Name: "gpt-5.1", Format: "OpenAI", Version: "2025-11-13", LifecycleStatus: "GenerallyAvailable"},
			},
			wantName:    "gpt-5.1",
			wantVersion: "2025-11-13",
		},
		{
			name: "falls-back-to-older-GA-version-of-same-model",
			models: []AvailableModel{
				{Name: "gpt-4o", Format: "OpenAI", Version: "2024-11-20", LifecycleStatus: "Deprecating"},
				{Name: "gpt-4o", Format: "OpenAI", Version: "2024-08-06", LifecycleStatus: "GenerallyAvailable"},
			},
			wantName:    "gpt-4o",
			wantVersion: "2024-08-06",
		},
		{
			name: "cascades-past-a-fully-blocked-family",
			models: []AvailableModel{
				{Name: "gpt-4o", Format: "OpenAI", Version: "2024-11-20", LifecycleStatus: "Deprecating"},
				{Name: "gpt-4o-mini", Format: "OpenAI", Version: "2024-07-18", LifecycleStatus: "Deprecated"},
				{Name: "gpt-4.1", Format: "OpenAI", Version: "2025-04-14", LifecycleStatus: "GenerallyAvailable"},
			},
			wantName:    "gpt-4.1",
			wantVersion: "2025-04-14",
		},
		{
			// Preview revisions were always eligible for the built-in aliases.
			name: "default-keeps-preview-eligibility",
			models: []AvailableModel{
				{Name: "gpt-5.1", Format: "OpenAI", Version: "2025-11-13", LifecycleStatus: "Preview"},
				{Name: "gpt-4.1", Format: "OpenAI", Version: "2025-04-14", LifecycleStatus: "GenerallyAvailable"},
			},
			wantName:    "gpt-5.1",
			wantVersion: "2025-11-13",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, alias := range []string{"chat-default", "chat-large", "ai/chat-default"} {
				state, err := selectWith(t, tc.models, alias, nil)
				require.NoError(t, err, alias)
				assert.Equal(t, tc.wantName, state.Model, alias)
				assert.Equal(t, tc.wantVersion, state.Version, alias)
				assert.Equal(t, "OpenAI", state.Format, alias)
				assert.Equal(t, common.ModelSourceDefault, state.Source, alias)
				assert.Equal(t, common.ModelRoleChat, state.SelectedRole, alias)
			}
		})
	}
}

func TestDefaultNoDeployableModel(t *testing.T) {
	// Only a deprecating model is available: selection must fail with a clear
	// error rather than return a version Azure will reject at deploy time.
	models := []AvailableModel{
		{Name: "gpt-4o", Format: "OpenAI", Version: "2024-11-20", LifecycleStatus: "Deprecating"},
	}
	_, err := selectWith(t, models, "chat-default", nil)
	require.ErrorIs(t, err, common.ErrModelUnavailable)
	assert.Contains(t, err.Error(), "defang-azure:models.chat-default")
}

func TestEmbeddingDefaultIsEmbeddingRole(t *testing.T) {
	models := []AvailableModel{
		{Name: "text-embedding-3-small", Format: "OpenAI", Version: "1", LifecycleStatus: "GenerallyAvailable",
			SKUs: []string{"Standard"}, Capabilities: map[string]string{"embeddings": "true"}},
	}
	state, err := selectWith(t, models, "embedding-default", nil)
	require.NoError(t, err)
	assert.Equal(t, "text-embedding-3-small", state.Model)
	assert.Equal(t, "Standard", state.Sku)
	assert.Equal(t, common.ModelRoleEmbedding, state.SelectedRole)
}

func TestRecipeSelection(t *testing.T) {
	models := []AvailableModel{
		{Name: "gpt-5.1", Format: "OpenAI", Version: "2025-11-13", LifecycleStatus: "GenerallyAvailable",
			SKUs: []string{"GlobalStandard", "DataZoneStandard"}, Capabilities: map[string]string{"chatCompletion": "true"}},
		{Name: "gpt-4.1", Format: "OpenAI", Version: "2025-04-14", LifecycleStatus: "GenerallyAvailable",
			SKUs: []string{"Standard"}, Capabilities: map[string]string{"chatCompletion": "true"}},
		{Name: "text-embedding-3-large", Format: "OpenAI", Version: "1", LifecycleStatus: "GenerallyAvailable",
			Capabilities: map[string]string{"embeddings": "true"}},
	}
	recipe, err := common.ParseModelRecipe(ModelCloud, `{
		"chat-large": [{"model": "gpt-4.1", "format": "OpenAI"}],
		"zone-chat": [{"model": "gpt-5.1", "sku": "DataZoneStandard"}],
		"pinned-missing": [{"model": "gpt-5.1", "version": "2024-01-01"}, {"model": "gpt-4.1"}],
		"wrong-role": [{"model": "text-embedding-3-large", "role": "chat"}],
		"bad-sku": [{"model": "gpt-4.1", "sku": "ProvisionedManaged"}, {"model": "gpt-5.1"}]
	}`)
	require.NoError(t, err)

	t.Run("chat-large-has-its-own-entry", func(t *testing.T) {
		state, err := selectWith(t, models, "chat-large", recipe)
		require.NoError(t, err)
		assert.Equal(t, "gpt-4.1", state.Model)
		assert.Equal(t, common.ModelSourceRecipe, state.Source)
	})
	t.Run("explicit-sku", func(t *testing.T) {
		state, err := selectWith(t, models, "zone-chat", recipe)
		require.NoError(t, err)
		assert.Equal(t, "DataZoneStandard", state.Sku)
		assert.Equal(t, common.ModelRoleChat, state.SelectedRole, "role comes from capabilities")
	})
	t.Run("default-sku-without-standard", func(t *testing.T) {
		state, err := selectWith(t, models, "gpt-5.1", recipe)
		require.NoError(t, err)
		assert.Equal(t, "GlobalStandard", state.Sku)
		assert.Equal(t, common.ModelSourceNative, state.Source)
	})
	t.Run("pinned-version-never-falls-through", func(t *testing.T) {
		_, err := selectWith(t, models, "pinned-missing", recipe)
		require.ErrorIs(t, err, common.ErrModelUnavailable)
		assert.Contains(t, err.Error(), "2024-01-01")
	})
	t.Run("role-mismatch", func(t *testing.T) {
		_, err := selectWith(t, models, "wrong-role", recipe)
		require.ErrorIs(t, err, common.ErrModelRequest)
	})
	t.Run("unoffered-sku-falls-through-for-floating", func(t *testing.T) {
		state, err := selectWith(t, models, "bad-sku", recipe)
		require.NoError(t, err)
		assert.Equal(t, "gpt-5.1", state.Model)
	})
}

func TestParseAccountID(t *testing.T) {
	account, err := parseAccountID(testAccountID)
	require.NoError(t, err)
	assert.Equal(t, accountRef{subscriptionID: "sub-1", resourceGroup: "rg-1", name: "acct-1"}, account)
	_, err = parseAccountID("acct-1")
	require.ErrorIs(t, err, ErrInvalidAccountScope)
}
