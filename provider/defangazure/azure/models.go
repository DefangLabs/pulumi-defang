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
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
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

// ModelSelector picks the best available model for a given role. Returns a
// pulumi Output so the ARM "list models" call is deferred until the Account's
// Name output resolves, i.e. after the AIServices account has actually been
// created in Azure (not just registered in Pulumi state).
type ModelSelector interface {
	SelectModel(role ModelRole) pulumi.Output
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
// at deploy time (not preview) after the account has been created.
//
// Model listing runs inside a pulumi ApplyT gated on rgName+accountName, so
// it blocks until the account's Name output is known — i.e. after Azure has
// finished creating it. The result is cached via sync.Once so every
// SelectModel call shares the same ARM round-trip.
type DynamicModelSelector struct {
	subscriptionID string
	rgName         pulumi.StringOutput
	accountName    pulumi.StringOutput

	once   sync.Once
	models pulumi.Output // cached []availableModel
}

// NewDynamicModelSelector builds a selector that lists models lazily, after
// the Account represented by accountName has been created.
func NewDynamicModelSelector(subscriptionID string, rgName, accountName pulumi.StringOutput) *DynamicModelSelector {
	return &DynamicModelSelector{
		subscriptionID: subscriptionID,
		rgName:         rgName,
		accountName:    accountName,
	}
}

// modelsOutput returns a lazily-computed pulumi.Output wrapping []availableModel.
// The ARM list call runs at most once per selector.
func (s *DynamicModelSelector) modelsOutput() pulumi.Output {
	s.once.Do(func() {
		s.models = pulumi.All(s.rgName, s.accountName).ApplyT(
			func(args []interface{}) ([]availableModel, error) {
				rg, _ := args[0].(string)
				acct, _ := args[1].(string)
				return listModels(s.subscriptionID, rg, acct)
			},
		)
	})
	return s.models
}

// SelectModel returns a pulumi.Output that resolves to a ModelSpec for the
// given role. The resolution depends on the Account's Name output, so it
// runs only after Azure has created the AIServices account.
func (s *DynamicModelSelector) SelectModel(role ModelRole) pulumi.Output {
	return s.modelsOutput().ApplyT(func(raw any) (ModelSpec, error) {
		models, _ := raw.([]availableModel)
		return pickModel(models, role)
	})
}

// listModels lists the models available for a specific Azure AI Foundry account.
func listModels(subscriptionID, resourceGroup, accountName string) ([]availableModel, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("creating credential for model listing: %w", err)
	}

	client, err := armcognitiveservices.NewAccountsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("creating cognitive services client: %w", err)
	}

	ctx := context.TODO()

	if accountName == "" {
		return nil, fmt.Errorf("%w: resource group %s", ErrNoAIServicesAccount, resourceGroup)
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
	return models, nil
}

// pickModel selects the best available model for the given role from a list.
func pickModel(models []availableModel, role ModelRole) (ModelSpec, error) {
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
		best := findBest(models, pref.name, pref.format)
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

// findBest returns the highest-version model matching name and format.
func findBest(models []availableModel, name, format string) *availableModel {
	var best *availableModel
	for i := range models {
		m := &models[i]
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
