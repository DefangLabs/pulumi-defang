package program

import (
	"encoding/json"
	"fmt"
	"strconv"
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
		// An Azure selection is named after its Foundry deployment; the other
		// clouds name it after the model.
		if selectionExists(state.Resources, name, false) ||
			selectionExists(state.Resources, foundryDeploymentName(project, name), true) {
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

// selectionExists reports whether a ModelSelection named name exists, among
// Azure's selections or among the other clouds'.
func selectionExists(resources []stateResource, name string, azure bool) bool {
	for _, r := range resources {
		if strings.HasSuffix(r.Type, modelSelectionTypeSuffix) && urnName(r.URN) == name &&
			strings.HasPrefix(r.Type, "defang-azure:") == azure {
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
// gateway, wherever the cloud resource keeps it: an ECS container's `command`
// (inside a JSON string), Cloud Run container `args`, or the docker command
// line of a Compute Engine instance template's cloud-init. Environment values
// and health checks are never read as commands. GCP's VERTEXAI_LOCATION comes
// from the same command line or from a Cloud Run env entry.
func seedFromGatewayCommand(seed *compose.ModelTarget, inputs map[string]any) {
	var walk func(v any)
	walk = func(v any) {
		if seed.Model != "" && seed.Location != "" {
			return
		}
		switch t := v.(type) {
		case map[string]any:
			for key, child := range t {
				lower := strings.ToLower(key)
				switch {
				case lower == "environment" || lower == "env" || lower == "envs":
					walkEnv(seed, child)
				case strings.Contains(lower, "health") || strings.Contains(lower, "probe"):
					// a health check's command is not the gateway's
				case lower == "command" || lower == "args":
					readModelArg(seed, child)
				default:
					walk(child)
				}
			}
		case []any:
			for _, child := range t {
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
			_, line, ok := strings.Cut(t, "docker run ")
			if !ok {
				return // only a cloud-init docker command line is parsed as a command
			}
			line, _, _ = strings.Cut(line, "\n")
			args := shellFields(line)
			for i, arg := range args {
				if arg == "--model" && i+1 < len(args) && seed.Model == "" {
					seed.Model = gatewayTarget(args[i+1])
				}
				if arg == "-e" && i+1 < len(args) && seed.Location == "" {
					if loc, ok := strings.CutPrefix(args[i+1], "VERTEXAI_LOCATION="); ok {
						seed.Location = loc
					}
				}
			}
		}
	}
	walk(inputs)
}

// shellFields splits a command line into arguments, keeping quoted text,
// such as the `-e "KEY=value"` flags of a cloud-init docker command (Go %q
// quoting), as one argument.
func shellFields(line string) []string {
	var args []string
	var cur strings.Builder
	inArg := false
	for i := 0; i < len(line); i++ {
		switch c := line[i]; c {
		case '"':
			inArg = true
			if j := closingQuote(line, i); j > i {
				if unq, err := strconv.Unquote(line[i : j+1]); err == nil {
					cur.WriteString(unq)
				} else {
					cur.WriteString(line[i+1 : j])
				}
				i = j
			}
		case '\'':
			inArg = true
			if j := strings.IndexByte(line[i+1:], '\''); j >= 0 {
				cur.WriteString(line[i+1 : i+1+j])
				i += j + 1
			}
		case ' ', '\t':
			if inArg {
				args = append(args, cur.String())
				cur.Reset()
				inArg = false
			}
		default:
			inArg = true
			cur.WriteByte(c)
		}
	}
	if inArg {
		args = append(args, cur.String())
	}
	return args
}

// closingQuote returns the index of the double quote that closes the one at
// line[start], skipping backslash escapes, or -1.
func closingQuote(line string, start int) int {
	for i := start + 1; i < len(line); i++ {
		switch line[i] {
		case '\\':
			i++
		case '"':
			return i
		}
	}
	return -1
}

// readModelArg reads the target after `--model` in a container's command or
// args list.
func readModelArg(seed *compose.ModelTarget, command any) {
	args, _ := command.([]any)
	for i, arg := range args {
		if s, _ := arg.(string); s == "--model" && i+1 < len(args) && seed.Model == "" {
			seed.Model, _ = args[i+1].(string)
			seed.Model = gatewayTarget(seed.Model)
		}
	}
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
