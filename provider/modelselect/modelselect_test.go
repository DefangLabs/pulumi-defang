package modelselect

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	createRequest = infer.CreateRequest[common.ModelSelectionInputs]
	updateRequest = infer.UpdateRequest[common.ModelSelectionInputs, common.ModelSelectionState]
	diffRequest   = infer.DiffRequest[common.ModelSelectionInputs, common.ModelSelectionState]
)

var errAccessDenied = errors.New("AccessDeniedException")

// fakeResolver resolves a candidate to the newest revision in latest
// ("family" -> "family:7"), or reports it unavailable / failing.
type fakeResolver struct {
	latest      map[string]string
	roles       map[string]string
	unavailable map[string]bool
	fail        error
	calls       int
}

func (f *fakeResolver) ResolveCandidate(
	_ context.Context, _ common.ModelSelectionInputs, c compose.ModelCandidate,
) (common.ResolvedModel, error) {
	f.calls++
	if f.fail != nil {
		return common.ResolvedModel{}, f.fail
	}
	if f.unavailable[c.Model] {
		return common.ResolvedModel{}, fmt.Errorf("%w: %s", common.ErrModelUnavailable, c.Model)
	}
	model := c.Model
	if l, ok := f.latest[c.Model]; ok {
		model = l
	}
	return common.ResolvedModel{Target: compose.ModelTarget{Model: model, Version: c.Version}, Role: f.roles[model]}, nil
}

var exactCloud = common.ModelCloud{
	Name: "test", Namespace: "defang-test",
	CandidateExact: func(c compose.ModelCandidate) bool { return c.Version != "" },
}

func inputs(request, role string, candidates ...compose.ModelCandidate) common.ModelSelectionInputs {
	return common.ModelSelectionInputs{
		Request: request, Role: role, Source: common.ModelSourceRecipe, Candidates: candidates,
	}
}

func TestResolveFallsThroughOnlyOnUnavailability(t *testing.T) {
	r := &fakeResolver{unavailable: map[string]bool{"gone": true}}
	ms := ModelSelector{Cloud: exactCloud, Resolver: r}

	state, err := ms.Resolve(context.Background(), inputs("chat-default", "chat",
		compose.ModelCandidate{Model: "gone"}, compose.ModelCandidate{Model: "next"}))
	require.NoError(t, err)
	assert.Equal(t, "next", state.Model)

	// A pinned candidate never falls through to an unrelated model.
	_, err = ms.Resolve(context.Background(), inputs("chat-default", "chat",
		compose.ModelCandidate{Model: "gone", Version: "1"}, compose.ModelCandidate{Model: "next"}))
	require.ErrorIs(t, err, common.ErrModelUnavailable)

	// Access denied, network and quota errors fail instead of falling through.
	ms.Resolver = &fakeResolver{fail: errAccessDenied}
	_, err = ms.Resolve(context.Background(), inputs("chat-default", "chat",
		compose.ModelCandidate{Model: "a"}, compose.ModelCandidate{Model: "b"}))
	require.ErrorIs(t, err, errAccessDenied)

	ms.Resolver = &fakeResolver{unavailable: map[string]bool{"a": true, "b": true}}
	_, err = ms.Resolve(context.Background(), inputs("chat-default", "chat",
		compose.ModelCandidate{Model: "a"}, compose.ModelCandidate{Model: "b"}))
	require.ErrorIs(t, err, common.ErrModelUnavailable)
	assert.Contains(t, err.Error(), "defang-test:models.chat-default")
}

func TestUnchangedInputsKeepSelection(t *testing.T) {
	r := &fakeResolver{latest: map[string]string{"fam": "fam:7"}}
	ms := ModelSelector{Cloud: exactCloud, Resolver: r}
	in := inputs("chat-default", "chat", compose.ModelCandidate{Model: "fam"})
	created, err := ms.Create(context.Background(), createRequest{Inputs: in})
	require.NoError(t, err)
	assert.Equal(t, "fam:7", created.Output.Model)

	// The catalog moves on, but an unchanged request does not re-resolve.
	r.latest["fam"] = "fam:8"
	calls := r.calls
	diff, err := ms.Diff(context.Background(), diffRequest{
		State: created.Output, Inputs: in,
	})
	require.NoError(t, err)
	assert.False(t, diff.HasChanges)
	updated, err := ms.Update(context.Background(), updateRequest{
		State: created.Output, Inputs: in,
	})
	require.NoError(t, err)
	assert.Equal(t, "fam:7", updated.Output.Model)
	assert.Equal(t, calls, r.calls, "no discovery for unchanged inputs")

	// A changed seed alone is not a change.
	seeded := in
	seeded.Seed = &compose.ModelTarget{Model: "other", Existed: true}
	diff, err = ms.Diff(context.Background(), diffRequest{
		State: created.Output, Inputs: seeded,
	})
	require.NoError(t, err)
	assert.False(t, diff.HasChanges)
}

