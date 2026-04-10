package azure

import (
	"fmt"
	"strings"

	cognitiveservices "github.com/pulumi/pulumi-azure-native-sdk/cognitiveservices/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// LLMInfra holds a shared Azure AI Foundry account for all LLM services.
type LLMInfra struct {
	Account *cognitiveservices.Account

	// APIKey is the primary access key for the Azure AI Foundry account.
	APIKey pulumi.StringOutput

	// BaseURL is the OpenAI-compatible v1 base URL for this account.
	// Format: "https://{name}.services.ai.azure.com/openai/v1/"
	// Compatible with OpenAI SDK's base_url parameter.
	BaseURL pulumi.StringOutput
}

// azureModelForAlias maps a Defang model alias to an Azure AI Foundry model name and format.
// Deployments are named after the alias (e.g. "chat-default") so that requests with
// model="chat-default" route to the correct deployment automatically.
//
// Model availability varies by region. text-embedding-ada-002 is used for embeddings
// because it has broad regional availability (text-embedding-3-small is limited).
func azureModelForAlias(alias string) (string, string) {
	switch alias {
	case "embedding-default":
		return "text-embedding-ada-002", "OpenAI"
	default: // "chat-default" and any unrecognised alias
		return "gpt-4o", "OpenAI"
	}
}

// CreateLLMInfra creates an Azure AI Foundry hub (Account with Kind "AIServices") and a
// child Project for the given stack. All LLM service deployments are created under this
// account via CreateLLMDeployment.
func CreateLLMInfra(
	ctx *pulumi.Context,
	name string,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*LLMInfra, error) {
	account, err := cognitiveservices.NewAccount(ctx, name+"-foundry", &cognitiveservices.AccountArgs{
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
			// CustomSubDomainName is required before creating child projects and must be
			// globally unique. Derive it from the RG's random suffix (same approach as postgres).
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
	// CustomSubDomainName is immutable after creation; force replacement if it changes.
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

	// Base URL: append "/openai/v1/" to the account endpoint.
	// For AIServices accounts the endpoint is "https://{name}.services.ai.azure.com/".
	// The /openai/v1/ path exposes the OpenAI-compatible API used by the OpenAI SDK.
	baseURL := account.Properties.Endpoint().ApplyT(func(ep string) string {
		return strings.TrimRight(ep, "/") + "/openai/v1/"
	}).(pulumi.StringOutput)

	return &LLMInfra{
		Account: account,
		APIKey:  apiKey,
		BaseURL: baseURL,
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
	azModelName, azFormat := azureModelForAlias(modelAlias)

	_, err := cognitiveservices.NewDeployment(ctx, deploymentName, &cognitiveservices.DeploymentArgs{
		AccountName:       llmInfra.Account.Name,
		ResourceGroupName: infra.ResourceGroup.Name,
		DeploymentName:    pulumi.String(deploymentName),
		Properties: &cognitiveservices.DeploymentPropertiesArgs{
			Model: &cognitiveservices.DeploymentModelArgs{
				Format: pulumi.String(azFormat),
				Name:   pulumi.String(azModelName),
			},
		},
		Sku: &cognitiveservices.SkuArgs{
			Name:     pulumi.String("Standard"),
			Capacity: pulumi.Int(10), // tokens-per-minute in thousands
		},
	}, append(opts, pulumi.Parent(llmInfra.Account))...)
	if err != nil {
		return fmt.Errorf("creating Azure AI Foundry deployment %s (%s): %w", deploymentName, azModelName, err)
	}
	return nil
}
