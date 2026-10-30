// Package modelselect implements the ModelSelection custom resource shared by
// every provider: it resolves a planned model request once and keeps the
// result until the request or its recipe entry changes.
package modelselect

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// ModelSelector implements the ModelSelection resource lifecycle for one cloud.
type ModelSelector struct {
	Cloud    common.ModelCloud
	Resolver common.ModelResolver
	// ScopeRequired means resolution needs Scope (Azure's account), which is
	// unknown during a preview that also creates the account.
	ScopeRequired bool
}

// deferred reports whether resolution must wait for an unknown scope.
func (ms ModelSelector) deferred(dryRun bool, in common.ModelSelectionInputs) bool {
	return dryRun && ms.ScopeRequired && in.Scope == ""
}

// Resolve tries the candidates in order.
func (ms ModelSelector) Resolve(
	ctx context.Context, in common.ModelSelectionInputs,
) (common.ModelSelectionState, error) {
	state := common.ModelSelectionState{ModelSelectionInputs: in}
	var unavailable []string
	for i, c := range in.Candidates {
		res, err := ms.Resolver.ResolveCandidate(ctx, in, c)
		if errors.Is(err, common.ErrModelUnavailable) && c.Version == "" {
			unavailable = append(unavailable, err.Error())
			continue
		}
		if err != nil {
			return state, fmt.Errorf("model %q (%s candidate %d, %s): %w", in.Request, in.Source, i, c.Model, err)
		}
		role := in.Role
		if role == "" {
			role = res.Role
		} else if res.Role != "" && res.Role != role {
			return state, fmt.Errorf("%w: model %q is a %s model, but %q needs a %s model", common.ErrModelRequest,
				res.Target.Model, res.Role, in.Request, role)
		}
		state.SetTarget(res.Target)
		state.SelectedRole = role
		return state, nil
	}
	return state, fmt.Errorf("%w: no candidate for %q can be deployed on %s: %s; add an available model to %s:%s.%s",
		common.ErrModelUnavailable, in.Request, ms.Cloud.Name, strings.Join(unavailable, "; "),
		ms.Cloud.Namespace, "models", in.Request)
}

// Create resolves a new selection. On the first upgrade of an existing
// deployment, an embedding model keeps the target it was deployed with.
func (ms ModelSelector) Create(
	ctx context.Context, req infer.CreateRequest[common.ModelSelectionInputs],
) (infer.CreateResponse[common.ModelSelectionState], error) {
	in := req.Inputs
	if ms.deferred(req.DryRun, in) {
		return infer.CreateResponse[common.ModelSelectionState]{ID: in.Request,
			Output: common.ModelSelectionState{ModelSelectionInputs: in}}, nil
	}
	state, err := ms.Resolve(ctx, in)
	if err != nil {
		return infer.CreateResponse[common.ModelSelectionState]{}, err
	}
	if seed := in.Seed; seed != nil && seed.Existed && state.SelectedRole == common.ModelRoleEmbedding &&
		!ms.Cloud.IsExact(in.Candidates) {
		if seed.Model == "" {
			return infer.CreateResponse[common.ModelSelectionState]{}, fmt.Errorf("%w: embedding model %q is already "+
				"deployed but its current target cannot be established; pin it with a native exact model ID or a "+
				"recipe key with one exact candidate, then redeploy", common.ErrEmbeddingMigration, in.Request)
		}
		if !sameModel(seed, &state) {
			p.GetLogger(ctx).Warningf("embedding model %q keeps its deployed target %s; the recipe now maps it to %s. "+
				"To migrate, request the new model explicitly and regenerate stored vectors.",
				in.Request, describeTarget(*seed), describeTarget(state.Target()))
			state.SetTarget(*seed)
		}
	}
	return infer.CreateResponse[common.ModelSelectionState]{ID: in.Request, Output: state}, nil
}

