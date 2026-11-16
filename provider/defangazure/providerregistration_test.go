package defangazure

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeProviders simulates ARM resource provider registration. A namespace
// missing from state is "NotRegistered"; after Register it turns
// "Registered" once Get has been called getsUntilRegistered more times.
type fakeProviders struct {
	mu                  sync.Mutex
	state               map[string]string
	pending             map[string]int
	getsUntilRegistered int
	registerErr         error
	registered          []string
	gets                int
}

var errBoom = errors.New("boom")

func newFakeProviders(state map[string]string) *fakeProviders {
	return &fakeProviders{state: state, pending: map[string]int{}}
}

func (f *fakeProviders) Get(
	_ context.Context, ns string, _ *armresources.ProvidersClientGetOptions,
) (armresources.ProvidersClientGetResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gets++
	if n, ok := f.pending[ns]; ok {
		if n <= 0 {
			f.state[ns] = registeredState
			delete(f.pending, ns)
		} else {
			f.pending[ns] = n - 1
		}
	}
	state, ok := f.state[ns]
	if !ok {
		state = "NotRegistered"
	}
	return armresources.ProvidersClientGetResponse{
		Provider: armresources.Provider{Namespace: &ns, RegistrationState: &state},
	}, nil
}

func (f *fakeProviders) Register(
	_ context.Context, ns string, _ *armresources.ProvidersClientRegisterOptions,
) (armresources.ProvidersClientRegisterResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.registerErr != nil {
		return armresources.ProvidersClientRegisterResponse{}, f.registerErr
	}
	f.registered = append(f.registered, ns)
	f.state[ns] = registeringState
	f.pending[ns] = f.getsUntilRegistered
	return armresources.ProvidersClientRegisterResponse{}, nil
}

func TestEnsureProviderRegisteredAlreadyRegistered(t *testing.T) {
	f := newFakeProviders(map[string]string{nsInsights: "registered"}) // case differs on purpose
	state, err := ensureProviderRegistered(context.Background(), f, nsInsights, time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, "registered", state)
	assert.Empty(t, f.registered, "no Register call for a registered namespace")
	assert.Equal(t, 1, f.gets)
}

func TestEnsureProviderRegisteredRegistersAndWaits(t *testing.T) {
	f := newFakeProviders(map[string]string{})
	f.getsUntilRegistered = 2
	state, err := ensureProviderRegistered(context.Background(), f, nsInsights, time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, registeredState, state)
	assert.Equal(t, []string{nsInsights}, f.registered)
}

func TestEnsureProviderRegisteredWaitsWhenRegistering(t *testing.T) {
	f := newFakeProviders(map[string]string{nsInsights: registeringState})
	f.pending[nsInsights] = 1
	state, err := ensureProviderRegistered(context.Background(), f, nsInsights, time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, registeredState, state)
	assert.Empty(t, f.registered, "no second Register call while Azure is already registering")
}

func TestEnsureProviderRegisteredTimesOut(t *testing.T) {
	f := newFakeProviders(map[string]string{})
	f.getsUntilRegistered = 1 << 30
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := ensureProviderRegistered(ctx, f, nsInsights, time.Millisecond)
	require.ErrorIs(t, err, ErrProviderRegistrationTimedOut)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestEnsureProviderRegisteredForbidden(t *testing.T) {
	f := newFakeProviders(map[string]string{})
	f.registerErr = &azcore.ResponseError{StatusCode: http.StatusForbidden, ErrorCode: "AuthorizationFailed"}
	_, err := ensureProviderRegistered(context.Background(), f, nsInsights, time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Microsoft.Insights/register/action")
	assert.Contains(t, err.Error(), "az provider register --namespace Microsoft.Insights")
}

func TestEnsureProviderRegisteredOtherError(t *testing.T) {
	f := newFakeProviders(map[string]string{})
	f.registerErr = errBoom
	_, err := ensureProviderRegistered(context.Background(), f, nsInsights, time.Millisecond)
	require.ErrorIs(t, err, errBoom)
}

func TestResourceProviderRegistrationCreate(t *testing.T) {
	f := newFakeProviders(map[string]string{})
	var gotSub string
	r := &ResourceProviderRegistration{newClient: func(sub string) (providersAPI, error) {
		gotSub = sub
		return f, nil
	}}
	in := ResourceProviderRegistrationInputs{SubscriptionID: "sub-1", Namespace: nsInsights}

	// Preview: no Azure call.
	resp, err := r.Create(context.Background(), infer.CreateRequest[ResourceProviderRegistrationInputs]{
		Name: "p-Microsoft.Insights", Inputs: in, DryRun: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "/subscriptions/sub-1/providers/Microsoft.Insights", resp.ID)
	assert.Zero(t, f.gets, "preview must not call Azure")

	resp, err = r.Create(context.Background(), infer.CreateRequest[ResourceProviderRegistrationInputs]{
		Name: "p-Microsoft.Insights", Inputs: in,
	})
	require.NoError(t, err)
	assert.Equal(t, "sub-1", gotSub)
	assert.Equal(t, "/subscriptions/sub-1/providers/Microsoft.Insights", resp.ID)
	assert.Equal(t, registeredState, resp.Output.RegistrationState)
	assert.Equal(t, in, resp.Output.ResourceProviderRegistrationInputs)
	assert.Equal(t, []string{nsInsights}, f.registered)
}

func TestResourceProviderRegistrationRead(t *testing.T) {
	f := newFakeProviders(map[string]string{nsApp: "Unregistered"})
	r := &ResourceProviderRegistration{newClient: func(string) (providersAPI, error) { return f, nil }}
	in := ResourceProviderRegistrationInputs{SubscriptionID: "sub-1", Namespace: nsApp}
	resp, err := r.Read(context.Background(),
		infer.ReadRequest[ResourceProviderRegistrationInputs, ResourceProviderRegistrationState]{
			ID:     "/subscriptions/sub-1/providers/Microsoft.App",
			Inputs: in,
			State:  ResourceProviderRegistrationState{ResourceProviderRegistrationInputs: in, RegistrationState: "Registered"},
		})
	require.NoError(t, err)
	assert.Equal(t, "Unregistered", resp.State.RegistrationState)
	assert.Empty(t, f.registered, "refresh must not register")
}

func TestRequiredResourceProviders(t *testing.T) {
	base := []string{nsKeyVault, nsManagedIdentity, nsNetwork}

	t.Run("container app", func(t *testing.T) {
		services := compose.Services{"web": {Image: pulumi.String("nginx")}}
		got := requiredResourceProviders(detectServiceTypes(services), services)
		assert.Equal(t, []string{
			nsApp, nsInsights, nsKeyVault, nsManagedIdentity,
			nsNetwork, nsOperationalInsights,
		}, got)
	})

	t.Run("managed services only", func(t *testing.T) {
		services := compose.Services{
			"db":    {Image: pulumi.String("postgres:17"), Postgres: &compose.PostgresConfig{}},
			"cache": {Image: pulumi.String("redis:7"), Redis: &compose.RedisConfig{}},
		}
		got := requiredResourceProviders(detectServiceTypes(services), services)
		assert.Equal(t, append([]string{nsCache, nsPostgreSQL}, base...), got)
		assert.NotContains(t, got, nsInsights)
	})
}
