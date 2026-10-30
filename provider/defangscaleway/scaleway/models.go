package scaleway

import (
	"context"
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ModelCloud is Scaleway's recipe contract: defang-scaleway:models candidates
// take `model` and `role`. Generative APIs model IDs are used exactly as
// given (release dates are never stripped), so every candidate is exact.
var ModelCloud = common.ModelCloud{
	Name:      "scaleway",
	Namespace: "defang-scaleway",
	Defaults: map[string][]compose.ModelCandidate{
		"chat-default":      {{Model: "qwen3-235b-a22b-instruct-2507"}},
		"chat-large":        {{Model: "qwen3-235b-a22b-instruct-2507"}},
		"embedding-default": {{Model: "bge-multilingual-gemma2"}},
	},
}

// GenerativeAPIsResolver resolves candidates without discovery: the first
// candidate's model ID.
type GenerativeAPIsResolver struct{}

// ResolveCandidate implements common.ModelResolver.
func (GenerativeAPIsResolver) ResolveCandidate(
	_ context.Context, _ common.ModelSelectionInputs, c compose.ModelCandidate,
) (common.ResolvedModel, error) {
	return common.ResolvedModel{Target: compose.ModelTarget{Model: c.Model}}, nil
}

// PrepareModels resolves the Compose models of a direct-inference project:
// each model's request is recorded in a ModelSelection and its model ID is
// written into the model variable of every bound service that does not set
// it already. There is no gateway; the CLI wrote the direct endpoint and the
// API key reference. A depends_on edge to a model (not a service) is dropped.
// An LLM service that brings its own container is left to the container
// validation, which rejects it on Scaleway.
func PrepareModels(
	ctx *pulumi.Context, services compose.Services, models compose.Models, region string,
	pluginID common.PluginIdentity, opts ...pulumi.ResourceOption,
) (compose.Services, error) {
	if len(models) == 0 {
		return services, nil
	}
	recipe, err := common.ReadModelRecipe(ctx, ModelCloud)
	if err != nil {
		return nil, err
	}
	out := make(compose.Services, len(services))
	for name, svc := range services {
		svc.DependsOn = withoutModelDeps(svc.DependsOn, services, models)
		out[name] = svc
	}
	for name, model := range common.Sorted(models) {
		plan, err := common.PlanModelRequest(ModelCloud, model.Model, recipe)
		if err != nil {
			return nil, fmt.Errorf("model %q: %w", name, err)
		}
		sel, err := common.RegisterModelSelection(ctx, "defang-scaleway", name, plan, region, nil, model.Seed,
			pluginID, opts...)
		if err != nil {
			return nil, err
		}
		// Direct inference: applications send the provider's own model ID.
		if err := common.InjectModelVar(out, name, sel.Model); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// withoutModelDeps drops depends_on edges to models that are not services.
func withoutModelDeps(
	deps compose.DependsOnConfig, services compose.Services, models compose.Models,
) compose.DependsOnConfig {
	var kept compose.DependsOnConfig
	for dep, cond := range deps {
		_, isModel := models[dep]
		_, isService := services[dep]
		if isModel && !isService {
			continue
		}
		if kept == nil {
			kept = make(compose.DependsOnConfig, len(deps))
		}
		kept[dep] = cond
	}
	return kept
}
