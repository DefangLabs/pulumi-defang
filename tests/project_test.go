package tests

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/types"
	integration "github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/stretchr/testify/assert"
)

func TestProject(t *testing.T) {
	t.Setenv("DEFANG_ACCESS_TOKEN", "test-defang-access-token")
	server := makeTestServer()

	inputs := resource.NewPropertyMapFromMap(map[string]interface{}{
		"providerID": "test-provider",
		"config": &types.Project{
			Services: map[string]types.ServiceConfig{
				"web": {
					Image: "nginx",
					Ports: []types.ServicePortConfig{
						{
							Target:    80,
							Published: "8080",
						},
					},
				},
			},
		},
	})

	integration.LifeCycleTest{
		Resource: "defang:index:Project",
		Create: integration.Operation{
			Inputs: inputs,
			Hook: func(_inputs, output resource.PropertyMap) {
				assert.Equal(t, "abc123", output["etag"].StringValue())
				albArn := "arn:aws:elasticloadbalancing:us-west-2:123456789012:loadbalancer/app/my-load-balancer/50dc6c495c0c9188"
				assert.Equal(t, albArn, output["albArn"].StringValue())
				services := output["services"].ObjectValue()
				serviceValue := services["service1-id"]
				serviceConfig := serviceValue.ObjectValue()
				assert.Equal(t, "service1-id", serviceConfig["resource_name"].StringValue())
				assert.Equal(t, "service1-role", serviceConfig["task_role"].StringValue())
			},
		},
		Updates: []integration.Operation{
			{
				Inputs: resource.NewPropertyMapFromMap(map[string]interface{}{
					"providerID":  "test-provider",
					"configPaths": []string{"../compose.yaml.example"},
				}),
				Hook: func(_inputs, output resource.PropertyMap) {
					assert.Equal(t, "abc123", output["etag"].StringValue())
					services := output["services"].ObjectValue()
					serviceValue := services["service1-id"]
					serviceConfig := serviceValue.ObjectValue()
					assert.Equal(t, "service1-id", serviceConfig["resource_name"].StringValue())
					assert.Equal(t, "service1-role", serviceConfig["task_role"].StringValue())
				},
			},
		},
	}.Run(t, server)
}
