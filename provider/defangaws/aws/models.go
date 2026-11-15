package aws

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// exactBedrockIDRE matches a Bedrock model or inference-profile ID with a
// concrete revision ("amazon.titan-embed-text-v2:0", optionally followed by a
// variant such as ":8k").
var exactBedrockIDRE = regexp.MustCompile(`:\d+(:[A-Za-z0-9._-]+)?$`)

// IsExactBedrockID reports whether id names one concrete Bedrock model: an
// ARN or an ID with a ":<revision>" suffix. A colon-less ID may still be
// exact (e.g. "zai.glm-5"), which only the catalog can tell.
func IsExactBedrockID(id string) bool {
	return strings.HasPrefix(id, "arn:") || exactBedrockIDRE.MatchString(id)
}

// ModelCloud is AWS's recipe contract: defang-aws:models candidates take only
// `model` and `role`; revisions are part of the Bedrock ID.
var ModelCloud = common.ModelCloud{
	Name:      "aws",
	Namespace: "defang-aws",
	Defaults: map[string][]compose.ModelCandidate{
		"chat-default":      {{Model: "us.amazon.nova-2-lite-v1:0"}},
		"chat-large":        {{Model: "zai.glm-5"}},
		"embedding-default": {{Model: "amazon.titan-embed-text-v2:0"}},
	},
	NativeExact:     IsExactBedrockID,
	CandidateExact:  func(c compose.ModelCandidate) bool { return IsExactBedrockID(c.Model) },
	RoleFromCatalog: true,
}

// BedrockCatalogAPI is the subset of the Bedrock control-plane API used for
// model discovery (bedrock:ListFoundationModels, bedrock:ListInferenceProfiles).
type BedrockCatalogAPI interface {
	ListFoundationModels(
		ctx context.Context, params *bedrock.ListFoundationModelsInput, optFns ...func(*bedrock.Options),
	) (*bedrock.ListFoundationModelsOutput, error)
	bedrock.ListInferenceProfilesAPIClient
}

type bedrockEntry struct {
	role   string
	active bool
}

type bedrockCatalog struct {
	entries map[string]bedrockEntry // foundation-model and inference-profile IDs
}

// BedrockResolver resolves defang-aws:models candidates against the Bedrock
// catalogs of the deployment region. Catalogs are listed at most once per
// region and only when a candidate needs them.
type BedrockResolver struct {
	NewClient func(ctx context.Context, region string) (BedrockCatalogAPI, error)

	mu       sync.Mutex
	catalogs map[string]*bedrockCatalog
}

// NewBedrockResolver returns a resolver using the default AWS credentials.
func NewBedrockResolver() *BedrockResolver {
	return &BedrockResolver{NewClient: func(ctx context.Context, region string) (BedrockCatalogAPI, error) {
		cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
		if err != nil {
			return nil, err
		}
		return bedrock.NewFromConfig(cfg), nil
	}}
}

// ResolveCandidate implements common.ModelResolver.
func (r *BedrockResolver) ResolveCandidate(
	ctx context.Context, in common.ModelSelectionInputs, c compose.ModelCandidate,
) (common.ResolvedModel, error) {
	id := c.Model
	exact := IsExactBedrockID(id)
	if strings.HasPrefix(id, "arn:") {
		// ARNs (application inference profiles, provisioned throughput) are
		// account resources: used as given.
		return common.ResolvedModel{Target: compose.ModelTarget{Model: id}}, nil
	}
	needsRole := in.Role == "" && in.Source == common.ModelSourceRecipe
	if exact && len(in.Candidates) == 1 && !needsRole {
		return common.ResolvedModel{Target: compose.ModelTarget{Model: id}}, nil
	}

	cat, err := r.catalog(ctx, in.Region)
	if err != nil {
		return common.ResolvedModel{}, err
	}
	if e, ok := cat.entries[id]; ok {
		if !exact && !e.active {
			return common.ResolvedModel{}, fmt.Errorf("%w: %s is not active in %s", common.ErrModelUnavailable, id, in.Region)
		}
		return common.ResolvedModel{Target: compose.ModelTarget{Model: id}, Role: e.role}, nil
	}
	if exact {
		return common.ResolvedModel{}, fmt.Errorf("%w: %s is not offered in %s", common.ErrModelUnavailable, id, in.Region)
	}

	// A family without its revision: the highest active "<id>:<n>". The
	// family, including its "-vN" major version, never changes.
	best, bestRev := "", -1
	for candidate, e := range cat.entries {
		revText, ok := strings.CutPrefix(candidate, id+":")
		if !ok || !e.active {
			continue
		}
		rev, err := strconv.Atoi(revText)
		if err != nil {
			continue // a variant such as ":0:8k" is not a revision of this family
		}
		if rev > bestRev {
			best, bestRev = candidate, rev
		}
	}
	if best == "" {
		return common.ResolvedModel{}, fmt.Errorf("%w: %s matches no active model or inference profile in %s; use an "+
			"exact ID such as %s:0, or a recipe key", common.ErrModelUnavailable, id, in.Region, id)
	}
	return common.ResolvedModel{Target: compose.ModelTarget{Model: best}, Role: cat.entries[best].role}, nil
}

