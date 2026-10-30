package azure

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cognitiveservices/armcognitiveservices"
	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
)

var ErrInvalidAccountScope = errors.New("invalid AI Foundry account ID")

// chatPreference and embeddingPreference are the compatibility defaults for
// the built-in aliases when a recipe has no defang-azure:models entry for
// them: the first candidate deployable in the account's region wins. Format
// "OpenAI" models are fully compatible with the OpenAI SDK; marketplace models
// may have compatibility gaps. Preview revisions were always eligible here,
// so every entry allows them.
var chatPreference = azureCandidates(
	"gpt-5.1", "OpenAI",
	"gpt-5", "OpenAI",
	"gpt-5-mini", "OpenAI",
	"gpt-4.1", "OpenAI",
	"gpt-4.1-mini", "OpenAI",
	"gpt-4o", "OpenAI",
	"gpt-4o-mini", "OpenAI",
	"gpt-4", "OpenAI",
	"gpt-35-turbo", "OpenAI",
	"Phi-4", "Microsoft",
	"DeepSeek-V3", "DeepSeek",
	"Mistral-small", "Mistral AI",
	"Meta-Llama-3.1-8B-Instruct", "Meta",
)

var embeddingPreference = azureCandidates(
	"text-embedding-3-small", "OpenAI",
	"text-embedding-3-large", "OpenAI",
	"text-embedding-ada-002", "OpenAI",
	"Cohere-embed-v3-english", "Cohere",
	"Cohere-embed-v3-multilingual", "Cohere",
)

func azureCandidates(nameFormat ...string) []compose.ModelCandidate {
	out := make([]compose.ModelCandidate, 0, len(nameFormat)/2)
	for i := 0; i+1 < len(nameFormat); i += 2 {
		out = append(out, compose.ModelCandidate{Model: nameFormat[i], Format: nameFormat[i+1], AllowPreview: true})
	}
	return out
}

// ModelCloud is Azure's recipe contract: defang-azure:models candidates take
// `model`, `role`, `version`, `format`, `sku` and `allow-preview`. A model
// name selects a family; `version` pins one ARM model revision.
var ModelCloud = common.ModelCloud{
	Name:      "azure",
	Namespace: "defang-azure",
	Fields: []string{
		common.ModelFieldVersion, common.ModelFieldFormat, common.ModelFieldSku, common.ModelFieldAllowPreview,
	},
	Defaults: map[string][]compose.ModelCandidate{
		"chat-default":      chatPreference,
		"chat-large":        chatPreference,
		"embedding-default": embeddingPreference,
	},
	CandidateExact:  func(c compose.ModelCandidate) bool { return c.Version != "" },
	RoleFromCatalog: true,
}

// AvailableModel is a model revision returned by the account's ListModels.
type AvailableModel struct {
	Name            string
	Format          string
	Version         string
	SKUs            []string
	LifecycleStatus string // e.g. "GenerallyAvailable", "Preview", "Deprecating", "Deprecated"
	Capabilities    map[string]string
}

// FoundryResolver resolves defang-azure:models candidates against the models
// an AI Foundry account can deploy. Each account's catalog is listed once.
type FoundryResolver struct {
	List func(ctx context.Context, subscriptionID, resourceGroup, account string) ([]AvailableModel, error)

	mu       sync.Mutex
	catalogs map[string][]AvailableModel
}

// NewFoundryResolver returns a resolver using the default Azure credential.
func NewFoundryResolver() *FoundryResolver {
	return &FoundryResolver{List: listModels}
}

// ResolveCandidate implements common.ModelResolver.
func (r *FoundryResolver) ResolveCandidate(
	ctx context.Context, in common.ModelSelectionInputs, c compose.ModelCandidate,
) (common.ResolvedModel, error) {
	models, err := r.catalog(ctx, in.Scope)
	if err != nil {
		return common.ResolvedModel{}, err
	}
	m, err := pickRevision(models, c)
	if err != nil {
		return common.ResolvedModel{}, err
	}
	sku := c.Sku
	switch {
	case sku == "":
		// The existing default: Standard when offered, else GlobalStandard.
		sku = "GlobalStandard"
		for _, s := range m.SKUs {
			if s == "Standard" {
				sku = "Standard"
				break
			}
		}
	case !containsFold(m.SKUs, sku):
		return common.ResolvedModel{}, fmt.Errorf("%w: %s version %s does not offer SKU %s (offered: %s)",
			common.ErrModelUnavailable, m.Name, m.Version, sku, strings.Join(m.SKUs, ", "))
	}
	log.Printf("FoundryResolver: selected %s (%s, v%s, %s) for %s", m.Name, m.Format, m.Version, sku, in.Request)
	return common.ResolvedModel{
		Target: compose.ModelTarget{Model: m.Name, Format: m.Format, Version: m.Version, Sku: sku},
		Role:   foundryRole(m.Capabilities),
	}, nil
}