func TestChangedChatRecipeReresolves(t *testing.T) {
	r := &fakeResolver{latest: map[string]string{"fam": "fam:7", "new": "new:1"}}
	ms := ModelSelector{Cloud: exactCloud, Resolver: r}
	in := inputs("chat-default", "chat", compose.ModelCandidate{Model: "fam"})
	created, err := ms.Create(context.Background(), createRequest{Inputs: in})
	require.NoError(t, err)

	changed := inputs("chat-default", "chat", compose.ModelCandidate{Model: "new"})
	diff, err := ms.Diff(context.Background(), diffRequest{
		State: created.Output, Inputs: changed,
	})
	require.NoError(t, err)
	assert.True(t, diff.HasChanges)
	assert.Contains(t, diff.DetailedDiff, "candidates")
	updated, err := ms.Update(context.Background(), updateRequest{
		State: created.Output, Inputs: changed, DryRun: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "new:1", updated.Output.Model, "preview shows the proposed target")
}

func TestEmbeddingProtection(t *testing.T) {
	r := &fakeResolver{latest: map[string]string{"embed": "embed:1"}}
	ms := ModelSelector{Cloud: exactCloud, Resolver: r}
	in := inputs("embedding-default", "embedding", compose.ModelCandidate{Model: "embed"})
	created, err := ms.Create(context.Background(), createRequest{Inputs: in})
	require.NoError(t, err)
	assert.Equal(t, "embed:1", created.Output.Model)
	old := created.Output

	update := func(in common.ModelSelectionInputs) (common.ModelSelectionState, error) {
		resp, err := ms.Update(context.Background(), updateRequest{
			State: old, Inputs: in,
		})
		return resp.Output, err
	}

	t.Run("recipe-change-keeps-model", func(t *testing.T) {
		r.latest["embed-v2"] = "embed-v2:1"
		state, err := update(inputs("embedding-default", "embedding", compose.ModelCandidate{Model: "embed-v2"}))
		require.NoError(t, err)
		assert.Equal(t, "embed:1", state.Model)
		assert.Equal(t, "embed-v2", state.Candidates[0].Model, "the new recipe entry is recorded")
	})
	t.Run("floating-request-is-refused", func(t *testing.T) {
		_, err := update(inputs("other-family", "embedding", compose.ModelCandidate{Model: "embed-v2"}))
		require.ErrorIs(t, err, common.ErrEmbeddingMigration)
	})
	t.Run("explicit-exact-request-migrates", func(t *testing.T) {
		state, err := update(inputs("embed-pinned", "embedding", compose.ModelCandidate{Model: "embed-v2", Version: "3"}))
		require.NoError(t, err)
		assert.Equal(t, "embed-v2:1", state.Model)
		assert.Equal(t, "3", state.Version)
	})
	t.Run("chat-is-not-protected", func(t *testing.T) {
		chat := old
		chat.SelectedRole = common.ModelRoleChat
		chat.Role = common.ModelRoleChat
		resp, err := ms.Update(context.Background(), updateRequest{
			State: chat, Inputs: inputs("embedding-default", "chat", compose.ModelCandidate{Model: "embed-v2"}),
		})
		require.NoError(t, err)
		assert.Equal(t, "embed-v2:1", resp.Output.Model)
	})
}

func TestFirstUpgradeSeed(t *testing.T) {
	r := &fakeResolver{latest: map[string]string{"embed": "embed:2"}}
	ms := ModelSelector{Cloud: exactCloud, Resolver: r}
	create := func(in common.ModelSelectionInputs) (common.ModelSelectionState, error) {
		resp, err := ms.Create(context.Background(), createRequest{Inputs: in})
		return resp.Output, err
	}

	in := inputs("embedding-default", "embedding", compose.ModelCandidate{Model: "embed"})
	in.Seed = &compose.ModelTarget{Model: "embed:1", Existed: true}
	state, err := create(in)
	require.NoError(t, err)
	assert.Equal(t, "embed:1", state.Model, "the deployed embedding model is kept")

	in.Seed = &compose.ModelTarget{Existed: true}
	_, err = create(in)
	require.ErrorIs(t, err, common.ErrEmbeddingMigration, "an unknown deployed target stops the migration")

	in.Seed = nil
	state, err = create(in)
	require.NoError(t, err)
	assert.Equal(t, "embed:2", state.Model, "a new embedding service resolves normally")

	chat := inputs("chat-default", "chat", compose.ModelCandidate{Model: "embed"})
	chat.Seed = &compose.ModelTarget{Model: "embed:1", Existed: true}
	state, err = create(chat)
	require.NoError(t, err)
	assert.Equal(t, "embed:2", state.Model, "chat follows the recipe")

	// An explicit exact request is honored even on the first upgrade.
	pinned := inputs("embed-pinned", "embedding", compose.ModelCandidate{Model: "embed", Version: "9"})
	pinned.Seed = &compose.ModelTarget{Model: "embed:1", Existed: true}
	state, err = create(pinned)
	require.NoError(t, err)
	assert.Equal(t, "9", state.Version)

	// The role can come from the catalog.
	r.roles = map[string]string{"embed:2": common.ModelRoleEmbedding}
	floating := inputs("my-embed", "", compose.ModelCandidate{Model: "embed"})
	floating.Seed = &compose.ModelTarget{Model: "embed:1", Existed: true}
	state, err = create(floating)
	require.NoError(t, err)
	assert.Equal(t, "embed:1", state.Model)
}

func TestDeferredUntilScopeKnown(t *testing.T) {
	r := &fakeResolver{}
	ms := ModelSelector{Cloud: exactCloud, Resolver: r, ScopeRequired: true}
	resp, err := ms.Create(context.Background(), createRequest{
		Inputs: inputs("chat-default", "chat", compose.ModelCandidate{Model: "a"}), DryRun: true,
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Output.Model)
	assert.Zero(t, r.calls)
}
