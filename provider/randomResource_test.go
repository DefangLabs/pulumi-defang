package provider

import (
	"testing"

	"github.com/blang/semver"
	integration "github.com/pulumi/pulumi-go-provider/integration"
	presource "github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/stretchr/testify/assert"
)

func TestRandomResource(t *testing.T) {
	server := integration.NewServer("defang", semver.Version{Minor: 1}, Provider())
	integration.LifeCycleTest{
		Resource: "defang:index:Random",
		Create: integration.Operation{
			Inputs: presource.NewPropertyMapFromMap(map[string]interface{}{
				"length": 24,
			}),
			Hook: func(_inputs, output presource.PropertyMap) {
				t.Logf("Outputs: %v", output)
				result := output["result"].StringValue()
				assert.Len(t, result, 24)
			},
		},
		Updates: []integration.Operation{
			{
				Inputs: presource.NewPropertyMapFromMap(map[string]interface{}{
					"length": 10,
				}),
				Hook: func(_inputs, output presource.PropertyMap) {
					result := output["result"].StringValue()
					assert.Len(t, result, 10)
				},
			},
		},
	}.Run(t, server)
}
