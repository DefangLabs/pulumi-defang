# Defang Pulumi Provider

![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/DefangLabs/pulumi-defang?label=Version)

The Pulumi Provider for [Defang](https://defang.io) — Take your app from Docker Compose to a secure and scalable cloud deployment with Pulumi.

## Example usage

You can find complete working TypeScript, Python, Go, .NET, and Yaml code samples in the [`./examples`](https://github.com/DefangLabs/pulumi-defang/tree/main/examples) directory, and some example snippets below:

{{< chooser language "typescript,python,go,dotnet,yaml" >}}
{{% choosable language typescript %}}
```typescript
import * as defangaws from "@defang-io/pulumi-defang-aws";

const project = new defangaws.Project("aws-nodejs", {
    services: {
        app: {
            image: "nginx",
            ports: [{
                target: 80,
                mode: "ingress",
                appProtocol: "http",
            }],
        },
    },
});

export const endpoints = project.endpoints;
```

{{% /choosable %}}

{{% choosable language python %}}
```python
import pulumi
import pulumi_defang as defang

my_project = defang.Project("myProject",
    provider_id="aws",
    config_paths=["compose.yaml"])
pulumi.export("output", {
    "albArn": my_project.alb_arn,
    "etag": my_project.etag,
})
```

{{% /choosable %}}

{{% choosable language go %}}
```go
package main

import (
	defangaws "github.com/DefangLabs/pulumi-defang/sdk/go/defang-aws"
	"github.com/DefangLabs/pulumi-defang/sdk/go/defang-aws/shared"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		proj, err := defangaws.NewProject(ctx, "aws-go", &defangaws.ProjectArgs{
			Services: shared.ServiceInputMap{
				"app": shared.ServiceInputArgs{
					Image: pulumi.String("nginx:latest"),
					Ports: shared.PortConfigArray{
						shared.PortConfigArgs{
							Target:      pulumi.Int(80),
							Mode:        pulumi.StringPtr("ingress"),
							AppProtocol: pulumi.StringPtr("http"),
						},
					},

				},
		})
		if err != nil {
			return err
		}

		ctx.Export("endpoints", proj.Endpoints)

		return nil
	})
}
```

{{% /choosable %}}

{{% choosable language dotnet %}}
```dotnet
using System.Collections.Generic;
using Pulumi;
using DefangLabs.DefangAws;
using DefangLabs.DefangAws.Shared.Inputs;

return await Deployment.RunAsync(() =>
{
    var project = new Project("aws-dotnet", new ProjectArgs
    {
        Services =
        {
            ["app"] = new ServiceInputArgs
            {
                Image = "nginx",
                Ports =
                {
                    new PortConfigArgs { Target = 80, Mode = "ingress", AppProtocol = "http" },
                },
            },
        },
    });

    return new Dictionary<string, object?>
    {
        ["endpoints"] = project.Endpoints,
    };
});
```

{{% /choosable %}}

{{% choosable language yaml %}}
```yaml
# Pulumi.yaml provider configuration file
name: configuration-example
runtime: yaml
config:
    defang:Project:
        providerID: aws
        configPaths:
            - ./compose.yaml
```

{{% /choosable %}}
{{< /chooser >}}

## Installation and Configuration

See our [Installation and Configuration](https://pulumi.com/registry/packages/defang/installation-configuration/) docs

## Development

See the [Contributing](https://github.com/DefangLabs/pulumi-defang/blob/main/CONTRIBUTING.md) doc.
