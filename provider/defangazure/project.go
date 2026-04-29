package defangazure

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providerazure "github.com/DefangLabs/pulumi-defang/provider/defangazure/azure"
	"github.com/pulumi/pulumi-azure-native-sdk/app/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/operationalinsights/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var ErrLLMInfraNotCreated = errors.New("LLM infra was not created")

// serviceComponent is a generic Pulumi component resource for a single service.
type serviceComponent struct{ pulumi.ResourceState }

// Project is the controller struct for the defang-azure:index:Project component.
type Project struct{}

// ProjectInputs defines the top-level inputs for the Azure Project component.
type ProjectInputs struct {
	// Services map: name -> service config
	Services compose.Services `pulumi:"services"          yaml:"services"`
	Networks compose.Networks `pulumi:"networks,optional" yaml:"networks,omitempty"`
	// Etag is the deployment identifier supplied by the CD program; the provider
	// stamps it onto every resource (tags) and onto Container App revisions
	// (RevisionSuffix) so that logs and Resource Graph queries can be filtered
	// per-deployment.
	Etag string `pulumi:"etag,optional" yaml:"etag,omitempty"`
}

// ProjectOutputs holds the outputs of the Project component.
type ProjectOutputs struct {
	pulumi.ResourceState

	// Per-service endpoint URLs (service name -> URL)
	Endpoints pulumi.StringMapOutput `pulumi:"endpoints"`

	// Load balancer DNS name (unused for Azure, kept for interface compat)
	LoadBalancerDNS pulumi.StringPtrOutput `pulumi:"loadBalancerDns,optional"`
}

// serviceTypes summarises which kinds of services are present in a project.
// pgServiceName / redisServiceName hold the first (alphabetically) service of each
// kind, used as Pulumi logical names for project-shared DNS zones. Empty when
// that kind has no services.
type serviceTypes struct {
	hasBuild         bool
	hasLLM           bool
	pgServiceName    string
	redisServiceName string
	llmModels        map[string]string // LLM service name → model alias
}

// detectServiceTypes scans all services and summarises feature flags, per-kind
// service names, and an LLM model alias map.
// llmModels maps each LLM service name to its model alias (e.g. "llm" → "chat-default").
// The CLI injects {UPPER(svcName)}_MODEL into dependent services; we reverse-look that up here.
func detectServiceTypes(services compose.Services) serviceTypes {
	result := serviceTypes{llmModels: make(map[string]string)}

	// Sort service names so the "first" pg/redis pick is deterministic across runs.
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, svcName := range names {
		svc := services[svcName]
		if svc.Postgres != nil && result.pgServiceName == "" {
			result.pgServiceName = svcName
		}
		if svc.Redis != nil && result.redisServiceName == "" {
			result.redisServiceName = svcName
		}
		if svc.Build != nil {
			result.hasBuild = true
		}
		if svc.LLM != nil {
			result.hasLLM = true
			result.llmModels[svcName] = llmModelAlias(svcName, services)
		}
	}
	return result
}

// withResourceDeps returns val as a StringOutput whose Pulumi dependency graph
// also includes the URNs of deps. Downstream resources that read this output
// (e.g. via ApplyT or as a field input) transitively DependOn the deps —
// without the caller having to thread explicit DependsOn options through.
// Useful for attaching invisible readiness signals (e.g. server parameter
// configurations) to an endpoint string that apps consume.
func withResourceDeps(val pulumi.StringOutput, deps []pulumi.Resource) pulumi.StringOutput {
	if len(deps) == 0 {
		return val
	}
	ins := make([]interface{}, 0, len(deps)+1)
	ins = append(ins, val)
	for _, d := range deps {
		ins = append(ins, d.URN())
	}
	return pulumi.All(ins...).ApplyT(func(parts []interface{}) string {
		s, _ := parts[0].(string)
		return s
	}).(pulumi.StringOutput)
}

