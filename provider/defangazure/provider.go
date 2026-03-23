package defangazure

import (
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi-go-provider/middleware/schema"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"

	csharpGen "github.com/pulumi/pulumi/pkg/v3/codegen/dotnet"
	nodejsGen "github.com/pulumi/pulumi/pkg/v3/codegen/nodejs"
	pythonGen "github.com/pulumi/pulumi/pkg/v3/codegen/python"
)

// Version is initialized by the Go linker to contain the semver of this build.
var Version string

const Name string = "defang-azure"

func Provider() p.Provider {
	return infer.Provider(infer.Options{
		Components: []infer.InferredComponent{
			infer.Component[*Project, ProjectInputs, *ProjectOutputs](&Project{}),
			infer.Component[*Service, AzureContainerAppInputs, *AzureContainerAppOutputs](&Service{}),
			infer.Component[*Postgres, AzurePostgresInputs, *AzurePostgresOutputs](&Postgres{}),
		},
		ModuleMap: map[tokens.ModuleName]tokens.ModuleName{
			"provider":    "index",
			"defangazure": "index",
		},

		Metadata: schema.Metadata{
			Description: "Deploy containerized services to Azure with Pulumi.",
			Keywords: []string{
				"category/cloud", "category/infrastructure", "kind/native", "defang", "docker",
				"cloud", "azure", "containerapp",
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
				"nodejs": nodejsGen.NodePackageInfo{
					PackageName: "@defang-io/pulumi-defang-azure",
				},
				"python": pythonGen.PackageInfo{},
			},
		},
	})
}
