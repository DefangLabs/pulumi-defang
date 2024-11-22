import * as grpc from "@grpc/grpc-js";
import * as pulumi from "@pulumi/pulumi";
import { createHash } from "crypto";
import { createReadStream, promises, readFileSync } from "fs";
import { Empty } from "google-protobuf/google/protobuf/empty_pb";
import { dirname, join } from "path";
import { promises as stream } from "stream";
import assert = require("assert");

import { newClient } from "./client";
import * as fabric from "./protos/io/defang/v1/fabric_grpc_pb";
import * as pb from "./protos/io/defang/v1/fabric_pb";
import { createTarball, uploadTarball } from "./upload";
import {
  deleteUndefined,
  isEqual,
  isValidUint,
  optionals,
  stableStringify,
  trueOr1,
} from "./utils";

let defaultFabric =
  process.env["DEFANG_FABRIC"] || "fabric-prod1.defang.dev:443";

/** Override the default Fabric service to use for all Defang resources. */
export function setDefaultFabric(fabricDns: string) {
  assert(fabricDns, "fabricDns must be non-empty");
  defaultFabric = fabricDns;
}

const debug = trueOr1(process.env["DEFANG_DEBUG"]);

// Pulumi stores the actual code of the dynamic provider in the stack. This
// means that if there's a bug in the provider, we can't fix it in existing
// stacks. To work around this, we can force an update of the provider code by
// setting the environment variable DEFANG_FORCE_UP to "true" or "1".
const forceUpdate = trueOr1(process.env["DEFANG_FORCE_UP"]);

// The access token is used to authenticate with the gRPC server. It can be
// passed in as an environment variable, read from the state file, or set using
// the `setAccessToken` function.
let accessToken: string | undefined;

function printDefangHint(fabricDns: string) {
  const arg = fabricDns === defaultFabric ? "" : ` --cluster ${fabricDns}`;
  pulumi.log.error("Please log in with the Defang CLI: defang login" + arg);
}

function readAccessToken(fabricDns: string): string {
  const tokenDir =
    process.env["XDG_STATE_HOME"] ||
    join(process.env["HOME"]!, ".local", "state");
  const tokenPath = join(tokenDir, "defang", fabricDns.replace(/:\d+$/, ""));
  try {
    if (debug) pulumi.log.info(`Reading access token from ${tokenPath}`);
    return readFileSync(tokenPath, "utf8").trim();
  } catch (err) {
    printDefangHint(fabricDns);
    throw err;
  }
}

function getAccessToken(fabricDns: string): string {
  return (
    accessToken ||
    process.env["DEFANG_ACCESS_TOKEN"] ||
    readAccessToken(fabricDns)
  );
}

/** Override the access token used to authenticate with the Fabric service. */
export function setAccessToken(token: string) {
  assert(token, "token must be non-empty");
  accessToken = token;
}

// Connect to our gRPC server
async function connect(
  fabricDns: string
): Promise<fabric.FabricControllerClient> {
  const client = newClient(fabricDns, getAccessToken(fabricDns));
  await new Promise<void>((resolve, reject) =>
    client.waitForReady(Date.now() + 5000, (err) =>
      err ? reject(err) : resolve()
    )
  );
  return client;
}

