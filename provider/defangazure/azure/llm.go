package azure

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	cognitiveservices "github.com/pulumi/pulumi-azure-native-sdk/cognitiveservices/v3"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ErrModelSelectorUnavailable is returned from CreateLLMDeployment when the
// selector is nil (typically during `pulumi preview`, which skips ARM queries).
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
	// Listing is deferred until the Account's Name output resolves, so the
	// ARM list call only runs after Azure has finished creating the account.
	ModelSelector ModelSelector
}

// CreateLLMInfra creates an Azure AI Foundry hub (Account with Kind "AIServices") and
// builds a lazy ModelSelector that will query the account for available models
// after it has been created.
func CreateLLMInfra(
	ctx *pulumi.Context,
	name string,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*LLMInfra, error) {
	// CustomSubDomainName is DNS-scoped globally (AI Foundry hands out
	// `<subdomain>.cognitiveservices.azure.com`), so it must be unique across
	// all of Azure. Azure also *reserves* a deleted account's subdomain for
	// ~48h before releasing it, so any naming scheme derived from stable
	// inputs (project/stack/subscription) collides with itself after a
	// `down`-then-`up` inside that window.
	//
	// Use pulumi-random's RandomString to get a suffix that's persisted in
	// state: stable across subsequent `up`s, but regenerated when state is
	// destroyed (so `down`-then-`up` produces a fresh name, sidestepping the
	// soft-delete reservation).
	subDomainSuffix, err := random.NewRandomString(ctx, name+"-subdomain", &random.RandomStringArgs{
		Length:  pulumi.Int(8),
		Lower:   pulumi.Bool(true),
		Upper:   pulumi.Bool(false),
		Numeric: pulumi.Bool(true),
		Special: pulumi.Bool(false),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating random subdomain suffix: %w", err)
	}
	subDomainName := subDomainSuffix.Result.ApplyT(func(suffix string) string {
		return llmSubDomainPrefix(name) + "-" + suffix
	}).(pulumi.StringOutput)

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
			CustomSubDomainName:    subDomainName.ToStringPtrOutput(),
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

	// Skip the selector during preview — its lazy ARM ListModels call fires
	// as soon as rgName+accountName are known, which can happen during
	// `pulumi preview` if the names are static Strings (they're unknown
	// here, but a cautious guard is cheaper than debugging preview-time
	// ARM slowness). Consumers get ErrModelSelectorUnavailable in preview.
	var selector ModelSelector
	if !ctx.DryRun() {
		selector = NewDynamicModelSelector(
			SubscriptionID(ctx),
			infra.ResourceGroup.Name.ToStringOutput(),
			account.Name.ToStringOutput(),
		)
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
	if llmInfra.ModelSelector == nil {
		return ErrModelSelectorUnavailable
	}
	specOutput := selectModelForAlias(modelAlias, llmInfra.ModelSelector)

	// Chained ApplyT over an AnyOutput requires `any` as the applier's input —
	// pulumi's reflection reads the Output's inner type as `interface{}` when
	// the source was produced via `pulumi.All(...).ApplyT(...)`.
	pickField := func(get func(ModelSpec) string) pulumi.StringOutput {
		return specOutput.ApplyT(func(raw any) (string, error) {
			spec, _ := raw.(ModelSpec)
			return get(spec), nil
		}).(pulumi.StringOutput)
	}
	modelFormat := pickField(func(s ModelSpec) string { return s.Format })
	modelName := pickField(func(s ModelSpec) string { return s.Name })
	modelVersion := pickField(func(s ModelSpec) string { return s.Version })
	modelSKU := pickField(func(s ModelSpec) string { return s.SKU })

	_, err := cognitiveservices.NewDeployment(ctx, deploymentName, &cognitiveservices.DeploymentArgs{
		AccountName:       llmInfra.Account.Name,
		ResourceGroupName: infra.ResourceGroup.Name,
		DeploymentName:    pulumi.String(deploymentName),
		Properties: &cognitiveservices.DeploymentPropertiesArgs{
			Model: &cognitiveservices.DeploymentModelArgs{
				Format:  modelFormat.ToStringPtrOutput(),
				Name:    modelName.ToStringPtrOutput(),
				Version: modelVersion.ToStringPtrOutput(),
			},
		},
		Sku: &cognitiveservices.SkuArgs{
			Name:     modelSKU,
			Capacity: pulumi.Int(1),
		},
	}, append(opts, pulumi.Parent(llmInfra.Account))...)
	if err != nil {
		return fmt.Errorf("creating Azure AI Foundry deployment %s: %w", deploymentName, err)
	}
	return nil
}

// subDomainPrefixRe matches characters to strip from an AI Foundry custom
// subdomain: anything outside `a-z0-9-`, plus leading/trailing hyphens.
var subDomainPrefixRe = regexp.MustCompile(`^-+|[^a-z0-9-]|-+$`)

// llmSubDomainPrefix sanitizes name into the leading portion of an AI Foundry
// custom subdomain: lowercase letters, digits, and hyphens only, ≤15 chars so
// there's room for "-<8-char suffix>" inside Azure's 24-char limit.
func llmSubDomainPrefix(name string) string {
	prefix := subDomainPrefixRe.ReplaceAllString(strings.ToLower(name), "")
	if len(prefix) > 15 {
		prefix = strings.Trim(prefix[:15], "-")
	}
	return prefix
}

// selectModelForAlias maps a Defang model alias to an Output of ModelSpec.
func selectModelForAlias(alias string, selector ModelSelector) pulumi.Output {
	var role ModelRole
	switch {
	case strings.Contains(alias, "embedding"):
		role = ModelRoleEmbedding
	default:
		role = ModelRoleChat
	}
	return selector.SelectModel(role)
}
