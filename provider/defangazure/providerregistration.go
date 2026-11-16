package defangazure

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ResourceProviderRegistrationType is the type token of the
// ResourceProviderRegistration resource.
const ResourceProviderRegistrationType = Name + ":index:ResourceProviderRegistration"

const (
	registeredState            = "Registered"
	registeringState           = "Registering"
	providerRegistrationWait   = 10 * time.Minute
	providerRegistrationPeriod = 5 * time.Second
)

var (
	ErrProviderRegistrationTimedOut  = errors.New("resource provider registration did not finish in time")
	ErrInvalidProviderRegistrationID = errors.New(
		"invalid resource provider registration ID, want /subscriptions/<id>/providers/<namespace>")
)

// ResourceProviderRegistration makes sure an Azure resource provider namespace
// is registered on the subscription before resources of that namespace are
// created. ARM registers most namespaces by itself on the first resource PUT,
// but not for every resource type (for example the Microsoft.Insights
// diagnostic settings), which then fail with SubscriptionNotRegistered.
//
// Deleting the resource leaves the namespace registered: registration is
// subscription-wide and other workloads can depend on it.
type ResourceProviderRegistration struct {
	// newClient is overridden in tests.
	newClient func(subscriptionID string) (providersAPI, error)
}

// ResourceProviderRegistrationInputs are the inputs of ResourceProviderRegistration.
type ResourceProviderRegistrationInputs struct {
	// Azure subscription ID
	SubscriptionID string `pulumi:"subscriptionId"`

	// Resource provider namespace, e.g. "Microsoft.Insights"
	Namespace string `pulumi:"namespace"`
}

// ResourceProviderRegistrationState is the state of ResourceProviderRegistration.
type ResourceProviderRegistrationState struct {
	ResourceProviderRegistrationInputs

	// Registration state reported by Azure, e.g. "Registered"
	RegistrationState string `pulumi:"registrationState"`
}

// Annotate documents the resource in the schema.
func (r *ResourceProviderRegistration) Annotate(a infer.Annotator) {
	a.Describe(r, "Registers an Azure resource provider namespace on the subscription and waits until it is "+
		"registered. Created by Project. Deleting it does not unregister the namespace.")
}

// Annotate documents the inputs in the schema.
func (i *ResourceProviderRegistrationInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.SubscriptionID, "Azure subscription ID.")
	a.Describe(&i.Namespace, "Resource provider namespace, for example Microsoft.Insights.")
}

// Annotate documents the state in the schema.
func (s *ResourceProviderRegistrationState) Annotate(a infer.Annotator) {
	a.Describe(&s.RegistrationState, "Registration state that Azure reported, for example Registered.")
}

func providerRegistrationID(subscriptionID, namespace string) string {
	return "/subscriptions/" + subscriptionID + "/providers/" + namespace
}

// parseProviderRegistrationID is the inverse of providerRegistrationID.
func parseProviderRegistrationID(id string) (ResourceProviderRegistrationInputs, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 5 || parts[0] != "" || !strings.EqualFold(parts[1], "subscriptions") ||
		!strings.EqualFold(parts[3], "providers") || parts[2] == "" || parts[4] == "" {
		return ResourceProviderRegistrationInputs{}, fmt.Errorf("%w: %q", ErrInvalidProviderRegistrationID, id)
	}
	return ResourceProviderRegistrationInputs{SubscriptionID: parts[2], Namespace: parts[4]}, nil
}

// Create registers the namespace if needed and waits until it is registered.
func (r *ResourceProviderRegistration) Create(
	ctx context.Context, req infer.CreateRequest[ResourceProviderRegistrationInputs],
) (infer.CreateResponse[ResourceProviderRegistrationState], error) {
	in := req.Inputs
	resp := infer.CreateResponse[ResourceProviderRegistrationState]{
		ID:     providerRegistrationID(in.SubscriptionID, in.Namespace),
		Output: ResourceProviderRegistrationState{ResourceProviderRegistrationInputs: in},
	}
	if req.DryRun {
		return resp, nil
	}
	client, err := r.client(in.SubscriptionID)
	if err != nil {
		return resp, err
	}
	ctx, cancel := context.WithTimeout(ctx, providerRegistrationWait)
	defer cancel()
	state, err := ensureProviderRegistered(ctx, client, in.Namespace, providerRegistrationPeriod)
	if err != nil {
		return resp, err
	}
	resp.Output.RegistrationState = state
	return resp, nil
}

