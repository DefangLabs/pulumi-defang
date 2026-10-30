package azure

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	cognitiveservices "github.com/pulumi/pulumi-azure-native-sdk/cognitiveservices/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/network/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/privatedns/v3"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
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

	// lastDeployment is the most recently-registered Deployment under this
	// account; subsequent CreateLLMDeployment calls DependsOn it to serialize
	// Azure's per-account deployment operations. Azure returns 409
	// "Another operation is being performed on the parent resource" if two
	// Deployments on the same Account run in parallel.
	lastDeployment pulumi.Resource
}

// CreateLLMInfra creates an Azure AI Foundry hub (Account with Kind "AIServices").
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

	// When a project VNet is available, lock the account to private-endpoint-only
	// access (data-plane unreachable from the public internet); otherwise leave
	// it Enabled so the API key — distributed via Key Vault to the user's
	// Container Apps — remains the only auth gate. The PATCH-time `Required`
	// quirk is satisfied either way because the field is now always explicit.
	useVNet := infra.Networking != nil && infra.DNS != nil
	publicAccess := pulumi.String("Enabled")
	if useVNet {
		publicAccess = pulumi.String("Disabled")
	}
	account, err := cognitiveservices.NewAccount(ctx, name, &cognitiveservices.AccountArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		// Location:          pulumi.StringPtr(config.GetLocation(ctx)),
		Kind: pulumi.String("AIServices"),
		Sku: &cognitiveservices.SkuArgs{
			Name: pulumi.String("S0"),
		},
		Identity: &cognitiveservices.IdentityArgs{
			Type: cognitiveservices.ResourceIdentityTypeSystemAssigned.ToResourceIdentityTypePtrOutput(),
		},
		Properties: &cognitiveservices.AccountPropertiesArgs{
			AllowProjectManagement: pulumi.Bool(true),
			CustomSubDomainName:    subDomainName.ToStringPtrOutput(),
			PublicNetworkAccess:    publicAccess,
		},
		Tags: ServiceTags(name),
	}, append(opts, pulumi.ReplaceOnChanges([]string{"properties.customSubDomainName"}))...)
	if err != nil {
		return nil, fmt.Errorf("creating Azure AI Foundry account: %w", err)
	}

	// pulumi.Parent(account) routes the invoke through the account's provider —
	// required because pulumi:disable-default-providers excludes azure-native
	// (see cd/main.go projectConfig).
	//
	// Called *before* the PE so its resolution can act as a sync barrier:
	// pulumi-azure-native treats the account LRO as complete at state=Accepted,
	// but ListAccountKeys won't succeed until the account is actually
	// provisioned. Threading keysOut.Key1() into the PE's PrivateLinkServiceId
	// gates PE creation on real account readiness.
	keysOut := cognitiveservices.ListAccountKeysOutput(ctx, cognitiveservices.ListAccountKeysOutputArgs{
		AccountName:       account.Name,
		ResourceGroupName: infra.ResourceGroup.Name,
	}, pulumi.Parent(account))

	if useVNet {
		if err := createLLMPrivateEndpoint(ctx, name, account, keysOut, infra, opts...); err != nil {
			return nil, fmt.Errorf("creating LLM private endpoint: %w", err)
		}
	}

	apiKey := keysOut.Key1().ApplyT(func(k *string) string {
		if k != nil {
			return *k
		}
		return ""
	}).(pulumi.StringOutput)

	baseURL := account.Properties.Endpoint().ApplyT(func(ep string) string {
		return strings.TrimRight(ep, "/") + "/openai/v1/"
	}).(pulumi.StringOutput)

	return &LLMInfra{
		Account: account,
		APIKey:  apiKey,
		BaseURL: baseURL,
	}, nil
}

