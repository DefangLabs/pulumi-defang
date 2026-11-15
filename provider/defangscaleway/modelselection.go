package defangscaleway

import (
	providerscaleway "github.com/DefangLabs/pulumi-defang/provider/defangscaleway/scaleway"
	"github.com/DefangLabs/pulumi-defang/provider/modelselect"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// ModelSelection persists the Generative APIs model chosen for a managed
// model request, so unchanged inputs keep the same model across deployments.
type ModelSelection struct{ modelselect.ModelSelector }

func newModelSelection() *ModelSelection {
	return &ModelSelection{modelselect.ModelSelector{
		Cloud:    providerscaleway.ModelCloud,
		Resolver: providerscaleway.GenerativeAPIsResolver{},
	}}
}

// Annotate documents the resource in the schema.
func (m *ModelSelection) Annotate(a infer.Annotator) {
	a.Describe(m, "The model selected for a managed model service. Created by Project; "+
		"unchanged inputs keep the selected model, and refresh never re-resolves it.")
}
