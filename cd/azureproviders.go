package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
)

// azureNativeModuleNamespaces maps each pulumi-azure-native-sdk module the
// Azure provider or the CD program imports to the ARM resource provider
// namespace its resources live in. An empty namespace means the module needs
// no registration (Microsoft.Authorization and Microsoft.Resources are always
// registered). TestAzureNamespacesCoverImportedModules keeps this in sync with
// go.mod.
var azureNativeModuleNamespaces = map[string]string{
	"app":                 "Microsoft.App",
	"authorization":       "",
	"cdn":                 "Microsoft.Cdn",
	"cognitiveservices":   "Microsoft.CognitiveServices",
	"compute":             "Microsoft.Compute",
	"containerregistry":   "Microsoft.ContainerRegistry",
	"dbforpostgresql":     "Microsoft.DBforPostgreSQL",
	"dns":                 "Microsoft.Network",
	"keyvault":            "Microsoft.KeyVault",
	"managedidentity":     "Microsoft.ManagedIdentity",
	"monitor":             "Microsoft.Insights", // DiagnosticSetting
	"network":             "Microsoft.Network",
	"operationalinsights": "Microsoft.OperationalInsights",
	"privatedns":          "Microsoft.Network",
	"redisenterprise":     "Microsoft.Cache",
	"resources":           "",
	"storage":             "Microsoft.Storage",
}

// azureResourceProviderNamespaces returns the sorted, de-duplicated namespaces
// that must be registered on the subscription before deploying.
func azureResourceProviderNamespaces() []string {
	seen := map[string]bool{}
	var namespaces []string
	for _, ns := range azureNativeModuleNamespaces {
		if ns != "" && !seen[ns] {
			seen[ns] = true
			namespaces = append(namespaces, ns)
		}
	}
	sort.Strings(namespaces)
	return namespaces
}

// providersAPI is the subset of armresources.ProvidersClient we use.
type providersAPI interface {
	Get(
		ctx context.Context, namespace string, options *armresources.ProvidersClientGetOptions,
	) (armresources.ProvidersClientGetResponse, error)
	Register(
		ctx context.Context, namespace string, options *armresources.ProvidersClientRegisterOptions,
	) (armresources.ProvidersClientRegisterResponse, error)
}

const (
	azureRegisteredState      = "Registered"
	azureRegistrationTimeout  = 5 * time.Minute
	azureRegistrationInterval = 5 * time.Second
)

// registerAzureResourceProviders registers every resource provider namespace
// the deployment may use, so ARM does not reject resources (notably extension
// resources like Microsoft.Insights diagnostic settings, which ARM does not
// auto-register) with SubscriptionNotRegistered. Namespaces that are already
// registered are left alone. Failures are logged and do not abort the deploy:
// if a namespace really is required, Pulumi reports the ARM error itself.
func registerAzureResourceProviders(ctx context.Context, subscriptionID string) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		warn("Warning: skipping Azure resource provider registration:", err)
		return
	}
	client, err := armresources.NewProvidersClient(subscriptionID, cred, nil)
	if err != nil {
		warn("Warning: skipping Azure resource provider registration:", err)
		return
	}
	ensureAzureResourceProviders(ctx, client, azureResourceProviderNamespaces(),
		azureRegistrationTimeout, azureRegistrationInterval)
}

func ensureAzureResourceProviders(
	ctx context.Context, client providersAPI, namespaces []string, timeout, interval time.Duration,
) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var wg sync.WaitGroup
	for _, ns := range namespaces {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := ensureAzureResourceProvider(ctx, client, ns, interval); err != nil {
				warn(fmt.Sprintf("Warning: could not register Azure resource provider %s: %v", ns, err))
			}
		}()
	}
	wg.Wait()
}

func ensureAzureResourceProvider(
	ctx context.Context, client providersAPI, namespace string, interval time.Duration,
) error {
	state, err := azureProviderState(ctx, client, namespace)
	if err != nil {
		return err
	}
	if strings.EqualFold(state, azureRegisteredState) {
		return nil
	}
	// "Registering" means somebody else already started it; just wait.
	if !strings.EqualFold(state, "Registering") {
		Println("Registering Azure resource provider", namespace, "on the subscription")
		if _, err := client.Register(ctx, namespace, nil); err != nil {
			return describeRegistrationError(namespace, err)
		}
	}
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("still %q after waiting: %w", state, ctx.Err())
		case <-time.After(interval):
		}
		if state, err = azureProviderState(ctx, client, namespace); err != nil {
			return err
		}
		if strings.EqualFold(state, azureRegisteredState) {
			return nil
		}
	}
}

func azureProviderState(ctx context.Context, client providersAPI, namespace string) (string, error) {
	resp, err := client.Get(ctx, namespace, nil)
	if err != nil {
		return "", err
	}
	if resp.RegistrationState == nil {
		return "", nil
	}
	return *resp.RegistrationState, nil
}

func describeRegistrationError(namespace string, err error) error {
	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
		return fmt.Errorf("the CD identity lacks %s/register/action on the subscription; "+
			"register it once with `az provider register --namespace %s`: %w", namespace, namespace, err)
	}
	return err
}