func (r *BedrockResolver) catalog(ctx context.Context, region string) (*bedrockCatalog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if cat, ok := r.catalogs[region]; ok {
		return cat, nil
	}
	client, err := r.NewClient(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("creating Bedrock client: %w", err)
	}
	cat, err := listBedrockCatalog(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("listing Bedrock models in %s (the CD role needs bedrock:ListFoundationModels and "+
			"bedrock:ListInferenceProfiles): %w", region, err)
	}
	if r.catalogs == nil {
		r.catalogs = map[string]*bedrockCatalog{}
	}
	r.catalogs[region] = cat
	return cat, nil
}

func listBedrockCatalog(ctx context.Context, client BedrockCatalogAPI) (*bedrockCatalog, error) {
	models, err := client.ListFoundationModels(ctx, &bedrock.ListFoundationModelsInput{})
	if err != nil {
		return nil, err
	}
	cat := &bedrockCatalog{entries: map[string]bedrockEntry{}}
	for _, m := range models.ModelSummaries {
		if m.ModelId == nil {
			continue
		}
		active := m.ModelLifecycle == nil || m.ModelLifecycle.Status == bedrocktypes.FoundationModelLifecycleStatusActive
		cat.entries[*m.ModelId] = bedrockEntry{role: bedrockRole(m.OutputModalities), active: active}
	}
	pager := bedrock.NewListInferenceProfilesPaginator(client, &bedrock.ListInferenceProfilesInput{
		TypeEquals: bedrocktypes.InferenceProfileTypeSystemDefined,
	})
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, p := range page.InferenceProfileSummaries {
			if p.InferenceProfileId == nil {
				continue
			}
			e := bedrockEntry{active: p.Status == bedrocktypes.InferenceProfileStatusActive}
			for _, m := range p.Models {
				if m.ModelArn == nil {
					continue
				}
				_, modelID, _ := strings.Cut(*m.ModelArn, "foundation-model/")
				if fm, ok := cat.entries[modelID]; ok {
					e.role = fm.role
					break
				}
			}
			cat.entries[*p.InferenceProfileId] = e
		}
	}
	return cat, nil
}

// bedrockRole classifies a model from its declared output modalities.
func bedrockRole(modalities []bedrocktypes.ModelModality) string {
	switch {
	case slices.Contains(modalities, bedrocktypes.ModelModalityEmbedding):
		return common.ModelRoleEmbedding
	case slices.Contains(modalities, bedrocktypes.ModelModalityText):
		return common.ModelRoleChat
	default:
		return ""
	}
}

// PrepareModels configures every managed model gateway in services.
func PrepareModels(
	ctx *pulumi.Context, services compose.Services, models compose.Models, region string,
	pluginID common.PluginIdentity, opts ...pulumi.ResourceOption,
) (compose.Services, error) {
	return common.GatewayModels{
		Cloud:           ModelCloud,
		Package:         "defang-aws",
		LiteLLMProvider: "bedrock",
		Region:          region,
		PluginID:        pluginID,
		Env: func(env compose.Environment, _ *common.ModelSelectionResult) {
			common.SetEnvDefault(env, "AWS_REGION", pulumi.String(region))
		},
	}.Prepare(ctx, services, models, opts...)
}
