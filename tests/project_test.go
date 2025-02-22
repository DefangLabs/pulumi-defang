package tests

import (
	"testing"

	integration "github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/stretchr/testify/assert"
)

func TestProject(t *testing.T) {
	server := makeTestServer()
	integration.LifeCycleTest{
		Resource: "defang:index:Project",
		Create: integration.Operation{
			Inputs: resource.NewPropertyMapFromMap(map[string]interface{}{
				"name":        "my-project",
				"providerID":  "test-provider",
				"configPaths": []string{"../compose.yaml.example"},
			}),
			Hook: func(_inputs, output resource.PropertyMap) {
				assert.Equal(t, output["name"].StringValue(), "my-project")
				assert.Equal(t, output["etag"].StringValue(), "abc123")
				assert.Equal(t, output["providerID"].StringValue(), "test-provider")
			},
		},
		Updates: []integration.Operation{
			{
				Inputs: resource.NewPropertyMapFromMap(map[string]interface{}{
					"name":        "my-project",
					"providerID":  "test-provider",
					"configPaths": []string{"../compose.yaml.example"},
				}),
				Hook: func(_inputs, output resource.PropertyMap) {
					assert.Equal(t, output["etag"].StringValue(), "abc123")
				},
			},
		},
	}.Run(t, server)
}
