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
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi-go-provider/middleware/schema"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"

	nodejsGen "github.com/pulumi/pulumi/pkg/v3/codegen/nodejs"

	pythonGen "github.com/pulumi/pulumi/pkg/v3/codegen/python"

	csharpGen "github.com/pulumi/pulumi/pkg/v3/codegen/dotnet"
)

// Version is initialized by the Go linker to contain the semver of this build.
var Version string

const Name string = "defang"

func Provider() p.Provider {
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

		Metadata: schema.Metadata{
			Description: "Take your app from Docker Compose to a secure and scalable cloud deployment with Pulumi.",
			Keywords: []string{
				"category/cloud", "category/infrastructure", "kind/native", "defang", "docker",
				"docker compose", "cloud", "aws", "azure", "gcp", "digital ocean",
			},
			Homepage:          "https://github.com/DefangLabs/pulumi-defang",
			Repository:        "https://github.com/DefangLabs/pulumi-defang",
			Publisher:         "Defang",
			LogoURL:           "https://raw.githubusercontent.com/DefangLabs/pulumi-defang/refs/heads/main/docs/logo.png",
			License:           "Apache-2.0",
			PluginDownloadURL: "github://api.github.com/DefangLabs",
			LanguageMap: map[string]any{
				"csharp": csharpGen.CSharpPackageInfo{
					RootNamespace: "DefangLabs",
				},
				// "go": goGen.GoPackageInfo{},
				"nodejs": nodejsGen.NodePackageInfo{
					PackageName: "@defang-io/pulumi-defang",
				},
				"python": pythonGen.PackageInfo{},
			},
		},
	})
}

// Define some provider-level configuration.
type Config struct{}
