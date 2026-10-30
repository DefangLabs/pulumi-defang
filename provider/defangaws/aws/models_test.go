package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errAccessDenied = errors.New("AccessDeniedException: not authorized to perform bedrock:ListFoundationModels")

type fakeBedrock struct {
	models       []bedrocktypes.FoundationModelSummary
	profilePages [][]bedrocktypes.InferenceProfileSummary
	err          error
	modelCalls   int
	profileCalls int
}

func (f *fakeBedrock) ListFoundationModels(
	context.Context, *bedrock.ListFoundationModelsInput, ...func(*bedrock.Options),
) (*bedrock.ListFoundationModelsOutput, error) {
	f.modelCalls++
	if f.err != nil {
		return nil, f.err
	}
	return &bedrock.ListFoundationModelsOutput{ModelSummaries: f.models}, nil
}

func (f *fakeBedrock) ListInferenceProfiles(
	_ context.Context, in *bedrock.ListInferenceProfilesInput, _ ...func(*bedrock.Options),
) (*bedrock.ListInferenceProfilesOutput, error) {
	f.profileCalls++
	page := 0
	if in.NextToken != nil {
		page = int((*in.NextToken)[0] - '0')
	}
	out := &bedrock.ListInferenceProfilesOutput{InferenceProfileSummaries: f.profilePages[page]}
	if page+1 < len(f.profilePages) {
		out.NextToken = aws.String(string(rune('0' + page + 1)))
	}
	return out, nil
}

func foundationModel(
	id string, modality bedrocktypes.ModelModality, status bedrocktypes.FoundationModelLifecycleStatus,
) bedrocktypes.FoundationModelSummary {
	return bedrocktypes.FoundationModelSummary{
		ModelId:          aws.String(id),
		OutputModalities: []bedrocktypes.ModelModality{modality},
		ModelLifecycle:   &bedrocktypes.FoundationModelLifecycle{Status: status},
	}
}

func profile(id, modelID string) bedrocktypes.InferenceProfileSummary {
	return bedrocktypes.InferenceProfileSummary{
		InferenceProfileId: aws.String(id),
		Status:             bedrocktypes.InferenceProfileStatusActive,
		Models: []bedrocktypes.InferenceProfileModel{{
			ModelArn: aws.String("arn:aws:bedrock:us-east-1::foundation-model/" + modelID),
		}},
	}
}

func newFakeCatalog() *fakeBedrock {
	active := bedrocktypes.FoundationModelLifecycleStatusActive
	legacy := bedrocktypes.FoundationModelLifecycleStatusLegacy
	return &fakeBedrock{
		models: []bedrocktypes.FoundationModelSummary{
			foundationModel("amazon.nova-2-lite-v1:0", bedrocktypes.ModelModalityText, active),
			foundationModel("amazon.titan-embed-text-v2:0", bedrocktypes.ModelModalityEmbedding, active),
			foundationModel("amazon.titan-embed-text-v2:0:8k", bedrocktypes.ModelModalityEmbedding, active),
			foundationModel("vendor.family-v1:1", bedrocktypes.ModelModalityText, active),
			foundationModel("vendor.family-v1:2", bedrocktypes.ModelModalityText, active),
			foundationModel("vendor.family-v1:10", bedrocktypes.ModelModalityText, active),
			foundationModel("vendor.family-v1:11", bedrocktypes.ModelModalityText, legacy),
			foundationModel("vendor.family-v2:0", bedrocktypes.ModelModalityText, active),
			foundationModel("zai.glm-5", bedrocktypes.ModelModalityText, active),
		},
		// Two pages: discovery must follow the paginator.
		profilePages: [][]bedrocktypes.InferenceProfileSummary{
			{profile("us.amazon.nova-2-lite-v1:0", "amazon.nova-2-lite-v1:0")},
			{profile("us.vendor.family-v1:2", "vendor.family-v1:2")},
		},
	}
}

func resolver(t *testing.T, fake *fakeBedrock) *BedrockResolver {
	t.Helper()
	return &BedrockResolver{NewClient: func(_ context.Context, region string) (BedrockCatalogAPI, error) {
		assert.Equal(t, "us-east-1", region)
		return fake, nil
	}}
}

