package azure

import (
	"errors"
	"fmt"
	"strings"

	cognitiveservices "github.com/pulumi/pulumi-azure-native-sdk/cognitiveservices/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var ErrModelSelectorUnavailable = errors.New("model selector not available (preview mode?)")

// LLMInfra holds a shared Azure AI Foundry account for all LLM services.
type LLMInfra struct {
	Account *cognitiveservices.Account

	// APIKey is the primary access key for the Azure AI Foundry account.
	APIKey pulumi.StringOutput

	// BaseURL is the OpenAI-compatible v1 base URL for this account.
	// Format: "https://{name}.services.ai.azure.com/openai/v1/"
	// Compatible with OpenAI SDK's base_url parameter.
	BaseURL pulumi.StringOutput

	// ModelSelector picks models at deploy time based on region availability.
	ModelSelector ModelSelector
}

// CreateLLMInfra creates an Azure AI Foundry hub (Account with Kind "AIServices") and
// initializes a ModelSelector that queries the account for available models.
func CreateLLMInfra(
	ctx *pulumi.Context,
	name string,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*LLMInfra, error) {
	account, err := cognitiveservices.NewAccount(ctx, name, &cognitiveservices.AccountArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		Location:          pulumi.StringPtr(Location(ctx)),
		Kind:              pulumi.String("AIServices"),
		Sku: &cognitiveservices.SkuArgs{
			Name: pulumi.String("S0"),
		},
		Identity: &cognitiveservices.IdentityArgs{
			Type: cognitiveservices.ResourceIdentityTypeSystemAssigned.ToResourceIdentityTypePtrOutput(),
		},
		Properties: &cognitiveservices.AccountPropertiesArgs{
			AllowProjectManagement: pulumi.Bool(true),
			CustomSubDomainName: infra.ResourceGroup.Name.ApplyT(func(rgName string) string {
				suffix := rgName
				if idx := strings.LastIndexByte(rgName, '-'); idx >= 0 {
					suffix = rgName[idx+1:]
				}
				s := strings.ToLower(name) + "-" + suffix
				if len(s) > 24 {
					s = s[:24]
				}
				return s
			}).(pulumi.StringOutput).ToStringPtrOutput(),
		},
	}, append(opts, pulumi.ReplaceOnChanges([]string{"properties.customSubDomainName"}))...)
	if err != nil {
		return nil, fmt.Errorf("creating Azure AI Foundry account: %w", err)
	}

	keysOut := cognitiveservices.ListAccountKeysOutput(ctx, cognitiveservices.ListAccountKeysOutputArgs{
		AccountName:       account.Name,
		ResourceGroupName: infra.ResourceGroup.Name,
	})

	apiKey := keysOut.Key1().ApplyT(func(k *string) string {
		if k != nil {
			return *k
		}
		return ""
	}).(pulumi.StringOutput)

	baseURL := account.Properties.Endpoint().ApplyT(func(ep string) string {
		return strings.TrimRight(ep, "/") + "/openai/v1/"
	}).(pulumi.StringOutput)

	// Query available models for this account. This runs at deploy time (not preview).
	// Pass empty account name — the selector discovers it from the resource group.
	var selector ModelSelector
	if !ctx.DryRun() {
		rgName := ExistingResourceGroup(ctx)
		selector, err = NewDynamicModelSelector(SubscriptionID(ctx), rgName, "")
		if err != nil {
			return nil, fmt.Errorf("initializing model selector: %w", err)
		}
	}

	return &LLMInfra{
		Account:       account,
		APIKey:        apiKey,
		BaseURL:       baseURL,
		ModelSelector: selector,
	}, nil
}

// CreateLLMDeployment creates an Azure AI model deployment under the shared Foundry account.
// deploymentName should be the Defang model alias (e.g. "chat-default", "embedding-default")
// so that requests with model="{alias}" route to this deployment automatically.
func CreateLLMDeployment(
	ctx *pulumi.Context,
	deploymentName string,
	modelAlias string,
	llmInfra *LLMInfra,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) error {
	model, err := selectModelForAlias(modelAlias, llmInfra.ModelSelector)
	if err != nil {
		return fmt.Errorf("selecting model for %s: %w", modelAlias, err)
	}

	_, err = cognitiveservices.NewDeployment(ctx, deploymentName, &cognitiveservices.DeploymentArgs{
		AccountName:       llmInfra.Account.Name,
		ResourceGroupName: infra.ResourceGroup.Name,
		DeploymentName:    pulumi.String(deploymentName),
		Properties: &cognitiveservices.DeploymentPropertiesArgs{
			Model: &cognitiveservices.DeploymentModelArgs{
				Format:  pulumi.String(model.Format),
				Name:    pulumi.String(model.Name),
				Version: pulumi.String(model.Version),
			},
		},
		Sku: &cognitiveservices.SkuArgs{
			Name:     pulumi.String(model.SKU),
			Capacity: pulumi.Int(1),
		},
	}, append(opts, pulumi.Parent(llmInfra.Account))...)
	if err != nil {
		return fmt.Errorf(
			"creating Azure AI Foundry deployment %s (%s/%s): %w",
			deploymentName, model.Format, model.Name, err,
		)
	}
	return nil
}

// selectModelForAlias maps a Defang model alias to a concrete ModelSpec.
func selectModelForAlias(alias string, selector ModelSelector) (ModelSpec, error) {
	if selector == nil {
		return ModelSpec{}, ErrModelSelectorUnavailable
	}

	var role ModelRole
	switch {
	case strings.Contains(alias, "embedding"):
		role = ModelRoleEmbedding
	default:
		role = ModelRoleChat
	}
	return selector.SelectModel(role)
}
