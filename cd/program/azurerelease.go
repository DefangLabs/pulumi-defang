package program

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	armappcontainers "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v3"
)

const (
	containerAppType       = "azure-native:app:ContainerApp"
	managedCertificateType = "Microsoft.App/managedEnvironments/managedCertificates"

	// releaseAppTimeout bounds the work for one Container App, so a stuck ARM
	// call can't stall the destroy that follows.
	releaseAppTimeout = 5 * time.Minute
)

// certInUseRetryDelay is the wait between delete attempts while Azure still
// reports the just-unbound certificate as in use. A var so tests can shorten it.
var certInUseRetryDelay = 10 * time.Second

// releaseStateResource is the part of a checkpoint resource that
// AzureContainerAppsToRelease reads.
type releaseStateResource struct {
	URN          string         `json:"urn"`
	Type         string         `json:"type"`
	Parent       string         `json:"parent"`
	Dependencies []string       `json:"dependencies"`
	DeletedWith  string         `json:"deletedWith"`
	Outputs      map[string]any `json:"outputs"`
}

// AzureContainerAppsToRelease returns the ARM IDs of the Container Apps in the
// stack checkpoint (the deployment object of `pulumi stack export`) that a
// destroy with the given targets deletes. With no targets, that is every
// Container App. With targets, it is the targets and, as with
// optdestroy.TargetDependents, everything that transitively depends on them
// through a parent, dependency or deletedWith edge.
func AzureContainerAppsToRelease(deployment []byte, targets []string) ([]string, error) {
	if len(deployment) == 0 {
		return nil, nil
	}
	var state struct {
		Resources []releaseStateResource `json:"resources"`
	}
	if err := json.Unmarshal(deployment, &state); err != nil {
		return nil, fmt.Errorf("reading stack state: %w", err)
	}
	selected := make(map[string]bool, len(targets))
	for _, t := range targets {
		selected[t] = true
	}
	dependsOnSelected := func(r *releaseStateResource) bool {
		return selected[r.Parent] || selected[r.DeletedWith] ||
			slices.ContainsFunc(r.Dependencies, func(d string) bool { return selected[d] })
	}
	// Checkpoints list a resource after its dependencies, so one pass is
	// normally enough; iterate to a fixed point so order doesn't matter.
	for changed := len(targets) > 0; changed; {
		changed = false
		for i := range state.Resources {
			r := &state.Resources[i]
			if !selected[r.URN] && dependsOnSelected(r) {
				selected[r.URN] = true
				changed = true
			}
		}
	}
	var ids []string
	for i := range state.Resources {
		r := &state.Resources[i]
		if r.Type != containerAppType || (len(targets) > 0 && !selected[r.URN]) {
			continue
		}
		if id, _ := r.Outputs["id"].(string); id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// customDomainAPI is the subset of the Container Apps ARM API that
// releaseCustomDomains needs. It is an interface so tests can fake it.
type customDomainAPI interface {
	customDomains(ctx context.Context, app *arm.ResourceID) ([]*armappcontainers.CustomDomain, error)
	clearCustomDomains(ctx context.Context, app *arm.ResourceID) error
	deleteManagedCert(ctx context.Context, cert *arm.ResourceID) error
}

// ReleaseAzureCustomDomains removes the custom domain bindings from each
// Container App and then deletes the managed certificates they used.
//
// The CD program binds these out of band (provisionCerts / aca.IssueCert), so
// they are not in the Pulumi state and destroy does not sequence them. Without
// this step Azure keeps the app delete in progress indefinitely and refuses to
// delete the certificate (CertificateInUse) while the binding exists; the
// managed environment delete then also fails. Run this before the destroy.
//
// Failures are logged and do not stop the destroy: destroy can still succeed
// without this step (for example, when the app has no bindings).
func ReleaseAzureCustomDomains(ctx context.Context, appIDs []string, log func(string, ...any)) {
	if len(appIDs) == 0 {
		return
	}
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log("custom domains: cannot release bindings before destroy: %v", err)
		return
	}
	releaseCustomDomains(ctx, &armCustomDomainAPI{cred: cred}, appIDs, log)
}

func releaseCustomDomains(ctx context.Context, api customDomainAPI, appIDs []string, log func(string, ...any)) {
	for _, appID := range appIDs {
		app, err := arm.ParseResourceID(appID)
		if err != nil {
			log("custom domains: skipping %q: %v", appID, err)
			continue
		}
		appCtx, cancel := context.WithTimeout(ctx, releaseAppTimeout)
		if err := releaseApp(appCtx, api, app, log); err != nil {
			log("custom domains: release on %s failed: %v", app.Name, err)
		}
		cancel()
	}
}

func releaseApp(ctx context.Context, api customDomainAPI, app *arm.ResourceID, log func(string, ...any)) error {
	domains, err := api.customDomains(ctx, app)
	if err != nil {
		if isNotFound(err) {
			return nil // already gone
		}
		return err
	}
	if len(domains) == 0 {
		return nil
	}
	var certs []*arm.ResourceID
	var names []string
	for _, d := range domains {
		if d == nil {
			continue
		}
		if d.Name != nil {
			names = append(names, *d.Name)
		}
		if d.CertificateID == nil {
			continue
		}
		cert, err := arm.ParseResourceID(*d.CertificateID)
		if err != nil || !strings.EqualFold(cert.ResourceType.String(), managedCertificateType) {
			continue // only managed certificates are ours to delete
		}
		if !slices.ContainsFunc(certs, func(c *arm.ResourceID) bool { return strings.EqualFold(c.String(), cert.String()) }) {
			certs = append(certs, cert)
		}
	}

	log("Removing custom domains %s from %s before destroy", strings.Join(names, ", "), app.Name)
	if err := api.clearCustomDomains(ctx, app); err != nil {
		return fmt.Errorf("removing custom domains: %w", err)
	}
	for _, cert := range certs {
		if err := deleteCertWhenUnused(ctx, api, cert); err != nil {
			log("custom domains: deleting managed certificate %s failed: %v", cert.Name, err)
		}
	}
	return nil
}

// deleteCertWhenUnused deletes the certificate, retrying while Azure still
// reports it in use: the unbind is complete in ARM before the environment
// stops counting the binding.
func deleteCertWhenUnused(ctx context.Context, api customDomainAPI, cert *arm.ResourceID) error {
	for {
		err := api.deleteManagedCert(ctx, cert)
		if err == nil || isNotFound(err) {
			return nil
		}
		if !isConflict(err) {
			return err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w (last error: %w)", ctx.Err(), err)
		case <-time.After(certInUseRetryDelay):
		}
	}
}

func isNotFound(err error) bool {
	var re *azcore.ResponseError
	return errors.As(err, &re) && re.StatusCode == http.StatusNotFound
}

func isConflict(err error) bool {
	var re *azcore.ResponseError
	return errors.As(err, &re) && (re.StatusCode == http.StatusConflict || re.ErrorCode == "CertificateInUse")
}

type armCustomDomainAPI struct {
	cred azcore.TokenCredential
}

func (a *armCustomDomainAPI) customDomains(ctx context.Context, app *arm.ResourceID) ([]*armappcontainers.CustomDomain, error) {
	client, err := armappcontainers.NewContainerAppsClient(app.SubscriptionID, a.cred, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Get(ctx, app.ResourceGroupName, app.Name, nil)
	if err != nil {
		return nil, err
	}
	p := resp.Properties
	if p == nil || p.Configuration == nil || p.Configuration.Ingress == nil {
		return nil, nil
	}
	return p.Configuration.Ingress.CustomDomains, nil
}

func (a *armCustomDomainAPI) clearCustomDomains(ctx context.Context, app *arm.ResourceID) error {
	client, err := armappcontainers.NewContainerAppsClient(app.SubscriptionID, a.cred, nil)
	if err != nil {
		return err
	}
	// PATCH merges, so only the customDomains array is replaced. An empty,
	// non-nil slice serializes as [] and removes every binding.
	body := armappcontainers.ContainerApp{
		Properties: &armappcontainers.ContainerAppProperties{
			Configuration: &armappcontainers.Configuration{
				Ingress: &armappcontainers.Ingress{
					CustomDomains: []*armappcontainers.CustomDomain{},
				},
			},
		},
	}
	poller, err := client.BeginUpdate(ctx, app.ResourceGroupName, app.Name, body, nil)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (a *armCustomDomainAPI) deleteManagedCert(ctx context.Context, cert *arm.ResourceID) error {
	client, err := armappcontainers.NewManagedCertificatesClient(cert.SubscriptionID, a.cred, nil)
	if err != nil {
		return err
	}
	_, err = client.Delete(ctx, cert.ResourceGroupName, cert.Parent.Name, cert.Name, nil)
	return err
}
