package azure

import (
	"context"
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
	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	managedCertificateType = "Microsoft.App/managedEnvironments/managedCertificates"

	// releaseTimeout bounds the release of one Container App.
	releaseTimeout = 5 * time.Minute
)

// certInUseRetryDelay is the wait between delete attempts while Azure still
// reports the just-unbound certificate as in use. A var so tests can shorten it.
var certInUseRetryDelay = 10 * time.Second

// CustomDomainReleaseType is the type token of the CustomDomainRelease resource.
const CustomDomainReleaseType = "defang-azure:index:CustomDomainRelease"

// RegisterCustomDomainRelease registers a CustomDomainRelease for the Container
// App. It takes the app's ID as input, so Pulumi deletes it before the app and
// its Delete can remove the custom domain bindings and managed certificates
// that the CD adds outside Pulumi (see customdomain.go).
func RegisterCustomDomainRelease(
	ctx *pulumi.Context,
	serviceName string,
	containerAppID pulumi.StringInput,
	pluginID common.PluginIdentity,
	opts ...pulumi.ResourceOption,
) error {
	var res pulumi.CustomResourceState
	err := ctx.RegisterResource(CustomDomainReleaseType, serviceName, pulumi.Map{
		"containerAppId": containerAppID,
	}, &res, pluginID.ResourceOptions(opts...)...)
	if err != nil {
		return fmt.Errorf("registering custom domain release for %s: %w", serviceName, err)
	}
	return nil
}

// customDomainAPI is the subset of the Container Apps ARM API that
// releaseCustomDomains needs. It is an interface so tests can fake it.
type customDomainAPI interface {
	customDomains(ctx context.Context, app *arm.ResourceID) ([]*armappcontainers.CustomDomain, error)
	clearCustomDomains(ctx context.Context, app *arm.ResourceID) error
	deleteManagedCert(ctx context.Context, cert *arm.ResourceID) error
}

// ReleaseCustomDomains removes the custom domain bindings from the Container
// App and then deletes the managed certificates they used. Without this,
// Azure keeps the app delete in progress indefinitely and refuses to delete
// the certificate (CertificateInUse) while the binding exists.
func ReleaseCustomDomains(ctx context.Context, containerAppID string, log func(string, ...any)) error {
	app, err := arm.ParseResourceID(containerAppID)
	if err != nil {
		return fmt.Errorf("parsing Container App ID %q: %w", containerAppID, err)
	}
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return fmt.Errorf("creating Azure credential: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, releaseTimeout)
	defer cancel()
	return releaseApp(ctx, &armCustomDomainAPI{cred: cred}, app, log)
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

	log("Removing custom domains %s from %s before deleting it", strings.Join(names, ", "), app.Name)
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

func (a *armCustomDomainAPI) customDomains(
	ctx context.Context, app *arm.ResourceID,
) ([]*armappcontainers.CustomDomain, error) {
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