func resolve(
	t *testing.T, r *BedrockResolver, request string, recipe common.ModelRecipe,
) (common.ResolvedModel, error) {
	t.Helper()
	plan, err := common.PlanModelRequest(ModelCloud, request, recipe)
	require.NoError(t, err)
	in := common.ModelSelectionInputs{
		Request: plan.Request, Role: plan.Role, Source: plan.Source, Candidates: plan.Candidates, Region: "us-east-1",
	}
	var last error
	for _, c := range plan.Candidates {
		res, err := r.ResolveCandidate(context.Background(), in, c)
		if errors.Is(err, common.ErrModelUnavailable) {
			last = err
			continue
		}
		return res, err
	}
	return common.ResolvedModel{}, last
}

func TestBedrockExactNeedsNoDiscovery(t *testing.T) {
	fake := newFakeCatalog()
	r := resolver(t, fake)
	for _, request := range []string{"chat-default", "embedding-default", "us.amazon.nova-2-lite-v1:0",
		"arn:aws:bedrock:us-east-1:123456789012:application-inference-profile/abc"} {
		res, err := resolve(t, r, request, nil)
		require.NoError(t, err, request)
		assert.NotEmpty(t, res.Target.Model)
	}
	assert.Zero(t, fake.modelCalls, "exact IDs and ARNs need no catalog")
}

func TestBedrockFamilySelection(t *testing.T) {
	fake := newFakeCatalog()
	r := resolver(t, fake)
	recipe := testRecipe(t, `{
		"family": [{"model": "vendor.family-v1"}],
		"profile-family": [{"model": "us.vendor.family-v1", "role": "chat"}],
		"fallback": [{"model": "vendor.retired-v1:0"}, {"model": "vendor.family-v2:0"}]
	}`)

	res, err := resolve(t, r, "family", recipe)
	require.NoError(t, err)
	// :10 beats :2 numerically; :11 is LEGACY; v2 is a different major version.
	assert.Equal(t, "vendor.family-v1:10", res.Target.Model)
	assert.Equal(t, common.ModelRoleChat, res.Role, "role from output modalities")

	res, err = resolve(t, r, "profile-family", recipe)
	require.NoError(t, err)
	assert.Equal(t, "us.vendor.family-v1:2", res.Target.Model, "profile prefix kept, found on page 2")

	res, err = resolve(t, r, "fallback", recipe)
	require.NoError(t, err)
	assert.Equal(t, "vendor.family-v2:0", res.Target.Model, "documented unavailability falls through")

	// A colon-less exact ID present in the catalog is used verbatim.
	res, err = resolve(t, r, "chat-large", nil)
	require.NoError(t, err)
	assert.Equal(t, "zai.glm-5", res.Target.Model)

	// The ":0:8k" variant is not a revision of the family.
	res, err = resolve(t, r, "amazon.titan-embed-text-v2", nil)
	require.NoError(t, err)
	assert.Equal(t, "amazon.titan-embed-text-v2:0", res.Target.Model)
	assert.Equal(t, common.ModelRoleEmbedding, res.Role)

	_, err = resolve(t, r, "vendor.unknown-v1", nil)
	require.ErrorIs(t, err, common.ErrModelUnavailable)
	assert.Contains(t, err.Error(), "vendor.unknown-v1:0")

	assert.Equal(t, 1, fake.modelCalls, "the catalog is listed once per region")
}

func TestBedrockAccessDeniedFails(t *testing.T) {
	fake := newFakeCatalog()
	fake.err = errAccessDenied
	r := resolver(t, fake)
	_, err := resolve(t, r, "amazon.titan-embed-text-v2", nil)
	require.Error(t, err)
	require.NotErrorIs(t, err, common.ErrModelUnavailable, "permission errors never fall through")
	require.ErrorIs(t, err, errAccessDenied)
	assert.Contains(t, err.Error(), "bedrock:ListInferenceProfiles")
}

func TestIsExactBedrockID(t *testing.T) {
	assert.True(t, IsExactBedrockID("amazon.titan-embed-text-v2:0"))
	assert.True(t, IsExactBedrockID("amazon.titan-embed-text-v2:0:8k"))
	assert.True(t, IsExactBedrockID("us.amazon.nova-2-lite-v1:0"))
	assert.True(t, IsExactBedrockID("arn:aws:bedrock:us-east-1:1:inference-profile/x"))
	assert.False(t, IsExactBedrockID("amazon.nova-2-lite-v1"))
	assert.False(t, IsExactBedrockID("zai.glm-5"))
}

// testRecipe parses a defang-aws:models test recipe.
func testRecipe(t *testing.T, raw string) common.ModelRecipe {
	t.Helper()
	recipe, err := common.ParseModelRecipe(ModelCloud, raw)
	require.NoError(t, err)
	return recipe
}
