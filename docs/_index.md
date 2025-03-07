# Defang Pulumi Provider

![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/DefangLabs/pulumi-defang?label=Version)

The Pulumi provider for [Defang - a radically simpler way to develop, deploy, and debug cloud applications](https://defang.io). The easiest way to deploy your Docker Compose project to the cloud with Pulumi.

## Installation

The Defang provider is available as a package in most Pulumi languages:

* JavaScript/TypeScript: [`@defang-io/pulumi-defang`](https://www.npmjs.com/package/@defang-io/pulumi-defang)
* Python: [`pulumi-defang`](https://pypi.org/project/pulumi-defang/)
* Go: [`github.com/checkly/pulumi-defang/sdk/v1/go/defang`](https://github.com/DefangLabs/pulumi-defang)

## Authentication

### Authenticating with Defang

Sign up for [Defang](https://defang.io) with your Github account.

#### Authenticating in Github Actions workflows

When run in a Github Actions workflow, the Defang Pulumi provider will automatically use environment varialbes Github providew to authenticate your Github user with Defang if you give your workflow the [appropriate permissions](https://docs.github.com/en/actions/security-for-github-actions/security-hardening-your-deployments/about-security-hardening-with-openid-connect#adding-permissions-settings). Defang use the `ACTIONS_ID_TOKEN_REQUEST_URL` and `ACTIONS_ID_TOKEN_REQUEST_TOKEN` env vars.

#### Authenticating with `defang token`

You can run `defang token --expires 30d` out of band with a reasonable duration and you can store the result in `DEFANG_ACCESS_TOKEN`.

### Authenticating with your cloud provider

You will also need to authenticate with your cloud provider.

* For AWS, there are many ways to authenticate
    - Use the [`aws-actions/configure-aws-credentials`](https://github.com/aws-actions/configure-aws-credentials) Github Action
    - Use AWS Access Keys by setting the `AWS_ACCESS_KEY_ID`, and `AWS_ACCESS_KEY_SECRET` env vars.
* For Digital Ocean, you will need to set the following env vars:
    - `DIGITALOCEAN_TOKEN`
    - `SPACES_ACCESS_KEY_ID`
    - `SPACES_SECRET_ACCESS_KEY`
* For Google Cloud, you may wish to use the [`google-github-actions/auth`](https://github.com/google-github-actions/auth) Github Action

## Example usage

You can find working Go, Python, and TypeScript code samples in the [`./examples`](https://github.com/DefangLabs/pulumi-defang/tree/main/examples) directory.

## Reference

For detailed reference documentation, please visit [the Pulumi registry](https://www.pulumi.com/registry/packages/defang/api-docs/).

## Development

See the [Contributing](CONTRIBUTING.md) doc.
