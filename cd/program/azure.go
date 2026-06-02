package program

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/DefangLabs/defang/src/pkg/clouds/azure/aca"
	"github.com/DefangLabs/defang/src/pkg/dns"
	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providerazure "github.com/DefangLabs/pulumi-defang/provider/defangazure/azure"
	defangazure "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-azure"
	azurecompose "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-azure/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/storage/v3"
	pulumiazure "github.com/pulumi/pulumi-azure-native-sdk/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/v3/config"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"google.golang.org/protobuf/proto"
)

func deployAzure(ctx *pulumi.Context, cf *compose.Project, domain, etag string, projectUpdate *defangv1.ProjectUpdate) (pulumi.StringMapOutput, pulumi.StringPtrOutput, error) {
	providerArgs := &pulumiazure.ProviderArgs{
		Location:                  pulumi.String(config.GetLocation(ctx)),
		UseDefaultAzureCredential: pulumi.BoolPtr(true),
	}
	if subID := config.GetSubscriptionId(ctx); subID != "" {
		providerArgs.SubscriptionId = pulumi.StringPtr(subID)
	}
	azureProvider, err := pulumiazure.NewProvider(ctx, "azure", providerArgs)
	if err != nil {
		return pulumi.StringMapOutput{}, pulumi.StringPtrOutput{}, err
	}

	project, err := defangazure.NewProject(ctx, cf.Name, toAzureArgs(cf, domain, etag), pulumi.Providers(azureProvider))
	if err != nil {
		return pulumi.StringMapOutput{}, pulumi.StringPtrOutput{}, err
	}

	// Provision delegate-domain TLS certs as part of the CD task itself —
	// not the CLI — so the deploy converges even if the user disconnects
	// after `defang compose up`. The work is chained off project.Endpoints
	// (which transitively depends on every per-service Container App + its
	// CNAME + asuid TXT records) so Pulumi sequences it after all required
	// resources exist, and waits on it before declaring the deploy done.
	// Each per-service call is idempotent: re-deploys are cheap, partial
	// failures pick up where they left off.
	if domain != "" {
		rg := providerazure.ProjectResourceGroupName(ctx, cf.Name)
		certsDone := project.Endpoints.ApplyT(func(map[string]string) (string, error) {
			provisionDelegateDomainCerts(ctx.Context(), cf, domain, config.GetSubscriptionId(ctx), rg)
			return "", nil
		}).(pulumi.StringOutput)
		// Export so Pulumi treats it as a stack output and won't garbage-collect
		// the ApplyT before its side effects complete. The value is empty by
		// design — we only care about the dependency edge.
		ctx.Export("azureDelegateDomainCerts", certsDone)
	}

	// Upload ProjectUpdate protobuf as a Pulumi-managed Azure Blob, gated on
	// the project component so it only runs after all services are created.
	// pulumi.Provider(azureProvider) is required because
	// pulumi:disable-default-providers excludes azure-native (see cd/main.go projectConfig).
	if projectUpdate != nil {
		updatedPb := project.Endpoints.ApplyT(func(endpoints map[string]string) ([]byte, error) {
			for _, svc := range projectUpdate.Services {
				svc.Status = "PROVISIONING"
				svc.State = defangv1.ServiceState_DEPLOYMENT_COMPLETED
				if ep, ok := endpoints[svc.GetService().GetName()]; ok {
					svc.Endpoints = []string{ep} // FIXME: support multiple endpoints per service
					if u, err := url.Parse(ep); err == nil && u.Host != "" {
						svc.PublicFqdn = u.Hostname() // FIXME: support private FQDNs
					}
				}
			}
			return proto.Marshal(projectUpdate)
		}).(pulumi.AnyOutput)

		if err := saveProjectPbAzure(ctx, updatedPb, project, pulumi.Provider(azureProvider)); err != nil {
			return pulumi.StringMapOutput{}, pulumi.StringPtrOutput{}, err
		}
	}
	return project.Endpoints, project.LoadBalancerDns, nil
}