// pickRevision selects one revision of exactly the candidate's model name
// (case-insensitive, never a prefix match). A pinned version must exist and
// be deployable; otherwise the highest deployable revision wins, ordered
// numerically ("10" after "9", "2025-04-14" after "2024-11-20").
func pickRevision(models []AvailableModel, c compose.ModelCandidate) (*AvailableModel, error) {
	var named []*AvailableModel
	formats := map[string]bool{}
	for i := range models {
		m := &models[i]
		if !strings.EqualFold(m.Name, c.Model) || (c.Format != "" && !strings.EqualFold(m.Format, c.Format)) {
			continue
		}
		named = append(named, m)
		formats[strings.ToLower(m.Format)] = true
	}
	if len(named) == 0 {
		return nil, fmt.Errorf("%w: %s is not offered by this account", common.ErrModelUnavailable, describeCandidate(c))
	}
	if len(formats) > 1 {
		return nil, fmt.Errorf("%w: %s exists in several formats; set `format` in the recipe candidate",
			common.ErrModelAmbiguous, c.Model)
	}

	if c.Version != "" {
		for _, m := range named {
			if m.Version != c.Version {
				continue
			}
			if !isDeployable(m.LifecycleStatus) {
				return nil, fmt.Errorf("%w: %s version %s is %s and cannot be deployed; pin a current version",
					common.ErrModelUnavailable, m.Name, m.Version, m.LifecycleStatus)
			}
			return m, nil
		}
		return nil, fmt.Errorf("%w: %s version %s is not offered by this account", common.ErrModelUnavailable,
			c.Model, c.Version)
	}

	var best *AvailableModel
	for _, m := range named {
		if !isDeployable(m.LifecycleStatus) || (isPreview(m.LifecycleStatus) && !c.AllowPreview) {
			continue
		}
		if best == nil || common.CompareRevisions(m.Version, best.Version) > 0 {
			best = m
		}
	}
	if best == nil {
		return nil, fmt.Errorf("%w: %s has no deployable revision (preview revisions need allow-preview)",
			common.ErrModelUnavailable, describeCandidate(c))
	}
	return best, nil
}

func describeCandidate(c compose.ModelCandidate) string {
	if c.Format != "" {
		return c.Model + " (" + c.Format + ")"
	}
	return c.Model
}

// foundryRole classifies a model from the capabilities ListModels reports.
func foundryRole(capabilities map[string]string) string {
	switch {
	case strings.EqualFold(capabilities["embeddings"], "true"):
		return common.ModelRoleEmbedding
	case strings.EqualFold(capabilities["chatCompletion"], "true"):
		return common.ModelRoleChat
	default:
		return ""
	}
}

func (r *FoundryResolver) catalog(ctx context.Context, scope string) ([]AvailableModel, error) {
	account, err := parseAccountID(scope)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if models, ok := r.catalogs[scope]; ok {
		return models, nil
	}
	models, err := r.List(ctx, account.subscriptionID, account.resourceGroup, account.name)
	if err != nil {
		return nil, err
	}
	if r.catalogs == nil {
		r.catalogs = map[string][]AvailableModel{}
	}
	r.catalogs[scope] = models
	return models, nil
}

// accountRef names an AI Foundry account.
type accountRef struct {
	subscriptionID, resourceGroup, name string
}

// parseAccountID splits
// /subscriptions/<sub>/resourceGroups/<rg>/providers/Microsoft.CognitiveServices/accounts/<name>.
func parseAccountID(id string) (accountRef, error) {
	parts := strings.Split(strings.Trim(id, "/"), "/")
	if len(parts) != 8 || !strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") || !strings.EqualFold(parts[6], "accounts") {
		return accountRef{}, fmt.Errorf("%w: %q", ErrInvalidAccountScope, id)
	}
	return accountRef{subscriptionID: parts[1], resourceGroup: parts[3], name: parts[7]}, nil
}

// listModels lists the models available for a specific Azure AI Foundry account.
func listModels(ctx context.Context, subscriptionID, resourceGroup, accountName string) ([]AvailableModel, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("creating credential for model listing: %w", err)
	}
	client, err := armcognitiveservices.NewAccountsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("creating cognitive services client: %w", err)
	}
	pager := client.NewListModelsPager(resourceGroup, accountName, nil)
	var models []AvailableModel
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing models of account %s: %w", accountName, err)
		}
		for _, m := range page.Value {
			if m == nil || m.Name == nil {
				continue
			}
			am := AvailableModel{Name: *m.Name, Format: derefStr(m.Format), Version: derefStr(m.Version)}
			if m.LifecycleStatus != nil {
				am.LifecycleStatus = string(*m.LifecycleStatus)
			}
			for _, s := range m.SKUs {
				if s != nil && s.Name != nil {
					am.SKUs = append(am.SKUs, *s.Name)
				}
			}
			if len(m.Capabilities) > 0 {
				am.Capabilities = make(map[string]string, len(m.Capabilities))
				for k, v := range m.Capabilities {
					am.Capabilities[k] = derefStr(v)
				}
			}
			models = append(models, am)
		}
	}
	log.Printf("FoundryResolver: found %d models in account %s", len(models), accountName)
	return models, nil
}

// isDeployable reports whether a model version can still be deployed for the
// first time. Azure keeps returning models from ListModels after closing them
// to new deployments, so the selector must screen them out itself:
//   - "Deprecating": blocked for new deployments — NewDeployment fails with
//     HTTP 400 ServiceModelDeprecating — even though existing deployments keep
//     working until the model's retirement date.
//   - "Deprecated": fully retired (HTTP 410).
//
// Every other status (GenerallyAvailable, Stable, Preview, or an empty/unknown
// value) is treated as deployable, so an absent status never over-filters.
func isDeployable(lifecycleStatus string) bool {
	// Match case-insensitively — a denylist can only ever err toward
	// eligible, so this never over-filters an absent/unknown status.
	switch {
	case strings.EqualFold(lifecycleStatus, string(armcognitiveservices.ModelLifecycleStatusDeprecating)),
		strings.EqualFold(lifecycleStatus, string(armcognitiveservices.ModelLifecycleStatusDeprecated)):
		return false
	default:
		return true
	}
}

// isPreview reports whether a revision is a preview, which a floating
// candidate selects only with allow-preview.
func isPreview(lifecycleStatus string) bool {
	return strings.EqualFold(lifecycleStatus, string(armcognitiveservices.ModelLifecycleStatusPreview))
}

func containsFold(list []string, s string) bool {
	for _, v := range list {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}

func derefStr(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
