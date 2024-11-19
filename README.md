# pulumi-defang
Typescript Pulumi provider for the Defang Opinionated Platform.

This [Dynamic Resource Provider](https://www.pulumi.com/docs/intro/concepts/resources/dynamic-providers/) allows you to manage your container services on the Defang Opinionated Platform. It was modelled after the [Compose Spec](https://www.compose-spec.io).

## Install

The provider is available from NPM Package Registry. See [installation instructions](https://www.npmjs.com/package/@defang-io/pulumi-defang).
```sh
npm i @defang-io/pulumi-defang
```

## Example

```ts
import { DefangService } from "@defang-io/pulumi-defang/lib";

const service = new DefangService("defang-demo", {
  // name: "…",                             // defaults to the Pulumi resource name
  // image: "nginx:latest",                 // use a pre-built container image, or
  build: {                                  // build from source
    context: ".",
    dockerfile: "Dockerfile.dev",
  },
  networks: {
    public: {
      //aliases: ["demo.example.com"],
    },
  },
  ports: [{ target: 80, protocol: "http", mode: "ingress" }],
  // fabricDNS: "…",                        // override the Defang Fabric Controller endpoint
  // platform: "…",                         // "linux/arm64" | "linux/amd64" | "linux" (default)
  // environment: {
  //   RDS_HOST: "rds.endpoint",
  // },
  // deploy: {
  //   replicas: 1,
  //   resources: {
  //     reservations: { cpus: 0.5, memory: 512 },
  //   },
  // },
  // healthcheck: {
  //   test: ["CMD", "curl", "-f", "http://localhost:80/"], // for Alpine-based images use ["CMD", "wget", "-q", "-O-", …]
  //   interval: 30,
  //   timeout: 10,
  //   retries: 3,
  // },
});

export const fqdn = service.publicFqdn;     // the final FQDN for your service
export const natIPs = service.natIPs;       // the public NAT IPs of the service
export const fabricDNS = service.fabricDNS;
```

## Environment Variables

* `DEFANG_ACCESS_TOKEN` - your access token; defaults to the token file from the CLI
* `DEFANG_FABRIC` - override the Defang Fabric service endpoint; defaults to prod
* `DEFANG_FORCE_UP` - set this to `1` or `true` to force an update (for debugging only)
* `DEFANG_DEBUG` - set this to `1` or `true` to enable debug logging
* `SOURCE_DATE_EPOCH` - the modification time for the files in the build context; defaults to 315532800 (1980-01-01 00:00:00 UTC)

## Contributing

### Release a New Version

To release a new version, let NPM update the version number in `package.json`. The CI/CD pipeline will then publish the new version to the NPM registry.
```sh
npm version patch
git push
```