function convertServiceInputs(inputs: DefangServiceInputs): pb.Service {
  const service = new pb.Service();

  service.setName(inputs.name);
  if (inputs.image) {
    service.setImage(inputs.image);
  }
  if (inputs.build?.context) {
    const build = new pb.Build();
    // inputs.build.context is handled in updatex()
    // build.setContext(…);
    if (inputs.build.dockerfile) {
      build.setDockerfile(inputs.build.dockerfile);
    }
    if (inputs.build.args) {
      for (const [key, value] of Object.entries(inputs.build.args)) {
        build.getArgsMap().set(key, value);
      }
    }
    if (inputs.build.shmSize) {
      build.setShmSize(inputs.build.shmSize);
    }
    if (inputs.build.target) {
      build.setTarget(inputs.build.target);
    }
    service.setBuild(build);
  }
  const deploy = new pb.Deploy();
  deploy.setReplicas(inputs.deploy?.replicas ?? 1);
  if (inputs.deploy?.resources) {
    const reservations = new pb.Resource();
    reservations.setCpus(inputs.deploy.resources.reservations?.cpus ?? 0.0);
    reservations.setMemory(inputs.deploy.resources.reservations?.memory ?? 0);
    const devices = inputs.deploy.resources.reservations?.devices?.map((d) => {
      const device = new pb.Device();
      device.setCapabilitiesList(d.capabilities ?? []);
      device.setCount(d.count ?? 0);
      return device;
    });
    reservations.setDevicesList(devices ?? []);
    const resources = new pb.Resources();
    resources.setReservations(reservations);
    deploy.setResources(resources);
  }
  const publicNetwork = inputs.networks?.public;
  const privateNetwork = inputs.networks?.private;
  if (publicNetwork !== undefined) {
    service.setNetworks(pb.Network.PUBLIC);
    service.setAliasesList(publicNetwork?.aliases ?? []);
  } else if (privateNetwork !== undefined) {
    service.setNetworks(pb.Network.PRIVATE);
  }
  service.setDeploy(deploy);
  service.setPlatform(
    inputs.platform === "linux/arm64"
      ? pb.Platform.LINUX_ARM64
      : pb.Platform.LINUX_AMD64
  );
  service.setPortsList(convertPorts(inputs.ports));
  Object.entries(inputs.environment ?? {}).forEach(([key, value]) => {
    if (value === null) {
      const secret = new pb.Secret();
      secret.setSource(key);
      service.getSecretsList().push(secret);
    } else {
      service.getEnvironmentMap().set(key, value);
    }
  });
  inputs.secrets?.forEach((s) => {
    const secret = new pb.Secret();
    secret.setSource(s.source);
    // secret.setTarget(s.target);
    service.getSecretsList().push(secret);
  });
  if (inputs.command) {
    service.setCommandList(inputs.command);
  }
  if (inputs.healthcheck) {
    const healthcheck = new pb.HealthCheck();
    healthcheck.setTestList(inputs.healthcheck.test);
    healthcheck.setInterval(inputs.healthcheck.interval ?? 0);
    healthcheck.setTimeout(inputs.healthcheck.timeout ?? 0);
    healthcheck.setRetries(inputs.healthcheck.retries ?? 0);
    service.setHealthcheck(healthcheck);
  }
  if (inputs.domainname) {
    service.setDomainname(inputs.domainname);
  }
  service.setInit(inputs.init ?? false);
  if (inputs.x_redis) {
    const redis = new pb.Redis();
    service.setRedis(redis);
  }
  if (inputs.x_static_files) {
    const staticFiles = new pb.StaticFiles();
    staticFiles.setFolder(inputs.x_static_files.folder);
    staticFiles.setRedirectsList(inputs.x_static_files.redirects ?? []);
    service.setStaticFiles(staticFiles);
  }
  return service;
}

function dummyServiceInfo(service: pb.Service): pb.ServiceInfo {
  const info = new pb.ServiceInfo();
  info.setService(service);
  return info;
}

async function sha256sum(path: string): Promise<string> {
  const hash = createHash("sha256");
  await stream.pipeline(createReadStream(path), hash);
  const digest = "sha256-" + hash.digest("base64"); // same as Nix
  if (debug) pulumi.log.info(`Digest: ${digest}`);
  return digest;
}

