package azure

import (
	"fmt"
	"sort"
	"strings"

	containerregistry "github.com/pulumi/pulumi-azure-native-sdk/containerregistry/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"gopkg.in/yaml.v3"
)

type acrTaskStep struct {
	Build string   `yaml:"build,omitempty"`
	Env   []string `yaml:"env,omitempty"`
	Push  []string `yaml:"push,omitempty"`
}

type acrTaskSpec struct {
	Version string        `yaml:"version"`
	Steps   []acrTaskStep `yaml:"steps"`
}

// generateTaskYAML creates the ACR task YAML for building and pushing a Docker image.
// Uses BuildKit layer cache stored in the registry (type=registry).
func generateTaskYAML(imageName, dockerfilePath string, buildArgs map[string]string, platform string) (string, error) {
	flags := make([]string, 0, 1+len(buildArgs)+5)

	if platform != "" {
		flags = append(flags, "--platform "+platform)
	}

	// Build args in stable order
	keys := make([]string, 0, len(buildArgs))
	for k := range buildArgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		flags = append(flags, fmt.Sprintf("--build-arg %s=%q", k, buildArgs[k]))
	}

	// BUILDKIT_INLINE_CACHE=1 embeds cache metadata into the image so it can
	// be used as --cache-from on the next run (inline cache strategy).
	flags = append(flags,
		"--build-arg BUILDKIT_INLINE_CACHE=1",
		fmt.Sprintf("-t {{.Run.Registry}}/%s:{{.Run.ID}}", imageName),
		fmt.Sprintf("-t {{.Run.Registry}}/%s:latest", imageName),
		"-f "+dockerfilePath,
		fmt.Sprintf("--cache-from {{.Run.Registry}}/%s:latest", imageName),
		".",
	)

	// Use build: shorthand (docker build) with DOCKER_BUILDKIT=1.
	// docker buildx is not available on ACR Tasks agents, but DOCKER_BUILDKIT=1
	// enables BuildKit in docker build, which supports inline cache.
	spec := acrTaskSpec{
		Version: "v1.1.0",
		Steps: []acrTaskStep{
			{
				Build: strings.Join(flags, " "),
				Env:   []string{"DOCKER_BUILDKIT=1"},
			},
			{
				Push: []string{
					fmt.Sprintf("{{.Run.Registry}}/%s:{{.Run.ID}}", imageName),
					fmt.Sprintf("{{.Run.Registry}}/%s:latest", imageName),
				},
			},
		},
	}

	out, err := yaml.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("marshaling ACR task YAML: %w", err)
	}
	return string(out), nil
}

// createACRTask creates an ACR task that builds and pushes a Docker image.
// encodedYAML is the base64-encoded ACR task YAML (from generateTaskYAML).
// The build context is NOT stored in the task definition because the ARM API
// strips SAS query strings from contextPath; it is passed at run time instead.
func createACRTask(
	ctx *pulumi.Context,
	name string,
	encodedYAML string,
	contextURL pulumi.StringInput,
	registry *containerregistry.Registry,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*containerregistry.Task, error) {
	// Log context URL for debugging (without exposing the token value).
	contextURL.ToStringOutput().ApplyT(func(s string) string {
		if idx := strings.Index(s, "?"); idx >= 0 {
			base := s[:idx]
			msg := fmt.Sprintf("ACR task %s: build context URL: %s (SAS token present, %d bytes)",
				name, base, len(s)-idx-1)
			_ = ctx.Log.Info(msg, nil)
		} else {
			_ = ctx.Log.Info(fmt.Sprintf("ACR task %s: build context URL: %s (no SAS token)", name, s), nil)
		}
		return s
	})

	// Derive service name from the task logical name (typically "{service}-build")
	// so build resources are tagged with the right defang-service value.
	serviceTag := strings.TrimSuffix(name, "-build")
	task, err := containerregistry.NewTask(ctx, name, &containerregistry.TaskArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		RegistryName:      registry.Name,
		Location:          registry.Location,
		Platform: &containerregistry.PlatformPropertiesArgs{
			Os: pulumi.String("Linux"),
		},
		Step: containerregistry.EncodedTaskStepArgs{
			Type:               pulumi.String("EncodedTask"),
			EncodedTaskContent: pulumi.String(encodedYAML),
			// ContextPath intentionally omitted: ARM strips SAS query strings when
			// persisting the task. The full SAS URL is passed at run time via
			// EncodedTaskRunRequest, which is never stored.
		},
		AgentConfiguration: &containerregistry.AgentPropertiesArgs{
			Cpu: pulumi.Int(2),
		},
		Timeout: pulumi.Int(3600),
		Tags:    DefangTags(ctx, infra.Etag, serviceTag),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ACR task: %w", err)
	}

	return task, nil
}
