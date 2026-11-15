package program

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
)

// ModelSeeds maps a Compose model name to the target it was deployed with
// before its selection was persisted in a ModelSelection resource.
type ModelSeeds map[string]*compose.ModelTarget

// stateResource is the subset of a Pulumi checkpoint resource that seeding reads.
type stateResource struct {
	URN     string         `json:"urn"`
	Type    string         `json:"type"`
	Parent  string         `json:"parent"`
	Inputs  map[string]any `json:"inputs"`
	Outputs map[string]any `json:"outputs"`
}

type stateDeployment struct {
	Resources []stateResource `json:"resources"`
}

const modelSelectionTypeSuffix = ":index:ModelSelection"

// SeedsFromState establishes, for every Compose model in composeYAML, the
// model it is currently deployed with according to the stack's checkpoint (the
// deployment object of `pulumi stack export`). Scaleway reads no seeds:
// recipe-based selection has not shipped there, so there is nothing to migrate.
// For the other clouds, a persisted selection needs no seed; the first upgrade
// of an existing deployment uses one so an embedding model is never replaced
// silently. Existed without Model means the service exists but its target could
// not be established, which stops an embedding migration.
func SeedsFromState(deployment []byte, composeYAML []byte, projectName string) (ModelSeeds, error) {
	if len(deployment) == 0 {
		return ModelSeeds{}, nil
	}
	var state stateDeployment
	if err := json.Unmarshal(deployment, &state); err != nil {
		return nil, fmt.Errorf("reading stack state: %w", err)
	}
	if len(state.Resources) == 0 {
		return ModelSeeds{}, nil
	}
	project, err := parseCompose(composeYAML, projectName)
	if err != nil {
		return nil, err
	}

	byURN := make(map[string]*stateResource, len(state.Resources))
	for i := range state.Resources {
		byURN[state.Resources[i].URN] = &state.Resources[i]
	}
	// under reports whether r is the named service's component or one of its descendants.
	under := func(r *stateResource, service string) bool {
		for cur := r; cur != nil; cur = byURN[cur.Parent] {
			if urnName(cur.URN) == service && strings.HasPrefix(cur.Type, "defang-") {
				return true
			}
		}
		return false
	}

	seeds := ModelSeeds{}
	for name := range project.Models {
		if selectionExists(state.Resources, name) {
			continue
		}
		seed := &compose.ModelTarget{}
		for i := range state.Resources {
			r := &state.Resources[i]
			if !under(r, name) {
				continue
			}
			seed.Existed = true
			if seed.Model != "" {
				continue
			}
			if r.Type == "azure-native:cognitiveservices:Deployment" {
				seedFromFoundryDeployment(seed, r)
			} else {
				seedFromGatewayCommand(seed, r.Inputs)
			}
		}
		if seed.Existed {
			seeds[name] = seed
		}
	}
	return seeds, nil
}

func urnName(urn string) string {
	if i := strings.LastIndex(urn, "::"); i >= 0 {
		return urn[i+2:]
	}
	return urn
}

func selectionExists(resources []stateResource, service string) bool {
	for _, r := range resources {
		if strings.HasSuffix(r.Type, modelSelectionTypeSuffix) && urnName(r.URN) == service {
			return true
		}
	}
	return false
}

// seedFromFoundryDeployment reads an Azure AI Foundry deployment's model.
func seedFromFoundryDeployment(seed *compose.ModelTarget, r *stateResource) {
	props := r.Outputs
	if props == nil {
		props = r.Inputs
	}
	properties, _ := props["properties"].(map[string]any)
	model, _ := properties["model"].(map[string]any)
	sku, _ := props["sku"].(map[string]any)
	seed.Model, _ = model["name"].(string)
	seed.Version, _ = model["version"].(string)
	seed.Format, _ = model["format"].(string)
	seed.Sku, _ = sku["name"].(string)
}

// seedFromGatewayCommand finds the LiteLLM `--model <target>` argument of a
// gateway, wherever the cloud resource keeps it: an ECS task definition's
// container definitions (a JSON string), Cloud Run container args, or the
// docker command line of a Compute Engine instance template's cloud-init.
func seedFromGatewayCommand(seed *compose.ModelTarget, inputs map[string]any) {
	var walk func(v any)
	walk = func(v any) {
		if seed.Model != "" {
			return
		}
		switch t := v.(type) {
		case map[string]any:
			for _, child := range t {
				walk(child)
			}
		case []any:
			for i, child := range t {
				if s, _ := child.(string); s == "--model" && i+1 < len(t) {
					if target, ok := t[i+1].(string); ok {
						seed.Model = gatewayTarget(target)
						return
					}
				}
				walk(child)
			}
		case string:
			trimmed := strings.TrimSpace(t)
			if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
				var nested any
				if json.Unmarshal([]byte(trimmed), &nested) == nil {
					walk(nested)
					return
				}
			}
			fields := strings.Fields(t)
			for i, f := range fields {
				if f == "--model" && i+1 < len(fields) {
					seed.Model = gatewayTarget(fields[i+1])
				}
				if loc, ok := strings.CutPrefix(f, "VERTEXAI_LOCATION="); ok && seed.Location == "" {
					seed.Location = strings.Trim(loc, `'"`)
				}
			}
		}
	}
	walk(inputs)
}

// gatewayTarget strips the LiteLLM provider prefix from a gateway target.
func gatewayTarget(target string) string {
	target = strings.Trim(target, `'"`)
	for _, prefix := range []string{"bedrock/", "vertex_ai/"} {
		if t, ok := strings.CutPrefix(target, prefix); ok {
			return t
		}
	}
	return target
}

// applySeeds attaches each seed to its Compose model.
func applySeeds(project *compose.Project, seeds ModelSeeds) {
	for name, seed := range seeds {
		model, ok := project.Models[name]
		if !ok {
			continue
		}
		model.Seed = seed
		project.Models[name] = model
	}
}
