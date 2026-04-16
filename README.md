# Defang Pulumi Provider

![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/DefangLabs/pulumi-defang?label=Version)

The Pulumi Provider for [Defang](https://defang.io) — Take your app from Docker Compose to a secure and scalable cloud deployment with Pulumi.

## Example usage

The examples below use AWS. GCP and Azure follow the same pattern — just swap the package name (e.g. `defang-aws` → `defang-gcp`).
Complete working samples for all clouds and languages are in the [`./examples`](https://github.com/DefangLabs/pulumi-defang/tree/main/examples) directory.

{{< chooser language "typescript,python,go,dotnet,yaml" >}}
{{% choosable language typescript %}}
<!-- source: examples/aws-nodejs/index.ts -->
```typescript
import * as pulumi from "@pulumi/pulumi";
import * as defang_aws from "@defang-io/pulumi-defang-aws";

const awsDemo = new defang_aws.Project("aws-demo", {services: {
    app: {
        image: "nginx",
        ports: [{
            target: 80,
            mode: "ingress",
            appProtocol: "http",
        }],
    },
}});
export const endpoints = awsDemo.endpoints;
```

{{% /choosable %}}

{{% choosable language python %}}
<!-- source: examples/aws-python/__main__.py -->
```python
import pulumi
import pulumi_defang_aws as defang_aws

aws_demo = defang_aws.Project("aws-demo", services={
    "app": {
        "image": "nginx",
        "ports": [{
            "target": 80,
            "mode": "ingress",
            "app_protocol": "http",
        }],
    },
})
pulumi.export("endpoints", aws_demo.endpoints)
```

{{% /choosable %}}

{{% choosable language go %}}
<!-- source: examples/aws-go/main.go -->
```go
package main

import (
	defangaws "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws"
	"github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		awsDemo, err := defangaws.NewProject(ctx, "aws-demo", &defangaws.ProjectArgs{
			Services: compose.ServiceConfigMap{
				"app": &compose.ServiceConfigArgs{
					Image: pulumi.String("nginx"),
					Ports: compose.ServicePortConfigArray{
						&compose.ServicePortConfigArgs{
							Target:      pulumi.Int(80),
							Mode:        pulumi.String("ingress"),
							AppProtocol: pulumi.String("http"),
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("endpoints", awsDemo.Endpoints)
		return nil
	})
}
```

{{% /choosable %}}

{{% choosable language dotnet %}}
<!-- source: examples/aws-dotnet/Program.cs -->
```csharp
using System.Collections.Generic;
using System.Linq;
using Pulumi;
using DefangAws = DefangLabs.DefangAws;

return await Deployment.RunAsync(() => 
{
    var awsDemo = new DefangAws.Project("aws-demo", new()
    {
        Services = 
        {
            { "app", new DefangAws.Compose.Inputs.ServiceConfigArgs
            {
                Image = "nginx",
                Ports = new[]
                {
                    new DefangAws.Compose.Inputs.ServicePortConfigArgs
                    {
                        Target = 80,
                        Mode = "ingress",
                        AppProtocol = "http",
                    },
                },
            } },
        },
    });

    return new Dictionary<string, object?>
    {
        ["endpoints"] = awsDemo.Endpoints,
    };
});

```

{{% /choosable %}}

{{% choosable language yaml %}}
<!-- source: examples/aws-yaml/Pulumi.yaml -->
```yaml
name: defang-aws
runtime: yaml
description: Example using defang-aws to deploy services to AWS

plugins:
  providers:
    - name: defang-aws
      path: ../../bin

resources:
  aws-demo:
    type: defang-aws:index:Project
    properties:
      services:
        app:
          image: nginx
          ports:
            - target: 80
              mode: ingress
              appProtocol: http

outputs:
  endpoints: ${aws-demo.endpoints}
```

{{% /choosable %}}
{{< /chooser >}}

## Installation and Configuration

See our [Installation and Configuration](https://pulumi.com/registry/packages/defang/installation-configuration/) docs

## Development

See the [Contributing](https://github.com/DefangLabs/pulumi-defang/blob/main/CONTRIBUTING.md) doc.
