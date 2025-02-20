package provider

import (
	"testing"

	"github.com/blang/semver"
	integration "github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/stretchr/testify/assert"
)

func TestProject(t *testing.T) {
	server := integration.NewServer("defang", semver.Version{Minor: 1}, Provider())
	integration.LifeCycleTest{
		Resource: "defang:index:Project",
		Create: integration.Operation{
			Inputs: resource.NewPropertyMapFromMap(map[string]interface{}{}),
			Hook: func(_inputs, output resource.PropertyMap) {
				t.Logf("Outputs: %v", output)
				result := output["result"].StringValue()
				assert.Len(t, result, 24)
			},
		},
		Updates: []integration.Operation{
			{
				Inputs: resource.NewPropertyMapFromMap(map[string]interface{}{}),
				Hook: func(_inputs, output resource.PropertyMap) {
					result := output["result"].StringValue()
					assert.Len(t, result, 10)
				},
			},
		},
	}.Run(t, server)
}
