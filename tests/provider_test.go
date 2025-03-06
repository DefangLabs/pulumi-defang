// Copyright 2016-2023, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tests

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectCreate(t *testing.T) {
	t.Setenv("DEFANG_ACCESS_TOKEN", "test-defang-access-token")
	server := makeTestServer()

	response, err := server.Create(p.CreateRequest{
		Urn: urn("Project"),
		Properties: resource.PropertyMap{
			"providerID": resource.NewStringProperty("test-provider"),
			"configPaths": resource.NewArrayProperty([]resource.PropertyValue{
				resource.NewStringProperty("../compose.yaml.example"),
			}),
		},
		Preview: false,
	})

	require.NoError(t, err)

	assert.Equal(t, "abc123", response.Properties["etag"].StringValue())
	assert.Equal(t, "test-provider", response.Properties["providerID"].StringValue())
	albArn := "arn:aws:elasticloadbalancing:us-west-2:123456789012:loadbalancer/app/my-load-balancer/50dc6c495c0c9188"
	assert.Equal(t, albArn, response.Properties["albArn"].StringValue())
}

// urn is a helper function to build an urn for running integration tests.
func urn(typ string) resource.URN {
	return resource.NewURN("stack", "proj", "",
		tokens.Type("test:index:"+typ), "name")
}
