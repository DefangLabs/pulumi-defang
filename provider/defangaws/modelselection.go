package defangaws

import (
	provideraws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/DefangLabs/pulumi-defang/provider/modelselect"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// ModelSelection persists the Bedrock model chosen for a managed model
// service, so unchanged inputs keep the same model across deployments.
type ModelSelection struct{ modelselect.ModelSelector }

func newModelSelection() *ModelSelection {
	return &ModelSelection{modelselect.ModelSelector{
		Cloud:    provideraws.ModelCloud,
		Resolver: provideraws.NewBedrockResolver(),
	}}
}

// Annotate documents the resource in the schema.
func (m *ModelSelection) Annotate(a infer.Annotator) {
	a.Describe(m, "The model selected for a managed model service. Created by Project; "+
		"unchanged inputs keep the selected model, and refresh never re-resolves it.")
}
