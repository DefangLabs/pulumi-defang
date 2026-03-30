package azure

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	containerregistry "github.com/pulumi/pulumi-azure-native-sdk/containerregistry/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// generateTaskYAML creates the ACR task YAML for building and pushing a Docker image.
// Uses BuildKit layer cache stored in the registry (type=registry).
func generateTaskYAML(imageName, dockerfilePath string, buildArgs map[string]string, platform string) string {
	var flags []string

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

	buildLine := strings.Join(flags, "\n      ")

	// Use build: shorthand (docker build) with DOCKER_BUILDKIT=1.
	// docker buildx is not available on ACR Tasks agents, but DOCKER_BUILDKIT=1
	// enables BuildKit in docker build, which supports inline cache.
	return fmt.Sprintf(`version: v1.1.0
steps:
  - build: >-
      %s
    env:
      - DOCKER_BUILDKIT=1
  - push:
      - {{.Run.Registry}}/%s:{{.Run.ID}}
      - {{.Run.Registry}}/%s:latest
`, buildLine, imageName, imageName)
}

// createACRTask creates an ACR task that builds and pushes a Docker image.
// The build context is supplied via build.Context (an Azure Blob SAS URL or plain URL).
func createACRTask(
	ctx *pulumi.Context,
	name string,
	build compose.BuildConfig,
	platform string,
	imageName string,
	registry *containerregistry.Registry,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*containerregistry.Task, error) {
	dockerfile := build.GetDockerfile()

	taskYAML := generateTaskYAML(imageName, dockerfile, build.Args, platform)
	encodedYAML := base64.StdEncoding.EncodeToString([]byte(taskYAML))

	// Split blob SAS URL into base URL and SAS token (if present).
	// ACR Tasks require the token separately as ContextAccessToken.
	contextURL := build.Context.ToStringOutput().ApplyT(func(s string) *string {
		if idx := strings.Index(s, "?"); idx >= 0 {
			url := s[:idx]
			return &url
		}
		return &s
	}).(pulumi.StringPtrOutput)

	contextToken := build.Context.ToStringOutput().ApplyT(func(s string) *string {
		if idx := strings.Index(s, "?"); idx >= 0 {
			token := s[idx+1:]
			return &token
		}
		return nil
	}).(pulumi.StringPtrOutput)

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
			ContextPath:        contextURL,
			ContextAccessToken: contextToken,
		},
		AgentConfiguration: &containerregistry.AgentPropertiesArgs{
			Cpu: pulumi.Int(2),
		},
		Timeout: pulumi.Int(3600),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ACR task: %w", err)
	}

	return task, nil
}