func createPostgresResources(
	ctx *pulumi.Context,
	svcName string,
	svc compose.ServiceConfig,
	infra *providerazure.SharedInfra,
	managedEndpoints map[string]pulumi.StringOutput,
	serviceHosts map[string]pulumi.StringOutput,
	comp *serviceComponent,
	childOpts []pulumi.ResourceOption,
) (pulumi.StringOutput, error) {
	if err := ctx.RegisterComponentResource("defang-azure:index:Postgres", svcName, comp, childOpts...); err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("registering Azure Postgres component %s: %w", svcName, err)
	}
	svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

	pgResult, err := providerazure.CreatePostgresFlexible(ctx, infra.ConfigProvider, svcName, svc, infra, svcOpts...)
	if err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("creating PostgreSQL for %s: %w", svcName, err)
	}

	// Downstream Container Apps read managedEndpoints/serviceHosts to build env
	// vars (e.g. DATABASE_URL). The Server's FQDN is available as soon as the
	// server exists, but running migrations that rely on server parameters
	// (`azure.extensions=VECTOR`, `require_secure_transport=OFF`) requires the
	// Configurations to be applied first. Thread those through as hidden deps
	// on the outputs apps consume, so Pulumi won't start the apps until the
	// server is actually ready for queries that need those settings.
	fqdn := withResourceDeps(pgResult.Server.FullyQualifiedDomainName, pgResult.Readiness)
	endpoint := pulumi.Sprintf("%s:5432", fqdn)
	managedEndpoints[svcName] = endpoint
	serviceHosts[svcName] = fqdn
	return endpoint, nil
}

func createRedisResources(
	ctx *pulumi.Context,
	svcName string,
	svc compose.ServiceConfig,
	infra *providerazure.SharedInfra,
	managedEndpoints map[string]pulumi.StringOutput,
	serviceHosts map[string]pulumi.StringOutput,
	comp *serviceComponent,
	childOpts []pulumi.ResourceOption,
) (pulumi.StringOutput, error) {
	if err := ctx.RegisterComponentResource("defang-azure:index:Redis", svcName, comp, childOpts...); err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("registering Azure Redis component %s: %w", svcName, err)
	}
	svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

	redisResult, err := providerazure.CreateRedisEnterprise(ctx, svcName, svc, infra, svcOpts...)
	if err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("creating Redis for %s: %w", svcName, err)
	}

	// ConnectionURL is the full redis:// or rediss:// URL including auth.
	managedEndpoints[svcName] = redisResult.ConnectionURL
	serviceHosts[svcName] = redisResult.Cluster.HostName
	return pulumi.Sprintf("%s:10000", redisResult.Cluster.HostName), nil
}