// Update re-resolves only when the request or its recipe entry changed, and
// never silently replaces an embedding model.
func (ms ModelSelector) Update(
	ctx context.Context, req infer.UpdateRequest[common.ModelSelectionInputs, common.ModelSelectionState],
) (infer.UpdateResponse[common.ModelSelectionState], error) {
	old, in := req.State, req.Inputs
	if sameSelection(old.ModelSelectionInputs, in) || ms.deferred(req.DryRun, in) {
		kept := old
		kept.ModelSelectionInputs = in
		return infer.UpdateResponse[common.ModelSelectionState]{Output: kept}, nil
	}
	role := old.SelectedRole
	if role == "" {
		role = in.Role
	}
	if old.Model == "" || role != common.ModelRoleEmbedding {
		state, err := ms.Resolve(ctx, in)
		return infer.UpdateResponse[common.ModelSelectionState]{Output: state}, err
	}

	if in.Request == old.Request {
		// The recipe entry changed under an existing embedding model: keep it.
		kept := old
		kept.ModelSelectionInputs = in
		if state, err := ms.Resolve(ctx, in); err != nil {
			p.GetLogger(ctx).Warningf("embedding model %q keeps %s; its recipe entry no longer resolves: %v",
				in.Request, describeTarget(old.Target()), err)
		} else if !sameTarget(old.Target(), state.Target()) {
			p.GetLogger(ctx).Warningf("embedding model %q keeps %s; the recipe now maps it to %s. To migrate, "+
				"request the new model explicitly (a native exact ID or a recipe key with one exact candidate) and "+
				"regenerate stored vectors.", in.Request, describeTarget(old.Target()), describeTarget(state.Target()))
		}
		return infer.UpdateResponse[common.ModelSelectionState]{Output: kept}, nil
	}
	if !ms.Cloud.IsExact(in.Candidates) {
		return infer.UpdateResponse[common.ModelSelectionState]{}, fmt.Errorf("%w: embedding model was %q (%s); "+
			"switching to %q needs an exact target: use a native exact model ID or a recipe key with one exact "+
			"candidate, and regenerate stored vectors", common.ErrEmbeddingMigration, old.Request,
			describeTarget(old.Target()), in.Request)
	}
	state, err := ms.Resolve(ctx, in)
	if err == nil && !sameTarget(old.Target(), state.Target()) {
		p.GetLogger(ctx).Warningf("embedding model changes from %s to %s; regenerate vectors stored with the old model",
			describeTarget(old.Target()), describeTarget(state.Target()))
	}
	return infer.UpdateResponse[common.ModelSelectionState]{Output: state}, err
}

// Diff reports input changes as in-place updates. The seed only matters on
// creation and never causes a diff.
func (ModelSelector) Diff(
	_ context.Context, req infer.DiffRequest[common.ModelSelectionInputs, common.ModelSelectionState],
) (infer.DiffResponse, error) {
	old, in := req.State.ModelSelectionInputs, req.Inputs
	diff := map[string]p.PropertyDiff{}
	check := func(name string, a, b any) {
		if !reflect.DeepEqual(a, b) {
			diff[name] = p.PropertyDiff{Kind: p.Update, InputDiff: true}
		}
	}
	check("request", old.Request, in.Request)
	check("role", old.Role, in.Role)
	check("source", old.Source, in.Source)
	check("candidates", normalizeCandidates(old.Candidates), normalizeCandidates(in.Candidates))
	check("region", old.Region, in.Region)
	check("scope", old.Scope, in.Scope)
	return infer.DiffResponse{HasChanges: len(diff) > 0, DetailedDiff: diff}, nil
}

func sameSelection(a, b common.ModelSelectionInputs) bool {
	return a.Request == b.Request && a.Role == b.Role && a.Region == b.Region && a.Scope == b.Scope &&
		reflect.DeepEqual(normalizeCandidates(a.Candidates), normalizeCandidates(b.Candidates))
}

func normalizeCandidates(c []compose.ModelCandidate) []compose.ModelCandidate {
	if len(c) == 0 {
		return nil
	}
	return c
}

func sameTarget(a, b compose.ModelTarget) bool {
	return a.Model == b.Model && a.Version == b.Version
}

func sameModel(seed *compose.ModelTarget, s *common.ModelSelectionState) bool {
	return sameTarget(*seed, s.Target())
}

func describeTarget(t compose.ModelTarget) string {
	s := t.Model
	if t.Version != "" {
		s += " version " + t.Version
	}
	if t.Location != "" {
		s += " in " + t.Location
	}
	return s
}
