package common

import (
	"context"
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ModelSelectionType is the resource type suffix each provider registers as
// "<package>:index:ModelSelection".
const ModelSelectionType = "ModelSelection"

// ModelSelectionInputs are the inputs of a ModelSelection resource: the
// request and only the recipe entry relevant to it, so unrelated recipe
// changes never re-resolve a model.
type ModelSelectionInputs struct {
	// Request is the normalized model request (application-facing name on gateway clouds).
	Request string `pulumi:"request"`
	// Role is the request's role when known before resolution.
	Role string `pulumi:"role,optional"`
	// Source is where the candidates came from: recipe, default, native or legacy-payload.
	Source string `pulumi:"source"`
	// Candidates are the ordered candidates to resolve.
	Candidates []compose.ModelCandidate `pulumi:"candidates"`
	// Region is the cloud region (AWS region; GCP default Vertex location).
	Region string `pulumi:"region,optional"`
	// Scope identifies the catalog to resolve against (Azure: the AI Foundry account ID).
	Scope string `pulumi:"scope,optional"`
	// Seed is the target deployed before this selection existed; first creation only.
	Seed *compose.ModelTarget `pulumi:"seed,optional"`
}

// ModelSelectionState is the persisted selection.
type ModelSelectionState struct {
	ModelSelectionInputs

	// Model is the selected provider-native model ID.
	Model string `pulumi:"model"`
	// Version is the selected Azure model revision.
	Version string `pulumi:"version"`
	// Location is the selected GCP Vertex AI location.
	Location string `pulumi:"location"`
	// Format is the selected Azure model format.
	Format string `pulumi:"format"`
	// Sku is the selected Azure deployment SKU.
	Sku string `pulumi:"sku"`
	// SelectedRole is the role of the selected model, when known.
	SelectedRole string `pulumi:"selectedRole"`
}

// Target returns the selected model.
func (s ModelSelectionState) Target() compose.ModelTarget {
	return compose.ModelTarget{Model: s.Model, Version: s.Version, Location: s.Location, Format: s.Format, Sku: s.Sku}
}

// SetTarget records the selected model.
func (s *ModelSelectionState) SetTarget(t compose.ModelTarget) {
	s.Model, s.Version, s.Location, s.Format, s.Sku = t.Model, t.Version, t.Location, t.Format, t.Sku
}

// ResolvedModel is one resolved candidate.
type ResolvedModel struct {
	Target compose.ModelTarget
	// Role is the role established from catalog metadata, or "".
	Role string
}

// ModelResolver resolves candidates against one cloud's catalog.
type ModelResolver interface {
	// ResolveCandidate returns the concrete target for c. An error wrapping
	// ErrModelUnavailable means the provider established c cannot be used,
	// which lets a floating selection fall through to the next candidate;
	// any other error (access denied, network, quota) fails the selection.
	ResolveCandidate(ctx context.Context, in ModelSelectionInputs, c compose.ModelCandidate) (ResolvedModel, error)
}

// ModelSelectionResult is the registered resource's outputs.
type ModelSelectionResult struct {
	pulumi.CustomResourceState

	Model        pulumi.StringOutput `pulumi:"model"`
	Version      pulumi.StringOutput `pulumi:"version"`
	Location     pulumi.StringOutput `pulumi:"location"`
	Format       pulumi.StringOutput `pulumi:"format"`
	Sku          pulumi.StringOutput `pulumi:"sku"`
	SelectedRole pulumi.StringOutput `pulumi:"selectedRole"`
}

// RegisterModelSelection registers "<pkg>:index:ModelSelection" for a planned
// request. scope may be an Output (Azure account ID).
func RegisterModelSelection(
	ctx *pulumi.Context,
	pkg, name string,
	plan ModelPlan,
	region string,
	scope pulumi.StringInput,
	seed *compose.ModelTarget,
	pluginID PluginIdentity,
	opts ...pulumi.ResourceOption,
) (*ModelSelectionResult, error) {
	candidates := make(pulumi.Array, 0, len(plan.Candidates))
	for _, c := range plan.Candidates {
		m := pulumi.Map{"model": pulumi.String(c.Model)}
		setIf := func(k, v string) {
			if v != "" {
				m[k] = pulumi.String(v)
			}
		}
		setIf("version", c.Version)
		setIf("role", c.Role)
		setIf("location", c.Location)
		setIf("format", c.Format)
		setIf("sku", c.Sku)
		if c.AllowPreview {
			m["allowPreview"] = pulumi.Bool(true)
		}
		candidates = append(candidates, m)
	}
	props := pulumi.Map{
		"request":    pulumi.String(plan.Request),
		"source":     pulumi.String(plan.Source),
		"candidates": candidates,
	}
	if plan.Role != "" {
		props["role"] = pulumi.String(plan.Role)
	}
	if region != "" {
		props["region"] = pulumi.String(region)
	}
	if scope != nil {
		props["scope"] = scope
	}
	if seed != nil {
		props["seed"] = pulumi.Map{
			"model":    pulumi.String(seed.Model),
			"version":  pulumi.String(seed.Version),
			"location": pulumi.String(seed.Location),
			"format":   pulumi.String(seed.Format),
			"sku":      pulumi.String(seed.Sku),
			"existed":  pulumi.Bool(seed.Existed),
		}
	}
	var res ModelSelectionResult
	if err := ctx.RegisterResource(pkg+":index:"+ModelSelectionType, name, props, &res,
		pluginID.ResourceOptions(opts...)...); err != nil {
		return nil, fmt.Errorf("registering model selection for %s: %w", name, err)
	}
	return &res, nil
}