// Read refreshes the registration state.
func (r *ResourceProviderRegistration) Read(
	ctx context.Context,
	req infer.ReadRequest[ResourceProviderRegistrationInputs, ResourceProviderRegistrationState],
) (infer.ReadResponse[ResourceProviderRegistrationInputs, ResourceProviderRegistrationState], error) {
	resp := infer.ReadResponse[ResourceProviderRegistrationInputs, ResourceProviderRegistrationState](req)
	// A lookup or import passes only the ID.
	in, err := parseProviderRegistrationID(req.ID)
	if err != nil {
		return resp, err
	}
	resp.Inputs = in
	resp.State.ResourceProviderRegistrationInputs = in
	client, err := r.client(in.SubscriptionID)
	if err != nil {
		return resp, err
	}
	state, err := providerState(ctx, client, in.Namespace)
	if err != nil {
		return resp, err
	}
	resp.State.RegistrationState = state
	return resp, nil
}

func (r *ResourceProviderRegistration) client(subscriptionID string) (providersAPI, error) {
	if r.newClient != nil {
		return r.newClient(subscriptionID)
	}
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("getting Azure credential: %w", err)
	}
	return armresources.NewProvidersClient(subscriptionID, cred, nil)
}

// providersAPI is the subset of armresources.ProvidersClient that is used here.
type providersAPI interface {
	Get(
		ctx context.Context, namespace string, options *armresources.ProvidersClientGetOptions,
	) (armresources.ProvidersClientGetResponse, error)
	Register(
		ctx context.Context, namespace string, options *armresources.ProvidersClientRegisterOptions,
	) (armresources.ProvidersClientRegisterResponse, error)
}

// ensureProviderRegistered registers namespace unless it is already registered
// (or registering), then polls until Azure reports it as registered.
func ensureProviderRegistered(
	ctx context.Context, client providersAPI, namespace string, interval time.Duration,
) (string, error) {
	state, err := providerState(ctx, client, namespace)
	if err != nil {
		return "", err
	}
	if strings.EqualFold(state, registeredState) {
		return state, nil
	}
	if !strings.EqualFold(state, registeringState) {
		if _, err := client.Register(ctx, namespace, nil); err != nil {
			return "", registrationError(namespace, err)
		}
	}
	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("%w: %s is %q: %w", ErrProviderRegistrationTimedOut, namespace, state, ctx.Err())
		case <-time.After(interval):
		}
		if state, err = providerState(ctx, client, namespace); err != nil {
			return "", err
		}
		if strings.EqualFold(state, registeredState) {
			return state, nil
		}
	}
}

func providerState(ctx context.Context, client providersAPI, namespace string) (string, error) {
	resp, err := client.Get(ctx, namespace, nil)
	if err != nil {
		return "", fmt.Errorf("reading registration state of %s: %w", namespace, err)
	}
	if resp.RegistrationState == nil {
		return "", nil
	}
	return *resp.RegistrationState, nil
}

func registrationError(namespace string, err error) error {
	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
		return fmt.Errorf("registering resource provider %s: the deploying identity needs %s/register/action "+
			"on the subscription, or register it once with `az provider register --namespace %s`: %w",
			namespace, namespace, namespace, err)
	}
	return fmt.Errorf("registering resource provider %s: %w", namespace, err)
}

// registerResourceProviders registers one ResourceProviderRegistration per
// namespace and returns them, for use as DependsOn of the resources that need
// them.
func registerResourceProviders(
	ctx *pulumi.Context,
	projectName string,
	subscriptionID pulumi.StringInput,
	namespaces []string,
	pluginID common.PluginIdentity,
	opts ...pulumi.ResourceOption,
) ([]pulumi.Resource, error) {
	deps := make([]pulumi.Resource, 0, len(namespaces))
	for _, ns := range namespaces {
		var res pulumi.CustomResourceState
		if err := ctx.RegisterResource(ResourceProviderRegistrationType, projectName+"-"+ns, pulumi.Map{
			"subscriptionId": subscriptionID,
			"namespace":      pulumi.String(ns),
		}, &res, pluginID.ResourceOptions(opts...)...); err != nil {
			return nil, fmt.Errorf("registering resource provider %s: %w", ns, err)
		}
		deps = append(deps, &res)
	}
	return deps, nil
}
