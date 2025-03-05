---
title: Defang Provider for Pulumi Installation & Configuration
meta_desc: Provides an overview on how to configure the Pulumi Defang Provider.
layout: package
---
## Installation

The Pulumi provider for [Defang] (https://defang.io) - the easiest way to deploy your Docker Compose project to the cloud with Pulumi.

The Defang Pulumi provider, is available in most pulumi languages. 

* JavaScript/TypeScript: [`@defang-io/pulumi-defang`](https://www.npmjs.com/package/@defang-io/pulumi-defang)
* Python: [`pulumi-defang`](https://pypi.org/project/pulumi-defang/)
* Go: [`github.com/DefangLabs/pulumi-defang/sdk/v1/go/defang`](https://github.com/DefangLabs/pulumi-defang)
* Java: Coming soon
* Dotnet: Coming soon

## Authentication

Sign up for [Defang](https://defang.io) with your Github account. When run in a Github Actions workflow, the Defang Pulumi provider will automatically use environment varialbes Github providew to authenticate your Github user with Defang if you give your workflow the [appropriate permissions](https://docs.github.com/en/actions/security-for-github-actions/security-hardening-your-deployments/about-security-hardening-with-openid-connect#adding-permissions-settings). Defang use the `ACTIONS_ID_TOKEN_REQUEST_URL` and `ACTIONS_ID_TOKEN_REQUEST_TOKEN` env vars.

You will also need to authenticate with your cloud platform.
* For AWS, there are many ways to authenticate
    - Use the [`aws-actions/configure-aws-credentials`](https://github.com/aws-actions/configure-aws-credentials) Github Action
    - Use AWS Access Keys by setting the `AWS_ACCESS_KEY_ID`, and `AWS_ACCESS_KEY_SECRET` env vars.
* For Digital Ocean, you will need to set the following env vars:
    - `DIGITALOCEAN_TOKEN`
    - `SPACES_ACCESS_KEY_ID`
    - `SPACES_SECRET_ACCESS_KEY`
* For Google Cloud, you may wish to use the [`google-github-actions/auth`](https://github.com/google-github-actions/auth) Github Action

## Example usage

{{< chooser language "typescript,python,go,yaml" >}}
{{% choosable language typescript %}}
```yaml
# Pulumi.yaml provider configuration file
name: configuration-example
runtime: nodejs
config:
    defang:Project:
        providerID: aws
        configPaths:
            - ./compose.yaml
```

{{% /choosable %}}

{{% choosable language python %}}
```yaml
# Pulumi.yaml provider configuration file
name: configuration-example
runtime: python
config:
    defang:Project:
        providerID: aws
        configPaths:
            - ./compose.yaml
```

{{% /choosable %}}

{{% choosable language go %}}
```yaml
# Pulumi.yaml provider configuration file
name: configuration-example
runtime: go
config:
    defang:Project:
        providerID: aws
        configPaths:
            - ./compose.yaml

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