// createServiceResources creates the component resource and underlying cloud resources
// for a single service. It updates managedEndpoints in-place for managed services and
// returns the service endpoint.
func createServiceResources(
	ctx *pulumi.Context,
	svcName string,
	svc compose.ServiceConfig,
	infra *providerazure.SharedInfra,
	managedEndpoints map[string]pulumi.StringOutput,
	serviceHosts map[string]pulumi.StringOutput,
	llmModels map[string]string,
	childOpts []pulumi.ResourceOption,
) (pulumi.StringOutput, error) {
	comp := &serviceComponent{}
	var endpoint pulumi.StringOutput

	switch {
	case svc.Postgres != nil:
		var err error
		endpoint, err = createPostgresResources(ctx, svcName, svc, infra, managedEndpoints, serviceHosts, comp, childOpts)
		if err != nil {
			return pulumi.StringOutput{}, err
		}

	case svc.Redis != nil:
		var err error
		endpoint, err = createRedisResources(ctx, svcName, svc, infra, managedEndpoints, serviceHosts, comp, childOpts)
		if err != nil {
			return pulumi.StringOutput{}, err
		}

	case svc.LLM != nil:
		if infra.LLMInfra == nil {
			return pulumi.StringOutput{}, fmt.Errorf("service %s: %w", svcName, ErrLLMInfraNotCreated)
		}
		if err := ctx.RegisterComponentResource("defang-azure:index:LLM", svcName, comp, childOpts...); err != nil {
			return pulumi.StringOutput{}, fmt.Errorf("registering Azure LLM component %s: %w", svcName, err)
		}
		svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

		modelAlias := llmModels[svcName]
		if err := providerazure.CreateLLMDeployment(
			ctx, modelAlias, modelAlias, infra.LLMInfra, infra, svcOpts...,
		); err != nil {
			return pulumi.StringOutput{}, fmt.Errorf("creating LLM deployment for %s: %w", svcName, err)
		}

		managedEndpoints[svcName] = infra.LLMInfra.BaseURL
		endpoint = infra.LLMInfra.BaseURL

	default:
		if err := ctx.RegisterComponentResource(
			"defang-azure:index:AzureContainerApp", svcName, comp, childOpts...,
		); err != nil {
			return pulumi.StringOutput{}, fmt.Errorf("registering Container App component %s: %w", svcName, err)
		}
		svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

		imageURI, err := providerazure.GetServiceImage(ctx, svcName, svc, infra.BuildInfra, infra, svcOpts...)
		if err != nil {
			return pulumi.StringOutput{}, fmt.Errorf("resolving image for %s: %w", svcName, err)
		}

		caResult, err := providerazure.CreateContainerApp(
			ctx, svcName, svc, infra, imageURI, managedEndpoints, serviceHosts, svcOpts...,
		)
		if err != nil {
			return pulumi.StringOutput{}, fmt.Errorf("creating Container App %s: %w", svcName, err)
		}
		endpoint = caResult.App.LatestRevisionFqdn.ApplyT(fqdnToHTTPS).(pulumi.StringOutput)
	}

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": endpoint,
	}); err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("registering outputs for %s: %w", svcName, err)
	}

	return endpoint, nil
}

// createProjectResourceGroup creates (or imports) the project's resource group.
func createProjectResourceGroup(
	ctx *pulumi.Context,
	name, location string,
	childOpts []pulumi.ResourceOption,
) (*resources.ResourceGroup, error) {
	rgArgs := &resources.ResourceGroupArgs{
		Location: pulumi.String(location),
	}
	rgOpts := childOpts
	if existingRG := providerazure.ExistingResourceGroup(ctx, name); existingRG != "" {
		// Import the existing RG: ResourceGroupName must match so Pulumi doesn't
		// propose a replacement on subsequent refreshes.
		rgArgs.ResourceGroupName = pulumi.String(existingRG)
		subID := providerazure.SubscriptionID(ctx)
		rgID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subID, existingRG)
		rgOpts = append(rgOpts, pulumi.Import(pulumi.ID(rgID)))
	}
	rg, err := resources.NewResourceGroup(ctx, name, rgArgs, rgOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating resource group: %w", err)
	}
	return rg, nil
}

// createManagedEnvironment provisions the Log Analytics workspace and Container
// App managed environment, attaching VNet config when networking is present.
// parentOpt carries the parent (and its provider) for the GetSharedKeys invoke,
// which ResourceOption slices can't express because InvokeOption is a sibling
// interface to ResourceOption.
func createManagedEnvironment(
	ctx *pulumi.Context,
	name, location string,
	infra *providerazure.SharedInfra,
	parentOpt pulumi.ResourceOrInvokeOption,
	childOpts []pulumi.ResourceOption,
) (*app.ManagedEnvironment, error) {
	logWorkspace, err := operationalinsights.NewWorkspace(ctx, name, &operationalinsights.WorkspaceArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		Location:          pulumi.String(location),
		Sku: &operationalinsights.WorkspaceSkuArgs{
			Name: pulumi.String("PerGB2018"),
		},
		RetentionInDays: pulumi.Int(30),
	}, childOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating Log Analytics workspace: %w", err)
	}
	logKeys := operationalinsights.GetSharedKeysOutput(ctx, operationalinsights.GetSharedKeysOutputArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		WorkspaceName:     logWorkspace.Name,
	}, parentOpt)

	envArgs := &app.ManagedEnvironmentArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		Location:          pulumi.String(location),
		AppLogsConfiguration: &app.AppLogsConfigurationArgs{
			Destination: pulumi.String("log-analytics"),
			LogAnalyticsConfiguration: &app.LogAnalyticsConfigurationArgs{
				CustomerId: logWorkspace.CustomerId,
				SharedKey:  logKeys.PrimarySharedKey(),
			},
		},
	}
	if infra.Networking != nil {
		envArgs.VnetConfiguration = &app.VnetConfigurationArgs{
			InfrastructureSubnetId: infra.Networking.AppsSubnet.ID().ToStringOutput(),
		}
	}

	// VnetConfiguration is immutable on Azure — adding/changing it requires replacement.
	envOpts := append([]pulumi.ResourceOption{pulumi.ReplaceOnChanges([]string{"vnetConfiguration"})}, childOpts...)
	env, err := app.NewManagedEnvironment(ctx, name, envArgs, envOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating managed environment: %w", err)
	}
	return env, nil
}