// provisionDelegateDomainCerts walks the compose project's ingress services
// and asks Azure to issue + bind a managed cert for each
// `<service>.<domain>` hostname. Records (CNAME + asuid TXT) are already in
// Azure DNS from the Pulumi-managed RecordSets, so aca.IssueCert's "wait for
// DNS" step passes immediately and the flow proceeds straight to registering
// the customDomain on the Container App, issuing the cert via CNAME
// validation, and re-binding with SniEnabled.
//
// Failures are logged but not surfaced as Pulumi errors: the Container App is
// already serving on its `*.azurecontainerapps.io` URL by this point, the
// cert flow is idempotent, and the next deploy will retry. A hard error
// would force a Pulumi destroy/replace cycle that doesn't actually fix
// anything cert-side.
// perServiceCertTimeout caps how long any single service's IssueCert call can
// hang. aca.IssueCert bounds its DNS/TXT/TLS polling loops internally
// (dnsWaitTimeout=30m, tokenDeadline=5m, tlsWaitTimeout=10m), but the ARM
// long-running operations (addHostnameDisabled, submitManagedCert,
// bindHostnameSniEnabled) run `poller.PollUntilDone(ctx, nil)` and have no
// deadline beyond this ctx — so a throttled or busy ARM PATCH would otherwise
// block the Pulumi run until the outer timeout fires. 45m fits the documented
// worst-case sum and still trips faster than a stuck poll.
const perServiceCertTimeout = 45 * time.Minute

func provisionDelegateDomainCerts(ctx context.Context, cf *compose.Project, domain, subscriptionID, resourceGroup string) {
	if subscriptionID == "" {
		// deployAzure forwards config.GetSubscriptionId(ctx) unconditionally;
		// when Pulumi config doesn't carry it, the ARM SDK clients we'd
		// construct below fail with an opaque URL parse error. Surface that
		// early — the Container App is already serving on its
		// azurecontainerapps.io URL; cert binding is a separate concern the
		// next deploy can retry.
		log.Print("delegate-domain cert: skipping; AZURE_SUBSCRIPTION_ID not set in Pulumi config")
		return
	}
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Printf("delegate-domain cert: build credential: %v", err)
		return
	}
	for name, svc := range cf.Services {
		if !svc.HasIngressPorts() {
			continue
		}
		hostname := name + "." + domain
		log.Printf("Issuing delegate-domain cert for %s at %s", name, hostname)
		svcCtx, cancel := context.WithTimeout(ctx, perServiceCertTimeout)
		if err := aca.IssueCert(svcCtx, cred, subscriptionID, resourceGroup, name, hostname, dns.DirectResolverAt); err != nil {
			log.Printf("delegate-domain cert: issuance for %s failed: %v", hostname, err)
		}
		cancel()
	}
}

func toAzureArgs(cf *compose.Project, domain, etag string) *defangazure.ProjectArgs {
	args := &defangazure.ProjectArgs{
		Services: toAzureServices(cf.Services),
	}
	if len(cf.Networks) > 0 {
		nm := make(azurecompose.NetworkConfigMap, len(cf.Networks))
		for k, v := range cf.Networks {
			nm[string(k)] = azurecompose.NetworkConfigArgs{Internal: pulumi.Bool(v.Internal)}
		}
		args.Networks = nm
	}
	if etag != "" {
		e := etag
		args.Etag = &e
	}
	if domain != "" {
		d := domain
		args.Domain = &d
	}
	return args
}

func toAzureServices(services compose.Services) azurecompose.ServiceConfigMap {
	m := make(azurecompose.ServiceConfigMap, len(services))
	for name, svc := range services {
		m[name] = toAzureServiceArgs(svc)
	}
	return m
}