// CreateLLMDeployment records the model selection for a planned request and
// creates the Azure AI model deployment for it under the shared Foundry
// account. deploymentName is the name applications send as "model" (the
// request), so requests route to this deployment.
func CreateLLMDeployment(
	ctx *pulumi.Context,
	serviceName string,
	deploymentName string,
	plan common.ModelPlan,
	seed *compose.ModelTarget,
	llmInfra *LLMInfra,
	infra *SharedInfra,
	pluginID common.PluginIdentity,
	opts ...pulumi.ResourceOption,
) (*common.ModelSelectionResult, error) {
	// The selection lists the account's models, so it waits for the account.
	sel, err := common.RegisterModelSelection(ctx, "defang-azure", serviceName, plan, "",
		llmInfra.Account.ID().ToStringOutput(), seed, pluginID, opts...)
	if err != nil {
		return nil, err
	}

	// Azure serializes deployments on a single AI Services Account — parallel
	// NewDeployment calls fail with 409 "Another operation is being performed
	// on the parent resource". Chain each new deployment on the previously-
	// registered one so Pulumi serializes them in its run.
	if llmInfra.lastDeployment != nil {
		opts = append(opts, pulumi.DependsOn([]pulumi.Resource{llmInfra.lastDeployment}))
	}

	deployment, err := cognitiveservices.NewDeployment(ctx, deploymentName, &cognitiveservices.DeploymentArgs{
		AccountName:       llmInfra.Account.Name,
		ResourceGroupName: infra.ResourceGroup.Name,
		DeploymentName:    pulumi.String(deploymentName),
		Properties: &cognitiveservices.DeploymentPropertiesArgs{
			Model: &cognitiveservices.DeploymentModelArgs{
				Format:  sel.Format.ToStringPtrOutput(),
				Name:    sel.Model.ToStringPtrOutput(),
				Version: sel.Version.ToStringPtrOutput(),
			},
		},
		Sku: &cognitiveservices.SkuArgs{
			Name:     sel.Sku,
			Capacity: pulumi.Int(LLMDeploymentCapacity.Get(ctx)),
		},
	}, append(opts, pulumi.Parent(llmInfra.Account))...)
	if err != nil {
		return nil, fmt.Errorf("creating Azure AI Foundry deployment %s: %w", deploymentName, err)
	}
	llmInfra.lastDeployment = deployment
	return sel, nil
}

// subDomainPrefixRe matches characters to strip from an AI Foundry custom
// subdomain: anything outside `a-z0-9-`, plus leading/trailing hyphens.
var subDomainPrefixRe = regexp.MustCompile(`^-+|[^a-z0-9-]|-+$`)

// llmSubDomainPrefix sanitizes name into the leading portion of an AI Foundry
// custom subdomain: lowercase letters, digits, and hyphens only, ≤15 chars so
// there's room for "-<8-char suffix>" inside Azure's 24-char limit.
func llmSubDomainPrefix(name string) string {
	prefix := subDomainPrefixRe.ReplaceAllLiteralString(strings.ToLower(name), "")
	if len(prefix) > 15 {
		prefix = strings.Trim(prefix[:15], "-")
	} else if len(prefix) == 0 {
		prefix = "llm"
	}
	return prefix
}