// setupSharedInfra creates the resource group and all shared infrastructure
// (networking, DNS, log analytics, managed environment, LLM/build infra, KV
// identity) used by the per-service resources.
func setupSharedInfra(
	ctx *pulumi.Context,
	name string,
	inputs ProjectInputs,
	parentOpt pulumi.ResourceOrInvokeOption,
	childOpts []pulumi.ResourceOption,
) (*providerazure.SharedInfra, map[string]string, error) {
	location := providerazure.Location(ctx)

	rg, err := createProjectResourceGroup(ctx, name, location, childOpts)
	if err != nil {
		return nil, nil, err
	}

	types := detectServiceTypes(inputs.Services)

	// Compute keyVaultURL up front so the ConfigProvider can assemble
	// ready-to-use secret URLs AND lazy-fetch user config on first access.
	// Empty when no vault is configured, in which case fetch + secret refs are
	// both disabled.
	var keyVaultURL string
	if kvName := providerazure.KeyVaultName(ctx, name); kvName != "" {
		keyVaultURL = "https://" + kvName + ".vault.azure.net"
	}

	infra := &providerazure.SharedInfra{
		ResourceGroup:  rg,
		KeyVaultURL:    keyVaultURL,
		ConfigProvider: providerazure.NewConfigProvider(name, keyVaultURL),
		Etag:           inputs.Etag,
	}

	if types.pgServiceName != "" || types.redisServiceName != "" {
		networking, err := providerazure.CreateNetworking(ctx, name, infra, location, childOpts...)
		if err != nil {
			return nil, nil, fmt.Errorf("creating networking: %w", err)
		}
		infra.Networking = networking

		dns, err := providerazure.CreateDNSZones(
			ctx, name, types.pgServiceName, types.redisServiceName, infra, networking, childOpts...,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("creating DNS zones: %w", err)
		}
		infra.DNS = dns
	}

	env, err := createManagedEnvironment(ctx, name, location, infra, parentOpt, childOpts)
	if err != nil {
		return nil, nil, err
	}
	infra.Environment = env

	if types.hasLLM {
		llmInfra, err := providerazure.CreateLLMInfra(ctx, name, infra, childOpts...)
		if err != nil {
			return nil, nil, fmt.Errorf("creating LLM infra: %w", err)
		}
		infra.LLMInfra = llmInfra
	}

	if types.hasBuild {
		buildInfra, err := providerazure.CreateBuildInfra(ctx, name, infra, location, childOpts...)
		if err != nil {
			return nil, nil, fmt.Errorf("creating build infrastructure: %w", err)
		}
		infra.BuildInfra = buildInfra
	}

	if kvName := providerazure.KeyVaultName(ctx, name); kvName != "" {
		kvIdentityID, err := providerazure.CreateKeyVaultIdentity(ctx, kvName, infra, location, childOpts...)
		if err != nil {
			return nil, nil, fmt.Errorf("creating Key Vault identity: %w", err)
		}
		infra.KeyVaultIdentityID = kvIdentityID
	}

	return infra, types.llmModels, nil
}

