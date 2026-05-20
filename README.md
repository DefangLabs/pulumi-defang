# Defang Pulumi Providers

[![Version](https://img.shields.io/github/v/tag/DefangLabs/pulumi-defang?label=Version)](https://github.com/DefangLabs/pulumi-defang/releases)
[![CI](https://github.com/DefangLabs/pulumi-defang/actions/workflows/test.yml/badge.svg)](https://github.com/DefangLabs/pulumi-defang/actions/workflows/test.yml)
[![License](https://img.shields.io/badge/License-AGPL_3.0_%2F_Apache_2.0-blue.svg)](#license)
[![Pulumi Registry](https://img.shields.io/badge/Pulumi%20Registry-defang-blueviolet)](https://www.pulumi.com/registry/packages/defang/)

The Pulumi Providers for [Defang](https://defang.io) — take your app from Docker Compose to a secure, scalable cloud deployment with Pulumi. **Develop once, deploy anywhere**: the AWS, GCP, and Azure providers share a near-identical Compose-shaped API, so the same program targets a different cloud by swapping a single import.

The repo also contains the source of our CD program in the [`cd/`](cd/) directory, which serves as the
driver for the Defang deployments: it receives a Compose file, translates it to a Pulumi program, and runs `pulumi up` (or `destroy` etc.) to (de)provision the resources in it.

Most users will want to use these components through the [Defang CLI](https://github.com/DefangLabs/defang), which connects to your cloud account, bootstraps the CD environment and runs the CD image built by this repo.

If you're a Pulumi user, check out our [Pulumi Registry listing](https://www.pulumi.com/registry/packages/defang/) for installation instructions, API docs, and example usage of using the providers in your own Pulumi programs.

The [examples/](examples/) directory in this repo also contains complete working samples for all clouds and languages, including [examples/multi-cloud](examples/multi-cloud/) which deploys a single Compose app to all three clouds from a single Pulumi program.

## Components

Each provider (`defang-aws`, `defang-gcp`, `defang-azure`) exposes the same Pulumi resource palette:

- **`Project`** — the recommended entry point. Takes a full `services` map (Compose-style) and provisions shared infrastructure (VPC, networking, DNS, load balancers, build pipelines) alongside each service.
- **`Service`** — a single container service. Standalone use is **image-only**: `image` must refer to a pre-built image. Build-from-source is a `Project` responsibility because it needs the shared build pipeline (Artifact Registry + Cloud Build on GCP, ECR + CodeBuild on AWS, ACR on Azure).
- **`Build`** (AWS only) — an image-build resource used internally by `Project`.
- **`Postgres`** / **`Redis`** — managed database / cache components. Can be used standalone or as part of a `Project`.

Managed components (`Service`, `Postgres`, `Redis`) share one implementation between standalone and project-scoped use. Standalone instantiations skip the shared-infra provisioning and therefore don't support features that depend on it (VPC access, build-from-source, external load-balancer wiring).

### Resource naming

When creating Pulumi resources, these Pulumi components will only specify the **logical** names, which is either the service name (e.g. `app`) or a name to describe the resource's role (e.g. `shared-vpc`, `ecr-public`).
The **physical** name of the underlying cloud resource is determined by the Pulumi engine. By default, this will be the logical name followed by a hyphen and 7 random hex characters (e.g. `app-abc1234`).
To control the physical name, configure `autonaming` rules in the Pulumi configuration files, either globally or per resource type. See the [Pulumi autonaming docs](https://www.pulumi.com/docs/intro/concepts/resources/#autonaming) for details. See the [CD code](cd/main.go) for examples of how autonaming is used in Defang.

## Installation and Configuration

See our [Installation and Configuration](https://pulumi.com/registry/packages/defang/installation-configuration/) docs

## Development

See the [Contributing](https://github.com/DefangLabs/pulumi-defang/blob/main/CONTRIBUTING.md) doc.

## Example usage

The examples below use AWS. **The same code targets GCP or Azure by swapping the import** (`defang-aws` → `defang-gcp` or `defang-azure`) — the `services` map and the rest of the program are unchanged. Complete working samples for all three clouds and four languages are in the [`./examples`](https://github.com/DefangLabs/pulumi-defang/tree/main/examples) directory, and [`examples/multi-cloud`](https://github.com/DefangLabs/pulumi-defang/tree/main/examples/multi-cloud) deploys the same Compose app to all three from a single program.

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

## License

Dual-licensed:

- The **provider source** in this repo (everything that builds the `pulumi-resource-defang-{aws,gcp,azure}` binaries and the CD program in [`cd/`](cd/)) is licensed under the [GNU Affero General Public License v3.0](LICENSE) (AGPL-3.0). If you fork the providers or run a modified version as a network service, AGPL's source-disclosure obligations apply.
- The **generated SDKs** in [`sdk/`](sdk/) (the typed TypeScript, Python, Go, and .NET client libraries that you import into your Pulumi program) are licensed under [Apache License 2.0](sdk/LICENSE). Embedding them in your application carries no copyleft obligation — they're tooling, not infrastructure-of-your-app.

Practically: if you just consume these providers from your Pulumi program (the common case), you're using Apache-2.0 code. AGPL only enters the picture if you fork the provider engine itself.
