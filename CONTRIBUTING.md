# Contributing

## Build & test the Defang provider

1. Run `make provider` to build the provider
1. Run `make sdks` to build the sdks
1. Run `make build` to build the provider and the sdks
1. Run `make install` to install the provider
1. Run `make examples` to generate the example programs in `examples/` off of the source `examples/yaml` example program.
1. Run `make up` to run the example program in `examples/yaml`.
1. Run `make down` to tear down the example program.

## Build the provider and install the plugin

   ```bash
   $ make build install
   ```
   
This will:

1. Create the SDK codegen binary and place it in a `./bin` folder (gitignored)
1. Create the provider binary and place it in the `./bin` folder (gitignored)
1. Generate the dotnet, Go, Node, and Python SDKs and place them in the `./sdk` folder
1. Install the provider on your machine.

## A brief repository overview

1. A `provider/` folder containing the building and implementation logic
    1. `cmd/pulumi-resource-defang/main.go` - holds the provider's implementation.
1. `sdk` - holds the generated code libraries.
1. `examples` a folder of Pulumi programs to try locally and/or use in CI.

## Additional Details

This repository depends on the pulumi-go-provider library. For more details on building providers, please check
the [Pulumi Go Provider docs](https://github.com/pulumi/pulumi-go-provider).

## Build Examples

Create an example program using the resources defined in your provider, and place it in the `examples/` folder.

You can now repeat the steps for [build, install, and test](#test-against-the-example).

## Configuring CI and releases

1. Follow the instructions laid out in the [deployment templates](./deployment-templates/README-DEPLOYMENT.md).

## References

Other resources/examples for implementing providers:
* [Pulumi Command provider](https://github.com/pulumi/pulumi-command/blob/master/provider/pkg/provider/provider.go)
* [Pulumi Go Provider repository](https://github.com/pulumi/pulumi-go-provider)