// Construct implements the ComponentResource interface for Project.
func (*Project) Construct(
	ctx *pulumi.Context, name, typ string, inputs ProjectInputs, opts pulumi.ResourceOption,
) (*ProjectOutputs, error) {
	// Cascade a transformation to all child resources that injects
	// defang-project / defang-stack / defang-etag into every azure-native
	// resource's Tags. azure-native has no DefaultTags, and pulumi-go-provider's
	// Construct ctx lacks a stack so RegisterStackTransformation panics — the
	// resource-level Transformations option is the supported cascade.
	tagOpts := opts
	if t := providerazure.DefaultTagsTransformation(providerazure.BaseTags(ctx, inputs.Etag)); t != nil {
		tagOpts = pulumi.Composite(opts, pulumi.Transformations([]pulumi.ResourceTransformation{t}))
	}

	comp := &ProjectOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, tagOpts); err != nil {
		return nil, err
	}

	// parentOpt retains the ResourceOrInvokeOption form so it can flow into
	// Pulumi invokes (data-source lookups), where the ResourceOption slice can't.
	parentOpt := pulumi.Parent(comp)
	childOpts := []pulumi.ResourceOption{parentOpt}

	infra, llmModels, err := setupSharedInfra(ctx, name, inputs, parentOpt, childOpts)
	if err != nil {
		return nil, err
	}

	endpoints := pulumi.StringMap{}

	// managedEndpoints accumulates connection URLs for managed services (Postgres, Redis, LLM).
	managedEndpoints := make(map[string]pulumi.StringOutput, len(inputs.Services))

	// serviceHosts maps managed-service Compose names (Postgres, Redis) to their bare
	// hostname (e.g. "srv123.postgres.database.azure.com"). Used by buildEnvVars to
	// rewrite env values that reference a service by its Compose name.
	serviceHosts := make(map[string]pulumi.StringOutput, len(inputs.Services))

	// Two-pass creation: managed services first (Postgres, Redis, LLM) to populate
	// managedEndpoints/serviceHosts, then container apps that reference them. Go maps
	// iterate in random order, so a single pass can process container apps before
	// managed services, leaving endpoints unresolved.
	for svcName, svc := range inputs.Services {
		if svc.Postgres == nil && svc.Redis == nil && svc.LLM == nil {
			continue
		}
		endpoint, err := createServiceResources(
			ctx, svcName, svc, infra, managedEndpoints, serviceHosts, llmModels, childOpts,
		)
		if err != nil {
			return nil, err
		}
		endpoints[svcName] = endpoint
	}
	for svcName, svc := range inputs.Services {
		if svc.Postgres != nil || svc.Redis != nil || svc.LLM != nil {
			continue
		}
		endpoint, err := createServiceResources(
			ctx, svcName, svc, infra, managedEndpoints, serviceHosts, llmModels, childOpts,
		)
		if err != nil {
			return nil, err
		}
		endpoints[svcName] = endpoint
	}

	loadBalancerDNS := pulumi.StringPtr("").ToStringPtrOutput()

	comp.Endpoints = endpoints.ToStringMapOutput()
	comp.LoadBalancerDNS = loadBalancerDNS

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoints":       endpoints.ToStringMapOutput(),
		"loadBalancerDns": loadBalancerDNS,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}

// llmModelAlias finds the model alias for an LLM service by scanning dependent services'
// environments. The CLI injects {UPPER(svcName)}_MODEL into every service that depends on
// the LLM service. Returns the alias (e.g. "chat-default") or svcName as a fallback.
func llmModelAlias(svcName string, services compose.Services) string {
	envKey := strings.ToUpper(svcName) + "_MODEL"
	for _, svc := range services {
		if v, ok := svc.Environment[envKey]; ok && v != nil && *v != "" {
			return *v
		}
	}
	return svcName // fallback: use service name as deployment name
}

// fqdnToHTTPS converts a Container App FQDN to an https:// URL, or returns "" for empty FQDNs.
// Used as an ApplyT callback for LatestRevisionFqdn.
func fqdnToHTTPS(fqdn string) string {
	if fqdn == "" {
		return ""
	}
	return "https://" + fqdn
}
