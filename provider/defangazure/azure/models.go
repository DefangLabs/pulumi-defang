package azure

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cognitiveservices/armcognitiveservices"
)

var (
	ErrNoAIServicesAccount = errors.New("no AIServices account found")
	ErrUnknownModelRole    = errors.New("unknown model role")
	ErrNoSuitableModel     = errors.New("no suitable model available in this account/region")
)

// ModelSpec describes an Azure AI model to deploy.
type ModelSpec struct {
	Name    string // e.g. "Phi-4", "text-embedding-3-small"
	Format  string // e.g. "OpenAI", "Microsoft", "Cohere"
	Version string // e.g. "5", "1"
	SKU     string // e.g. "Standard", "GlobalStandard"
}

// ModelRole identifies what kind of model is needed.
type ModelRole string

const (
	ModelRoleChat      ModelRole = "chat"
	ModelRoleEmbedding ModelRole = "embedding"
)

// ModelSelector picks the best available model for a given role.
// Implementations can query Azure at deploy time or use a static mapping.
type ModelSelector interface {
	// SelectModel returns the best available model for the given role,
	// or an error if no suitable model is found.
	SelectModel(role ModelRole) (ModelSpec, error)
}

// chatPreference and embeddingPreference define model preference order.
// Earlier entries are preferred. Format "OpenAI" models are fully compatible
// with the OpenAI SDK; marketplace models may have compatibility gaps.
var chatPreference = []struct {
	name, format string
}{
	{"gpt-4o", "OpenAI"},
	{"gpt-4o-mini", "OpenAI"},
	{"gpt-4", "OpenAI"},
	{"gpt-35-turbo", "OpenAI"},
	{"Phi-4", "Microsoft"},
	{"DeepSeek-V3", "DeepSeek"},
	{"Mistral-small", "Mistral AI"},
	{"Meta-Llama-3.1-8B-Instruct", "Meta"},
}

var embeddingPreference = []struct {
	name, format string
}{
	{"text-embedding-3-small", "OpenAI"},
	{"text-embedding-3-large", "OpenAI"},
	{"text-embedding-ada-002", "OpenAI"},
	{"Cohere-embed-v3-english", "Cohere"},
	{"Cohere-embed-v3-multilingual", "Cohere"},
}

// availableModel is a model returned by the Azure API.
type availableModel struct {
	name    string
	format  string
	version string
	skus    []string
}

// DynamicModelSelector queries the Azure AI account for available models
// at deploy time and picks the best one based on preference order.
type DynamicModelSelector struct {
	models []availableModel
}

// NewDynamicModelSelector queries the Foundry account's available models
// and returns a selector that picks the best match at deploy time.
// If accountName is empty, the first AIServices account in the resource group is used.
func NewDynamicModelSelector(subscriptionID, resourceGroup, accountName string) (*DynamicModelSelector, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("creating credential for model listing: %w", err)
	}

	client, err := armcognitiveservices.NewAccountsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("creating cognitive services client: %w", err)
	}

	ctx := context.Background()

	if accountName == "" {
		rgPager := client.NewListByResourceGroupPager(resourceGroup, nil)
		for rgPager.More() {
			page, err := rgPager.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("listing accounts in RG: %w", err)
			}
			for _, acct := range page.Value {
				if acct.Kind != nil && *acct.Kind == "AIServices" && acct.Name != nil {
					accountName = *acct.Name
					break
				}
			}
			if accountName != "" {
				break
			}
		}
		if accountName == "" {
			return nil, fmt.Errorf("%w: resource group %s", ErrNoAIServicesAccount, resourceGroup)
		}
	}

	pager := client.NewListModelsPager(resourceGroup, accountName, nil)

	var models []availableModel
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing models: %w", err)
		}
		for _, m := range page.Value {
			if m.Name == nil {
				continue
			}
			am := availableModel{
				name:   *m.Name,
				format: derefStr(m.Format),
			}
			if m.Version != nil {
				am.version = *m.Version
			}
			for _, s := range m.SKUs {
				if s != nil && s.Name != nil {
					am.skus = append(am.skus, *s.Name)
				}
			}
			models = append(models, am)
		}
	}

	log.Printf("DynamicModelSelector: found %d models in account %s", len(models), accountName)
	return &DynamicModelSelector{models: models}, nil
}

// SelectModel picks the best available model for the given role.
func (s *DynamicModelSelector) SelectModel(role ModelRole) (ModelSpec, error) {
	var prefs []struct{ name, format string }
	switch role {
	case ModelRoleChat:
		prefs = chatPreference
	case ModelRoleEmbedding:
		prefs = embeddingPreference
	default:
		return ModelSpec{}, fmt.Errorf("%w: %s", ErrUnknownModelRole, role)
	}

	for _, pref := range prefs {
		best := s.findBest(pref.name, pref.format)
		if best != nil {
			sku := "GlobalStandard"
			for _, sk := range best.skus {
				if sk == "Standard" {
					sku = "Standard"
					break
				}
			}
			log.Printf("DynamicModelSelector: selected %s (%s, v%s, %s) for %s", best.name, best.format, best.version, sku, role)
			return ModelSpec{
				Name:    best.name,
				Format:  best.format,
				Version: best.version,
				SKU:     sku,
			}, nil
		}
	}

	return ModelSpec{}, fmt.Errorf("%w: %s role", ErrNoSuitableModel, role)
}

// findBest returns the model with the highest version matching the given name and format.
func (s *DynamicModelSelector) findBest(name, format string) *availableModel {
	var best *availableModel
	for i := range s.models {
		m := &s.models[i]
		if !strings.EqualFold(m.name, name) || !strings.EqualFold(m.format, format) {
			continue
		}
		if best == nil || m.version > best.version {
			best = m
		}
	}
	return best
}

func derefStr(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
