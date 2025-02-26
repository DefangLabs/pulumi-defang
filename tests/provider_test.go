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
	server := makeTestServer()

	response, err := server.Create(p.CreateRequest{
		Urn: urn("Project"),
		Properties: resource.PropertyMap{
			"name":       resource.NewStringProperty("my-project"),
			"providerID": resource.NewStringProperty("test-provider"),
			"configPaths": resource.NewArrayProperty([]resource.PropertyValue{
				resource.NewStringProperty("../compose.yaml.example"),
			}),
		},
		Preview: false,
	})

	require.NoError(t, err)

	assert.Equal(t, response.Properties["name"].StringValue(), "my-project")
	assert.Equal(t, response.Properties["etag"].StringValue(), "abc123")
	assert.Equal(t, response.Properties["providerID"].StringValue(), "test-provider")
}

// urn is a helper function to build an urn for running integration tests.
func urn(typ string) resource.URN {
	return resource.NewURN("stack", "proj", "",
		tokens.Type("test:index:"+typ), "name")
}