// createLLMPrivateEndpoint provisions a private endpoint for the AI Foundry
// account, two privatelink DNS zones (cognitiveservices.azure.com and
// openai.azure.com — the AIServices kind exposes both data planes), VNet links
// for each zone, and a PrivateDnsZoneGroup binding both zones to the endpoint.
// Once applied, the account's data plane endpoint is reachable only from
// within the VNet via the private endpoint IP — combined with
// PublicNetworkAccess=Disabled on the account, the endpoint is not callable
// from the public internet at all.
func createLLMPrivateEndpoint(
	ctx *pulumi.Context,
	serviceName string,
	account *cognitiveservices.Account,
	keysOut cognitiveservices.ListAccountKeysResultOutput,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) error {
	peName := serviceName + "-llm"
	// Gate PrivateLinkServiceId on keysOut so PE creation waits for the
	// account to be actually provisioned. Plain account.ID() resolves at
	// state=Accepted, which Azure rejects with AccountProvisioningStateInvalid
	// when used as PE PrivateLinkServiceId.
	gatedAccountID := pulumi.All(account.ID(), keysOut.Key1()).ApplyT(
		func(args []interface{}) string {
			return string(args[0].(pulumi.ID))
		},
	).(pulumi.StringOutput)
	pe, err := network.NewPrivateEndpoint(ctx, peName, &network.PrivateEndpointArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		// Location:          pulumi.StringPtr(location),
		Subnet: &network.SubnetTypeArgs{
			Id: infra.Networking.PrivateEndpointsSubnet.ID().ToStringOutput(),
		},
		PrivateLinkServiceConnections: network.PrivateLinkServiceConnectionArray{
			&network.PrivateLinkServiceConnectionArgs{
				Name:                 pulumi.String(peName),
				PrivateLinkServiceId: gatedAccountID,
				GroupIds:             pulumi.StringArray{pulumi.String("account")},
			},
		},
		Tags: ServiceTags("llm"),
	}, opts...)
	if err != nil {
		return fmt.Errorf("creating LLM private endpoint: %w", err)
	}

	csZone, csZoneOpts, err := createLLMPrivatelinkZone(
		ctx, serviceName+"-cs", "privatelink.cognitiveservices.azure.com", infra, opts...)
	if err != nil {
		return err
	}
	if _, err := privatedns.NewVirtualNetworkLink(ctx, serviceName+"-cs", &privatedns.VirtualNetworkLinkArgs{
		ResourceGroupName:   infra.ResourceGroup.Name,
		PrivateZoneName:     csZone.Name,
		Location:            pulumi.String("global"),
		RegistrationEnabled: pulumi.Bool(false),
		VirtualNetwork:      &privatedns.SubResourceArgs{Id: infra.Networking.VNet.ID().ToStringOutput()},
	}, csZoneOpts...); err != nil {
		return fmt.Errorf("linking cognitiveservices private DNS zone to VNet: %w", err)
	}

	openaiZone, openaiZoneOpts, err := createLLMPrivatelinkZone(
		ctx, serviceName+"-openai", "privatelink.openai.azure.com", infra, opts...)
	if err != nil {
		return err
	}
	if _, err := privatedns.NewVirtualNetworkLink(ctx, serviceName+"-openai", &privatedns.VirtualNetworkLinkArgs{
		ResourceGroupName:   infra.ResourceGroup.Name,
		PrivateZoneName:     openaiZone.Name,
		Location:            pulumi.String("global"),
		RegistrationEnabled: pulumi.Bool(false),
		VirtualNetwork:      &privatedns.SubResourceArgs{Id: infra.Networking.VNet.ID().ToStringOutput()},
	}, openaiZoneOpts...); err != nil {
		return fmt.Errorf("linking openai private DNS zone to VNet: %w", err)
	}

	if _, err := network.NewPrivateDnsZoneGroup(ctx, peName, &network.PrivateDnsZoneGroupArgs{
		ResourceGroupName:       infra.ResourceGroup.Name,
		PrivateEndpointName:     pe.Name,
		PrivateDnsZoneGroupName: pulumi.String("default"),
		PrivateDnsZoneConfigs: network.PrivateDnsZoneConfigArray{
			&network.PrivateDnsZoneConfigArgs{
				Name:             pulumi.String("cognitiveservices"),
				PrivateDnsZoneId: csZone.ID().ToStringOutput(),
			},
			&network.PrivateDnsZoneConfigArgs{
				Name:             pulumi.String("openai"),
				PrivateDnsZoneId: openaiZone.ID().ToStringOutput(),
			},
		},
	}, append(opts, pulumi.Parent(pe))...); err != nil {
		return fmt.Errorf("creating LLM private DNS zone group: %w", err)
	}
	return nil
}

// createLLMPrivatelinkZone creates a privatelink private DNS zone for one of
// the AIServices data planes. Returns the zone plus a child opts slice with
// the zone as parent, suitable for use creating the VNet link.
func createLLMPrivatelinkZone(
	ctx *pulumi.Context,
	logicalName, zoneName string,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*privatedns.PrivateZone, []pulumi.ResourceOption, error) {
	zone, err := privatedns.NewPrivateZone(ctx, logicalName, &privatedns.PrivateZoneArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		Location:          pulumi.String("global"),
		PrivateZoneName:   pulumi.String(zoneName),
		Tags:              ServiceTags("llm"),
	}, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("creating %s private DNS zone: %w", zoneName, err)
	}
	return zone, append(opts, pulumi.Parent(zone)), nil
}
