package defangazure

import (
	providerazure "github.com/DefangLabs/pulumi-defang/provider/defangazure/azure"
	"github.com/DefangLabs/pulumi-defang/provider/modelselect"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// ModelSelection persists the Azure AI model revision chosen for a managed
// model service, so unchanged inputs keep the same revision across deployments.
type ModelSelection struct{ modelselect.ModelSelector }

func newModelSelection() *ModelSelection {
	return &ModelSelection{modelselect.ModelSelector{
		Cloud:         providerazure.ModelCloud,
		Resolver:      providerazure.NewFoundryResolver(),
		ScopeRequired: true, // the account is unknown in a preview that creates it
	}}
}

// Annotate documents the resource in the schema.
func (m *ModelSelection) Annotate(a infer.Annotator) {
	a.Describe(m, "The model selected for a managed model service. Created by Project; "+
		"unchanged inputs keep the selected model, and refresh never re-resolves it.")
}
