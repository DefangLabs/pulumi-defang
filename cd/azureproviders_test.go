package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
)

type fakeProviders struct {
	mu sync.Mutex
	// state per namespace; namespaces not in the map are "NotRegistered".
	state map[string]string
	// getsUntilRegistered: number of Gets after Register before the state flips.
	getsUntilRegistered int
	pending             map[string]int
	registerErr         error
	registered          []string
}

func (f *fakeProviders) Get(_ context.Context, ns string, _ *armresources.ProvidersClientGetOptions) (armresources.ProvidersClientGetResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if n, ok := f.pending[ns]; ok {
		if n <= 0 {
			f.state[ns] = azureRegisteredState
			delete(f.pending, ns)
		} else {
			f.pending[ns] = n - 1
		}
	}
	state, ok := f.state[ns]
	if !ok {
		state = "NotRegistered"
	}
	return armresources.ProvidersClientGetResponse{Provider: armresources.Provider{Namespace: &ns, RegistrationState: &state}}, nil
}

func (f *fakeProviders) Register(_ context.Context, ns string, _ *armresources.ProvidersClientRegisterOptions) (armresources.ProvidersClientRegisterResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.registerErr != nil {
		return armresources.ProvidersClientRegisterResponse{}, f.registerErr
	}
	f.registered = append(f.registered, ns)
	f.state[ns] = "Registering"
	f.pending[ns] = f.getsUntilRegistered
	return armresources.ProvidersClientRegisterResponse{}, nil
}

func newFakeProviders(state map[string]string) *fakeProviders {
	return &fakeProviders{state: state, pending: map[string]int{}}
}

func TestEnsureAzureResourceProvidersSkipsRegistered(t *testing.T) {
	f := newFakeProviders(map[string]string{"Microsoft.App": "Registered", "Microsoft.Insights": "registered"})
	ensureAzureResourceProviders(context.Background(), f, []string{"Microsoft.App", "Microsoft.Insights"}, time.Second, time.Millisecond)
	if len(f.registered) != 0 {
		t.Errorf("expected no Register calls, got %v", f.registered)
	}
}

func TestEnsureAzureResourceProvidersRegistersAndWaits(t *testing.T) {
	f := newFakeProviders(map[string]string{"Microsoft.App": "Registered"})
	f.getsUntilRegistered = 2
	ensureAzureResourceProviders(context.Background(), f, []string{"Microsoft.App", "Microsoft.Insights"}, time.Second, time.Millisecond)
	if len(f.registered) != 1 || f.registered[0] != "Microsoft.Insights" {
		t.Errorf("expected only Microsoft.Insights to be registered, got %v", f.registered)
	}
	if f.state["Microsoft.Insights"] != azureRegisteredState {
		t.Errorf("expected Microsoft.Insights to end Registered, got %q", f.state["Microsoft.Insights"])
	}
}

func TestEnsureAzureResourceProviderWaitsWhenAlreadyRegistering(t *testing.T) {
	f := newFakeProviders(map[string]string{"Microsoft.Insights": "Registering"})
	f.pending["Microsoft.Insights"] = 1
	if err := ensureAzureResourceProvider(context.Background(), f, "Microsoft.Insights", time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if len(f.registered) != 0 {
		t.Errorf("expected no Register call while already Registering, got %v", f.registered)
	}
}

func TestEnsureAzureResourceProviderTimesOut(t *testing.T) {
	f := newFakeProviders(map[string]string{})
	f.getsUntilRegistered = 1 << 30 // never completes
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := ensureAzureResourceProvider(ctx, f, "Microsoft.Insights", time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected deadline exceeded, got %v", err)
	}
}

func TestEnsureAzureResourceProviderForbidden(t *testing.T) {
	f := newFakeProviders(map[string]string{})
	f.registerErr = &azcore.ResponseError{StatusCode: http.StatusForbidden, ErrorCode: "AuthorizationFailed"}
	err := ensureAzureResourceProvider(context.Background(), f, "Microsoft.Insights", time.Millisecond)
	if err == nil {
		t.Fatal("expected an error")
	}
	if want := "Microsoft.Insights/register/action"; !regexp.MustCompile(regexp.QuoteMeta(want)).MatchString(err.Error()) {
		t.Errorf("expected error to mention %q, got %v", want, err)
	}
	// Failures are non-fatal for the batch helper.
	ensureAzureResourceProviders(context.Background(), f, []string{"Microsoft.Insights"}, time.Second, time.Millisecond)
}

func TestAzureResourceProviderNamespaces(t *testing.T) {
	got := azureResourceProviderNamespaces()
	seen := map[string]bool{}
	for i, ns := range got {
		if ns == "" || seen[ns] {
			t.Errorf("unexpected empty or duplicate namespace %q in %v", ns, got)
		}
		seen[ns] = true
		if i > 0 && got[i-1] > ns {
			t.Errorf("namespaces not sorted: %v", got)
		}
	}
	for _, want := range []string{"Microsoft.Insights", "Microsoft.App", "Microsoft.OperationalInsights", "Microsoft.CognitiveServices", "Microsoft.ContainerRegistry"} {
		if !seen[want] {
			t.Errorf("missing %s in %v", want, got)
		}
	}
}

// Every pulumi-azure-native-sdk module that the provider (root go.mod) or the
// CD program (cd/go.mod) depends on must map to a resource provider namespace,
// so a newly used Azure service is not forgotten here.
func TestAzureNamespacesCoverImportedModules(t *testing.T) {
	re := regexp.MustCompile(`github\.com/pulumi/pulumi-azure-native-sdk/([a-z]+)/v\d+ `)
	for _, gomod := range []string{"../go.mod", "go.mod"} {
		data, err := os.ReadFile(gomod)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range re.FindAllStringSubmatch(string(data), -1) {
			if _, ok := azureNativeModuleNamespaces[m[1]]; !ok {
				t.Errorf("%s: pulumi-azure-native-sdk/%s has no entry in azureNativeModuleNamespaces", gomod, m[1])
			}
		}
	}
}
