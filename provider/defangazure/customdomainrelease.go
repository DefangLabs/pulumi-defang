package defangazure

import (
	"context"
	"fmt"

	providerazure "github.com/DefangLabs/pulumi-defang/provider/defangazure/azure"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// CustomDomainRelease removes, on delete, the custom domain bindings and
// managed certificates that the CD adds to a Container App outside Pulumi.
// It depends on the Container App, so Pulumi deletes it first; without it,
// deleting an app with a bound managed certificate does not complete.
type CustomDomainRelease struct{}

// CustomDomainReleaseInputs are the inputs of a CustomDomainRelease.
type CustomDomainReleaseInputs struct {
	// ARM resource ID of the Container App
	ContainerAppID string `pulumi:"containerAppId"`
}

// CustomDomainReleaseState is the state of a CustomDomainRelease.
type CustomDomainReleaseState struct {
	CustomDomainReleaseInputs
}

// Annotate documents the resource in the schema.
func (r *CustomDomainRelease) Annotate(a infer.Annotator) {
	a.Describe(r, "Removes a Container App's custom domain bindings and managed certificates "+
		"before the app is deleted. Created by Project.")
}

// Annotate documents the inputs in the schema.
func (i *CustomDomainReleaseInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.ContainerAppID, "ARM resource ID of the Container App.")
}

// Create makes no Azure call: the bindings are added later, by the CD.
func (*CustomDomainRelease) Create(
	_ context.Context, req infer.CreateRequest[CustomDomainReleaseInputs],
) (infer.CreateResponse[CustomDomainReleaseState], error) {
	return infer.CreateResponse[CustomDomainReleaseState]{
		ID:     req.Name,
		Output: CustomDomainReleaseState{CustomDomainReleaseInputs: req.Inputs},
	}, nil
}

// Delete removes the bindings and certificates. It is a no-op when the app
// has none or no longer exists.
func (*CustomDomainRelease) Delete(
	ctx context.Context, req infer.DeleteRequest[CustomDomainReleaseState],
) (infer.DeleteResponse, error) {
	logger := p.GetLogger(ctx)
	log := func(format string, args ...any) { logger.Infof(format, args...) }
	if err := providerazure.ReleaseCustomDomains(ctx, req.State.ContainerAppID, log); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("releasing custom domains: %w", err)
	}
	return infer.DeleteResponse{}, nil
}