func toAzureServiceArgs(svc compose.ServiceConfig) azurecompose.ServiceConfigArgs {
	args := azurecompose.ServiceConfigArgs{
		Image:       pulumi.StringPtrFromPtr(svc.Image),
		Platform:    pulumi.StringPtrFromPtr(svc.Platform),
		Environment: pulumi.ToStringMap(svc.ResolvedEnvironment()),
		Command:     pulumi.ToStringArray(svc.Command),
		Entrypoint:  pulumi.ToStringArray(svc.Entrypoint),
	}
	if svc.DomainName != "" {
		args.DomainName = pulumi.StringPtr(svc.DomainName)
	}
	if svc.Build != nil {
		args.Build = azurecompose.BuildConfigArgs{
			Context:    svc.Build.Context,
			Dockerfile: pulumi.StringPtrFromPtr(svc.Build.Dockerfile),
			Args:       pulumi.ToStringMap(svc.Build.Args),
			ShmSize:    pulumi.StringPtrFromPtr(svc.Build.ShmSize),
			Target:     pulumi.StringPtrFromPtr(svc.Build.Target),
		}
	}
	if len(svc.Ports) > 0 {
		ports := make(azurecompose.ServicePortConfigArray, 0, len(svc.Ports))
		for _, p := range svc.Ports {
			ports = append(ports, azurecompose.ServicePortConfigArgs{
				Target:      pulumi.Int(int(p.Target)),
				Mode:        pulumi.String(p.Mode),
				Protocol:    pulumi.String(p.Protocol),
				AppProtocol: pulumi.String(p.AppProtocol),
			})
		}
		args.Ports = ports
	}
	if svc.Deploy != nil {
		da := azurecompose.DeployConfigArgs{}
		if svc.Deploy.Replicas != nil {
			da.Replicas = pulumi.IntPtr(int(*svc.Deploy.Replicas))
		}
		if svc.Deploy.Resources != nil && svc.Deploy.Resources.Reservations != nil {
			r := svc.Deploy.Resources.Reservations
			ra := azurecompose.ResourceConfigArgs{
				Cpus:   pulumi.Float64PtrFromPtr(r.CPUs),
				Memory: pulumi.StringPtrFromPtr(r.Memory),
			}
			da.Resources = azurecompose.ResourcesArgs{Reservations: ra}
		}
		args.Deploy = da
	}
	if svc.Postgres != nil {
		args.Postgres = azurecompose.PostgresConfigArgs{
			AllowDowntime: pulumi.BoolPtrFromPtr(svc.Postgres.AllowDowntime),
			FromSnapshot:  pulumi.StringPtrFromPtr(svc.Postgres.FromSnapshot),
		}
	}
	if svc.Redis != nil {
		args.Redis = azurecompose.RedisConfigArgs{
			AllowDowntime: pulumi.BoolPtrFromPtr(svc.Redis.AllowDowntime),
			FromSnapshot:  pulumi.StringPtrFromPtr(svc.Redis.FromSnapshot),
		}
	}
	if svc.HealthCheck != nil {
		args.HealthCheck = azurecompose.HealthCheckConfigArgs{
			Test:               pulumi.ToStringArray(svc.HealthCheck.Test),
			IntervalSeconds:    pulumi.IntPtr(int(svc.HealthCheck.IntervalSeconds)),
			TimeoutSeconds:     pulumi.IntPtr(int(svc.HealthCheck.TimeoutSeconds)),
			Retries:            pulumi.IntPtr(int(svc.HealthCheck.Retries)),
			StartPeriodSeconds: pulumi.IntPtr(int(svc.HealthCheck.StartPeriodSeconds)),
		}
	}
	if len(svc.Networks) > 0 {
		nm := make(azurecompose.ServiceNetworkConfigMap, len(svc.Networks))
		for k, v := range svc.Networks {
			nm[string(k)] = azurecompose.ServiceNetworkConfigArgs{Aliases: pulumi.ToStringArray(v.Aliases)}
		}
		args.Networks = nm
	}
	if len(svc.DependsOn) > 0 {
		dm := make(azurecompose.ServiceDependencyMap, len(svc.DependsOn))
		for k, v := range svc.DependsOn {
			dm[k] = azurecompose.ServiceDependencyArgs{Condition: pulumi.String(v.Condition), Required: pulumi.Bool(v.Required)}
		}
		args.DependsOn = dm
	}
	if svc.LLM != nil {
		args.Llm = azurecompose.LlmConfigArgs{}
	}
	return args
}

// saveProjectPbAzure uploads data as a Pulumi-managed Azure Blob in the CD
// storage account's `projects` container. See saveProjectPbAWS for semantics.
// data is a pulumi.AnyOutput wrapping []byte so callers can produce the bytes
// inside an ApplyT (e.g. after updating endpoints) without creating resources
// inside that callback.
func saveProjectPbAzure(ctx *pulumi.Context, data pulumi.AnyOutput, dep pulumi.Resource, opts ...pulumi.ResourceOption) error {
	u, err := parseStateURL(ctx)
	if err != nil || u == nil {
		return err
	}
	if u.Scheme != "azblob" || u.Host == "" {
		return fmt.Errorf("DEFANG_STATE_URL must be an azblob:// URL with a container for Azure uploads, got %q", u.String())
	}
	account := u.Query().Get("storage_account")
	if account == "" {
		return fmt.Errorf("DEFANG_STATE_URL %q missing storage_account", u.String())
	}

	// The CD storage account lives in the shared CD resource group, named
	// `defang-cd` (single per subscription, location-independent —
	// see defang/src/pkg/clouds/azure/cd/driver.go).
	cdRG := "defang-cd"

	source := data.ApplyT(func(v any) (pulumi.Asset, error) {
		return NewTempFileAsset("defang-cd-*-project.pb", v.([]byte))
	}).(pulumi.AssetOutput)

	_, err = storage.NewBlob(ctx, "project-pb", &storage.BlobArgs{
		ResourceGroupName: pulumi.String(cdRG),
		AccountName:       pulumi.String(account),
		ContainerName:     pulumi.String(u.Host), // Host is the container in azblob:// URLs
		BlobName:          pulumi.String(projectPbKey(ctx)),
		Source:            source,
		ContentType:       pulumi.String(protobufContentType),
	}, common.MergeOptions(opts, pulumi.DependsOn([]pulumi.Resource{dep}))...)
	return err
}
