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

package main

import (
	"context"

	p "github.com/pulumi/pulumi-go-provider"

	"github.com/DefangLabs/defang/src/pkg/cli"
	"github.com/DefangLabs/defang/src/pkg/cli/client"
	defang "github.com/DefangLabs/pulumi-defang/provider"
)

// Serve the provider against Pulumi's Provider protocol.
func main() {
	ctx := context.Background()
	fabric := cli.NewGrpcClient(ctx, defang.Fabric)
	// FIXME: "aws" is a place-holder value.
	// we won't know which cloud provider id to use yet.
	providerID := client.ProviderID("aws")
	cloudProvider, err := cli.NewProvider(ctx, providerID, fabric)
	if err != nil {
		panic(err)
	}
	_ = p.RunProvider(defang.Name, defang.Version, defang.Provider(ctx, fabric, cloudProvider))
}
