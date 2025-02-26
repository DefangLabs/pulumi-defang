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

package provider

import (
	"context"
	"fmt"

	"github.com/DefangLabs/defang/src/pkg/cli"
	"github.com/DefangLabs/defang/src/pkg/cli/client"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
)

// Version is initialized by the Go linker to contain the semver of this build.
var Version string

const Name string = "defang"

var (
	Fabric         string = cli.DefangFabric
	fabricClient   client.FabricClient
	providerClient client.Provider
)

func Provider(ctx context.Context, fabric client.FabricClient, provider client.Provider) p.Provider {
	// FIXME: I'm not sure how to set a new attribute on the p.Provider, so I'm writing to a global for now
	fabricClient = fabric
	providerClient = provider

	if err := Authenticate(ctx); err != nil {
		panic(fmt.Errorf("failed to authenticate: %w", err))
	}

	// We tell the provider what resources it needs to support.
	// In this case, a single resource and component
	return infer.Provider(infer.Options{
		Resources: []infer.InferredResource{
			infer.Resource[Project, ProjectArgs, ProjectState](),
		},
		Components: []infer.InferredComponent{},
		Config:     infer.Config[Config](),
		ModuleMap: map[tokens.ModuleName]tokens.ModuleName{
			"provider": "index",
		},
	})
}

// Define some provider-level configuration.
type Config struct{}

func Authenticate(ctx context.Context) error {
	token := cli.GetExistingToken(Fabric)
	if token != "" {
		return nil
	}

	err := cli.NonInteractiveLogin(ctx, fabricClient, Fabric)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	return nil
}