const s3InvalidCharsRegexp = /[^a-zA-Z0-9!_.*'()-]/g; // from fabric_grpc.go

function mockCreateUploadURL(old: string, digest: string): string {
  const suffix = digest.replace(s3InvalidCharsRegexp, "_"); // from fabric_grpc.go
  const baseUrl = dirname(old);
  return `${baseUrl}/${suffix}`;
}

async function getRemoteBuildContextDigest(
  context: string,
  dockerfile?: string
): Promise<string> {
  const temppath = await createTarball(context, dockerfile);
  try {
    return await sha256sum(temppath);
  } finally {
    await promises.rm(temppath);
    await promises.rmdir(dirname(temppath));
  }
}

async function getRemoteBuildContext(
  client: fabric.FabricControllerClient,
  context: string,
  force: boolean,
  dockerfile?: string
): Promise<string> {
  const temppath = await createTarball(context, dockerfile);
  try {
    const req = new pb.UploadURLRequest();

    if (!force) {
      // Calculate the digest of the tarball and pass it to the fabric controller (to avoid building the same image twice)
      req.setDigest(await sha256sum(temppath));
    }

    const uploadUrlResponse = await new Promise<pb.UploadURLResponse>(
      (resolve, reject) =>
        client.createUploadURL(req, (err, res) =>
          err ? reject(err) : resolve(res!)
        )
    );
    const putUrl = uploadUrlResponse.getUrl();
    if (debug) pulumi.log.info(`Uploading build context to ${putUrl}`);

    await uploadTarball(putUrl, temppath);

    return putUrl.replace(/\?.*$/, ""); // remove query params
  } finally {
    await promises.rm(temppath);
    await promises.rmdir(dirname(temppath));
  }
}

async function upsert(
  inputs: DefangServiceInputs,
  force: boolean = false
): Promise<pb.ServiceInfo> {
  const service = convertServiceInputs(inputs);
  service.setProject(inputs.project);
  const client = await connect(inputs.fabricDNS);
  try {
    // Upload the build context, if provided
    if (inputs.build?.context) {
      const build = service.getBuild();
      assert(build, "service.build should've been set in convertServiceInputs");
      build.setContext(
        await getRemoteBuildContext(
          client,
          inputs.build.context,
          force,
          build.getDockerfile()
        )
      );
    }

    // Update any secrets w/ values first in case the service update depends on them
    await Promise.all(
      inputs.secrets?.map((secret) => {
        if (secret.value === undefined) {
          return;
        }
        const sv = new pb.PutConfigRequest();
        sv.setProject(inputs.project);
        sv.setName(secret.source);
        sv.setValue(secret.value);
        return new Promise((resolve, reject) =>
          client.putSecret(sv, (err, res) =>
            err ? reject(err) : resolve(res!)
          )
        );
      }) ?? []
    );

    const serviceInfo = await new Promise<pb.ServiceInfo>((resolve, reject) =>
      client.update(service, (err, res) => (err ? reject(err) : resolve(res!)))
    );
    if (inputs.waitForSteadyState) {
      await retryWaitForSteadyState(
        client,
        serviceInfo.getEtag(),
        serviceInfo.getService()!.getName()
      );
    }
    // A service with a domainname but no zoneId is using Let's Encrypt
    if (serviceInfo.getService()?.getDomainname() && !serviceInfo.getZoneId()) {
      pulumi.log.warn(
        "Run `defang cert generate` to get a TLS certificate for the service."
      );
    }
    return serviceInfo;
  } catch (err) {
    if ((err as grpc.ServiceError).code === grpc.status.UNAUTHENTICATED) {
      printDefangHint(inputs.fabricDNS);
    }
    if (!force) throw err;
    pulumi.log.warn(`Forced update; ignoring error: ${err}`);
    return dummyServiceInfo(service);
  } finally {
    client.close();
  }
}

function waitForSteadyState(
  client: fabric.FabricControllerClient,
  etag: string,
  service: string
) {
  const subscribeRequest = new pb.SubscribeRequest();
  // subscribeRequest.setProject(project);
  subscribeRequest.setEtag(etag);
  subscribeRequest.addServices(service);
  const subscribeStream = client.subscribe(subscribeRequest);
  return new Promise<void>((resolve, reject) => {
    subscribeStream.on("error", (err) => {
      reject(err);
    });
    subscribeStream.on("data", (event: pb.SubscribeResponse) => {
      switch (event.getState()) {
        case pb.ServiceState.BUILD_FAILED:
        case pb.ServiceState.DEPLOYMENT_FAILED:
          reject(new Error(`Update failed: ${event.getStatus()}`));
          break;
        case pb.ServiceState.DEPLOYMENT_COMPLETED:
          resolve();
          break;
        default:
          console.debug(event.getStatus());
      }
    });
  });
}

type GrpcError = Error & { code: grpc.status; details: string; metadata: any };

async function retryWaitForSteadyState(
  client: fabric.FabricControllerClient,
  etag: string,
  service: string
) {
  while (true) {
    try {
      return await waitForSteadyState(client, etag, service);
    } catch (err) {
      switch ((err as GrpcError).code) {
        case grpc.status.INTERNAL:
          // {"code":13,"details":"Received RST_STREAM with code 2 (Internal server error)","metadata":{}}
          if (debug) pulumi.log.info(`Retry after INTERNAL error: ${err}`);
          break;
        case grpc.status.UNAVAILABLE:
          // {"code":14,"details":"unavailable","metadata":{"server":["awselb/2.0"],"date":["Fri, 18 Oct 2024 03:25:32 GMT"],"content-type":["application/grpc"],"content-length":["0"],"grpc-previous-rpc-attempts":["4"]}}
          if (debug) pulumi.log.info(`Retry after UNAVAILABLE error: ${err}`);
          break;
        default:
          throw err;
      }
      // Wait for 1s before retrying; this matches the behavior of the Defang CLI
      await new Promise((resolve) => setTimeout(resolve, 1000));
    }
  }
}

function convertProtocol(protocol?: Protocol) {
  switch (protocol) {
    case "tcp":
      return pb.Protocol.TCP;
    case "udp":
      return pb.Protocol.UDP;
    case "http":
      return pb.Protocol.HTTP;
    case "http2":
      return pb.Protocol.HTTP2;
    case "grpc":
      return pb.Protocol.GRPC;
    default:
      return pb.Protocol.ANY;
  }
}

function convertPorts(ports: Port[] = []): pb.Port[] {
  return ports.map((p) => {
    const port = new pb.Port();
    port.setTarget(p.target);
    port.setProtocol(convertProtocol(p.protocol));
    port.setMode(p.mode === "ingress" ? pb.Mode.INGRESS : pb.Mode.HOST); // default to HOST
    return port;
  });
}

interface StaticFiles {
  folder: string;
  redirects?: string[];
}

interface DefangServiceInputs {
  build?: pulumi.Unwrap<Build>;
  command?: string[];
  deploy?: Deploy;
  domainname?: string;
  environment?: { [key: string]: string | null };
  fabricDNS: string;
  forceNewDeployment?: boolean;
  healthcheck?: HealthCheck;
  image?: string;
  init?: boolean;
  name: string;
  networks?: Record<NetworkName, Network>;
  platform?: Platform;
  ports?: Port[];
  project: string;
  secrets?: pulumi.Unwrap<Secret>[];
  waitForSteadyState?: boolean;
  x_redis?: unknown;
  x_static_files?: StaticFiles;
}

interface DefangServiceOutputs {
  endpoints: string[];
  etag: string;
  fabricDNS: string;
  lbIPs: string[];
  natIPs: string[];
  privateFqdn?: string;
  publicFqdn?: string;
  service: pb.Service.AsObject; // this might contain undefined, which is not allowed
  // status: string;
  // tenant: string;
}

function unempty(s: string): string | undefined {
  return s || undefined;
}

function toOutputs(
  fabricDNS: string,
  service: pb.ServiceInfo,
  old?: DefangServiceOutputs
): DefangServiceOutputs {
  // FIXME: the fallbacks of `?? old?.` here don't work since the lhs is never undefined
  return deleteUndefined({
    endpoints: service.getEndpointsList() ?? old?.endpoints,
    etag: service.getEtag() ?? old?.etag,
    fabricDNS,
    lbIPs: service.getLbIpsList() ?? old?.lbIPs,
    natIPs: service.getNatIpsList() ?? old?.natIPs,
    privateFqdn: unempty(service.getPrivateFqdn() ?? old?.privateFqdn), // can be empty string
    publicFqdn: unempty(service.getPublicFqdn() ?? old?.publicFqdn), // can be empty string
    service: service.getService()!.toObject(),
  });
}

function isOptionalFloatGt0(x?: number): boolean {
  return x === undefined || x > 0; // returns false for NaN
}

function isOptionalIntGt0(x?: number): boolean {
  return x === undefined || (Number.isSafeInteger(x) && x > 0);
}

const defangServiceProvider: pulumi.dynamic.ResourceProvider<
  DefangServiceInputs,
  DefangServiceOutputs
> = {
  async check(
    olds: DefangServiceInputs,
    news: DefangServiceInputs
  ): Promise<pulumi.dynamic.CheckResult<DefangServiceInputs>> {
    const failures: pulumi.dynamic.CheckFailure[] = [];

    if (!news.fabricDNS) {
      failures.push({ property: "fabricDNS", reason: "fabricDNS is required" });
    }
    // TODO: validate name
    if (!news.name) {
      failures.push({ property: "name", reason: "name is required" });
    }
    if (news.deploy) {
      if (!isValidUint(news.deploy.replicas ?? 0)) {
        failures.push({
          property: "deploy",
          reason: "replicas must be an integer ≥ 0",
        });
      }
      if (!isOptionalFloatGt0(news.deploy.resources?.reservations?.cpus)) {
        failures.push({
          property: "deploy",
          reason: "cpus reservation must be > 0",
        });
      }
      if (!isOptionalFloatGt0(news.deploy.resources?.reservations?.memory)) {
        failures.push({
          property: "deploy",
          reason: "memory reservation must be > 0",
        });
      }
    }
    if (!news.deploy?.resources?.reservations?.memory) {
      pulumi.log.warn(
        "missing memory reservation; specify deploy.resources.reservations.memory to avoid out-of-memory errors"
      );
    }
    for (const port of news.ports || []) {
      // port.protocol = port.protocol || "tcp"; TODO: should we set defaults here?
      if (
        port.target < 1 ||
        port.target > 32767 ||
        !Number.isInteger(port.target)
      ) {
        failures.push({
          property: "ports",
          reason: "target port must be an integer between 1 and 32767",
        });
      }
      if (port.mode === "ingress") {
        if (["udp", "tcp"].includes(port.protocol!)) {
          failures.push({
            property: "ports",
            reason: "ingress is not support by protocol " + port.protocol,
          });
        }
        if (!news.healthcheck?.test) {
          pulumi.log.warn(
            "ingress port without healthcheck defaults to GET / HTTP/1.1"
          );
        }
      }
      if (news.healthcheck) {
        if (!isOptionalIntGt0(news.healthcheck.interval)) {
          failures.push({
            property: "healthcheck.interval",
            reason: "interval must be an integer > 0",
          });
        }
        if (!isOptionalIntGt0(news.healthcheck.timeout)) {
          failures.push({
            property: "healthcheck.timeout",
            reason: "timeout must be an integer > 0",
          });
        }
        if (!isOptionalIntGt0(news.healthcheck.retries)) {
          failures.push({
            property: "healthcheck.retries",
            reason: "retries must be an integer > 0",
          });
        }
      }
    }
    if (Object.keys(news.networks || {}).length > 1) {
      failures.push({
        property: "networks",
        reason: "only one network can be specified",
      });
    }
    for (const secret of news.secrets || []) {
      // TODO: validate source name
      if (!secret.source) {
        failures.push({
          property: "secrets",
          reason: "secret source is required",
        });
      }
    }
    if (news.build) {
      if (news.image) {
        failures.push({
          property: "image",
          reason: "cannot specify both build and image",
        });
      }
      if (!news.build.context) {
        failures.push({
          property: "build.context",
          reason: "build context is required",
        });
      }
      if (news.build.dockerfile === "") {
        failures.push({
          property: "build.dockerfile",
          reason: "dockerfile cannot be empty string",
        });
      }
      if (!isOptionalFloatGt0(news.build.shmSize)) {
        failures.push({
          property: "build.shmSize",
          reason: "shmSize must be an integer > 0",
        });
      }
      if (news.build.target === "") {
        failures.push({
          property: "build.target",
          reason: "target cannot be empty string",
        });
      }
    } else if (!news.image) {
      failures.push({ property: "image", reason: "image is required" });
    }
    return { inputs: news, failures };
  },
  async create(
    inputs: DefangServiceInputs
  ): Promise<pulumi.dynamic.CreateResult<DefangServiceOutputs>> {
    const result = await upsert(inputs);
    return {
      id: result.getService()!.getName(), // TODO: do we need to return a unique ID?
      outs: toOutputs(inputs.fabricDNS, result),
    };
  },
  async delete(id: string, olds: DefangServiceOutputs): Promise<void> {
    const serviceIds = new pb.DeleteRequest();
    serviceIds.setProject(olds.service.project);
    serviceIds.addNames(id);
    const client = await connect(olds.fabricDNS);
    try {
      await new Promise<Empty>((resolve, reject) =>
        client.delete(serviceIds, (err, res) =>
          err ? reject(err) : resolve(res!)
        )
      );
    } catch (err) {
      switch ((err as grpc.ServiceError).code) {
        case grpc.status.NOT_FOUND: // Ignore "not found" errors
          pulumi.log.debug(`Service ${id} not found; assuming already deleted`);
          break;
        case grpc.status.UNAUTHENTICATED:
          printDefangHint(olds.fabricDNS);
          throw err;
        default:
          // TODO: what if the fabric was destroyed? Assume the service is gone iff DEFANG_FORCE_UP?
          throw err;
      }
    } finally {
      client.close();
    }
  },
  async diff(
    id: string,
    oldOutputs: DefangServiceOutputs,
    newInputs: DefangServiceInputs
  ): Promise<pulumi.dynamic.DiffResult> {
    assert.equal(id, oldOutputs.service.name);
    const newService = convertServiceInputs(newInputs).toObject();
    if (newInputs.build && oldOutputs.service.build) {
      // Create a (mock) upload URL for the build context, so we can compare it below
      // TODO: this is a bit hacky, because we mimic the behavior of the Fabric controller
      assert(
        newService.build,
        "service.build should've been set in convertServiceInputs"
      );
      newService.build.context = mockCreateUploadURL(
        oldOutputs.service.build.context,
        await getRemoteBuildContextDigest(
          newInputs.build.context,
          newInputs.build.dockerfile
        )
      );
    }
    if (debug) pulumi.log.info(`Old: ${stableStringify(oldOutputs.service)}`);
    if (debug) pulumi.log.info(`New: ${stableStringify(newService)}`);
    return {
      changes:
        newInputs.forceNewDeployment ||
        !isEqual(oldOutputs.service, newService),
      replaces: forceUpdate
        ? [] // prevent calling delete
        : [
            ...optionals(oldOutputs.service.name !== newInputs.name, "name"),
            ...optionals(
              oldOutputs.fabricDNS !== newInputs.fabricDNS,
              "fabricDNS"
            ),
          ],
      // deleteBeforeReplace: false,
      // stables: [],
    };
  },
  async update(
    id: string,
    olds: DefangServiceOutputs,
    news: DefangServiceInputs
  ): Promise<pulumi.dynamic.UpdateResult<DefangServiceOutputs>> {
    assert.equal(id, olds.service.name);
    assert.equal(olds.service.name, news.name);
    assert.equal(olds.fabricDNS, news.fabricDNS);
    const result = await upsert(news, forceUpdate);
    assert.strictEqual(result.getService()?.getName(), id);
    return {
      outs: toOutputs(news.fabricDNS, result, olds),
    };
  },
  async read(
    id: string,
    olds?: DefangServiceOutputs
  ): Promise<pulumi.dynamic.ReadResult<DefangServiceOutputs>> {
    const serviceId = new pb.ServiceID();
    // serviceId.setProject(project);
    serviceId.setName(id);
    assert(olds?.fabricDNS, "fabricDNS is required");
    const client = await connect(olds.fabricDNS);
    try {
      const result = await new Promise<pb.ServiceInfo | undefined>(
        (resolve, reject) =>
          client.get(serviceId, (err, res) =>
            err && err.code !== grpc.status.NOT_FOUND //&& !forceUpdate
              ? reject(err)
              : resolve(res)
          )
      );
      return result
        ? {
            id,
            props: toOutputs(olds.fabricDNS, result),
          }
        : {};
    } catch (err) {
      if ((err as grpc.ServiceError).code === grpc.status.UNAUTHENTICATED) {
        printDefangHint(olds.fabricDNS);
      }
      throw err;
    } finally {
      client.close();
    }
  },
};

export type Platform = "linux/arm64" | "linux/amd64" | "linux";
export type Protocol = "tcp" | "udp" | "http" | "http2" | "grpc";
export type DeviceCapability = "gpu";
export type NetworkName = "private" | "public";
export type Network = { aliases?: string[] } | null; // TODO: pulumi.Input<string>[]

export interface Port {
  target: number;
  protocol?: Protocol;
  mode?: "ingress" | "host";
}

export interface Device {
  capabilities?: DeviceCapability[];
  count?: number;
  // driver?: string; not currently supported
}

export interface Resource {
  /** number of vCPUs to reserve */
  cpus?: number;
  /** number of MiB to reserve */
  memory?: number;
  devices?: Device[];
}

export interface Deploy {
  replicas?: number;
  resources?: {
    reservations?: Resource;
    // limits?: Resource;
  };
}

export interface Secret {
  /** the name of the secret to expose as an environment variable */
  source: pulumi.Input<string>;
  /** optional value of the secret; alternatively, the secret can be set via the Defang CLI */
  value?: pulumi.Input<string>;
}

export interface HealthCheck {
  /** health check test command; could use `curl` or `wget` (for Alpine based images) */
  test: ["CMD", ...string[]] | ["CMD_SHELL", string] | ["NONE"];
  interval?: number;
  timeout?: number;
  retries?: number;
}

export interface Build {
  /** the folder to send to the builder */
  context: string;
  /** the path to the Dockerfile; defaults to Dockerfile */
  dockerfile?: string;
  /** the build args to pass to the builder */
  args?: pulumi.Input<{ [key: string]: pulumi.Input<string> }>;
  /** the shm_size in MiB to pass to the builder */
  shmSize?: pulumi.Input<number>;
  /** target defines the stage to build as defined inside a multi-stage Dockerfile */
  target?: pulumi.Input<string>;
}

export interface DefangServiceArgs {
  /** the DNS name of the Defang Fabric service; defaults to the value of DEFANG_FABRIC or prod, if unset */
  fabricDNS?: pulumi.Input<string>;
  /** the name of the service; defaults to the name of the resource */
  name?: pulumi.Input<string>;
  /** the container image to deploy; required when no build configuration was provided */
  image?: pulumi.Input<string>;
  /** the platform to deploy to; defaults to "linux/amd64" */
  platform?: pulumi.Input<Platform>;
  /** which network the service is in, ie. whether the service requires a public IP or not; defaults to "private" (was: internal=true) */
  networks?: { [key in NetworkName]?: Network };
  /** the optional deployment configuration */
  deploy?: pulumi.Input<Deploy>;
  /** the ports to expose */
  ports?: pulumi.Input<pulumi.Input<Port>[]>;
  /** the environment variables to set; use `null` to mark at sensitive */
  environment?: pulumi.Input<{ [key: string]: pulumi.Input<string> | null }>;
  /** the secrets to expose as environment variables @deprecated use environment key with value `null` */
  secrets?: pulumi.Input<pulumi.Input<Secret>[]>;
  /** force deployment of the service even if nothing has changed */
  forceNewDeployment?: pulumi.Input<boolean>;
  /** the command to run; overrides the container image's CMD */
  command?: pulumi.Input<pulumi.Input<string>[]>;
  /** the optional build configuration; required when no image was provided */
  build?: pulumi.Input<Build>;
  /** the optional health-check test for the service */
  healthcheck?: pulumi.Input<HealthCheck>;
  /** the optional fully qualified domain name for the service; requires CNAME to the publicFqdn */
  domainname?: pulumi.Input<string>;
  /** experimental: mark this service as (managed) Redis */
  x_redis?: pulumi.Input<unknown>;
  /** experimental: mark this service as serving static files */
  x_static_files?: pulumi.Input<StaticFiles>;
  /** if true, this provider will wait for the service to reach a steady state before continuing */
  waitForSteadyState?: pulumi.Input<boolean>;
  /** the project to deploy the service to */
  project?: pulumi.Input<string>;
}

/**
 * A Pulumi custom resource for managing a service on Defang.
 */
export class DefangService extends pulumi.dynamic.Resource {
  /** the name of the service resource */
  public readonly name!: pulumi.Output<string>;
  /** the DNS name of the Defang Fabric service */
  public readonly fabricDNS!: pulumi.Output<string>;
  /** the fully qualified endpoint(s) of the service, one for each port */
  public readonly endpoints!: pulumi.Output<string[]>;
  /** the public NAT IPs of the service; useful for allow-lists */
  public readonly natIPs!: pulumi.Output<string[]>;
  /** the "etag" or deployment ID for the update; useful for tail */
  public readonly etag!: pulumi.Output<string>;
  /** the private load balancer IPs; useful for allow-lists */
  public readonly lbIPs!: pulumi.Output<string[]>;
  /** the private fully qualified domain name of the service; only for services with host ports */
  public readonly privateFqdn!: pulumi.Output<string | undefined>;
  /** the public fully qualified domain name of the service; only for services with ingress ports */
  public readonly publicFqdn!: pulumi.Output<string | undefined>;

  /**
   * Declare a new service on Defang.
   * @constructor
   * @param name the name of the resource
   * @param args the arguments to configure the service
   * @param opts the Pulumi resource options
   */
  constructor(
    name: string,
    args: DefangServiceArgs,
    opts?: pulumi.CustomResourceOptions
  ) {
    if (!args.name) {
      args.name = name;
    }
    if (!args.fabricDNS) {
      args.fabricDNS = defaultFabric;
    }
    if (!args.project) {
      args.project = pulumi.getProject() + "-" + pulumi.getStack();
    }
    super(
      defangServiceProvider,
      name,
      {
        endpoints: undefined,
        etag: undefined,
        lbIPs: undefined,
        natIPs: undefined,
        privateFqdn: undefined,
        publicFqdn: undefined,
        ...args,
      },
      opts
    );

    if (Object.values(args.environment ?? {}).some((v) => v === null)) {
      pulumi.log.info(
        "Use `defang config` to manage sensitive environment variables",
        this
      );
    }
  }
}
