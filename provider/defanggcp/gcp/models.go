package gcp

import (
	"context"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ModelCloud is GCP's recipe contract: defang-gcp:models candidates take
// `model`, `role` and `location`. Vertex AI model IDs are used exactly as
// Google documents them (numeric suffixes such as -001 are part of the ID),
// so there is no catalog lookup and every candidate is exact.
var ModelCloud = common.ModelCloud{
	Name:      "gcp",
	Namespace: "defang-gcp",
	Fields:    []string{common.ModelFieldLocation},
	Defaults: map[string][]compose.ModelCandidate{
		"chat-default":      {{Model: "gemini-3.5-flash", Location: "global"}},
		"chat-large":        {{Model: "gemini-3.1-pro-preview", Location: "global"}},
		"embedding-default": {{Model: "gemini-embedding-001"}},
	},
}

// VertexResolver resolves candidates without discovery: the first candidate,
// in its own location or the deployment region.
type VertexResolver struct{}

// ResolveCandidate implements common.ModelResolver.
func (VertexResolver) ResolveCandidate(
	_ context.Context, in common.ModelSelectionInputs, c compose.ModelCandidate,
) (common.ResolvedModel, error) {
	location := c.Location
	if location == "" {
		location = in.Region
	}
	return common.ResolvedModel{Target: compose.ModelTarget{Model: c.Model, Location: location}}, nil
}

// PrepareModels configures every managed model gateway in services.
func PrepareModels(
	ctx *pulumi.Context, services compose.Services, models compose.Models, gcpProject, region string,
	pluginID common.PluginIdentity, opts ...pulumi.ResourceOption,
) (compose.Services, error) {
	return common.GatewayModels{
		Cloud:             ModelCloud,
		Package:           "defang-gcp",
		LiteLLMProvider:   "vertex_ai",
		Region:            region,
		PluginID:          pluginID,
		LegacyLocationEnv: "VERTEXAI_LOCATION",
		Env: func(env compose.Environment, sel *common.ModelSelectionResult) {
			common.SetEnvDefault(env, "VERTEXAI_PROJECT", pulumi.String(gcpProject))
			env["VERTEXAI_LOCATION"] = sel.Location // always the selected location
		},
	}.Prepare(ctx, services, models, opts...)
}
