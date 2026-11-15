package program

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
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
	// under reports whether r is the named gateway's component or one of its
	// descendants. Only a service or LLM component owns a gateway: a project
	// named like the model does not.
	under := func(r *stateResource, service string) bool {
		for cur := r; cur != nil; cur = byURN[cur.Parent] {
			if urnName(cur.URN) == service && isGatewayComponent(cur.Type) {
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
			seedFromGatewayCommand(seed, r.Inputs)
		}
		// An Azure Foundry deployment is parented to the shared AI account, not
		// to its LLM component; it is named after the model applications send.
		if seed.Model == "" {
			if r := foundryDeployment(state.Resources, foundryDeploymentName(project, name)); r != nil {
				seed.Existed = true
				seedFromFoundryDeployment(seed, r)
			}
		}
		if seed.Existed {
			seeds[name] = seed
		}
	}
	return seeds, nil
}

func isGatewayComponent(typ string) bool {
	return strings.HasPrefix(typ, "defang-") &&
		(strings.HasSuffix(typ, ":index:Service") || strings.HasSuffix(typ, ":index:LLM"))
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

const foundryDeploymentType = "azure-native:cognitiveservices:Deployment"

// foundryDeploymentName is the deployment an LLM service uses on Azure: the
// alias of an older CLI's gateway, otherwise the normalized model request.
func foundryDeploymentName(project *compose.Project, name string) string {
	if svc, ok := project.Services[name]; ok {
		for i, arg := range svc.Command {
			if arg == "--alias" && i+1 < len(svc.Command) {
				return svc.Command[i+1]
			}
		}
	}
	return common.NormalizeModelRequest(project.Models[name].Model)
}

func foundryDeployment(resources []stateResource, deploymentName string) *stateResource {
	for i := range resources {
		if resources[i].Type == foundryDeploymentType && urnName(resources[i].URN) == deploymentName {
			return &resources[i]
		}
	}
	return nil
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
// container command (a JSON string), Cloud Run container args, or the docker
// command line of a Compute Engine instance template's cloud-init. Environment
// values are never read as commands. GCP's VERTEXAI_LOCATION comes from the
// same command line or from a Cloud Run env entry.
func seedFromGatewayCommand(seed *compose.ModelTarget, inputs map[string]any) {
	var walk func(v any)
	walk = func(v any) {
		if seed.Model != "" && seed.Location != "" {
			return
		}
		switch t := v.(type) {
		case map[string]any:
			for key, child := range t {
				if key == "environment" || key == "env" || key == "envs" {
					walkEnv(seed, child)
					continue
				}
				walk(child)
			}
		case []any:
			for i, child := range t {
				if s, _ := child.(string); s == "--model" && i+1 < len(t) {
					if target, ok := t[i+1].(string); ok && seed.Model == "" {
						seed.Model = gatewayTarget(target)
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
			if !strings.Contains(t, "docker run") {
				return // only a cloud-init docker command line is parsed as a command
			}
			fields := strings.Fields(t)
			for i, f := range fields {
				if f == "--model" && i+1 < len(fields) && seed.Model == "" {
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

// walkEnv reads only VERTEXAI_LOCATION from an environment list of
// {name, value} entries.
func walkEnv(seed *compose.ModelTarget, env any) {
	list, _ := env.([]any)
	for _, e := range list {
		entry, _ := e.(map[string]any)
		if name, _ := entry["name"].(string); name == "VERTEXAI_LOCATION" && seed.Location == "" {
			seed.Location, _ = entry["value"].(string)
		}
	}
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
