import * as grpc from "@grpc/grpc-js";
import { Status } from "@grpc/grpc-js/build/src/constants";
import * as fabric from "./protos/io/defang/v1/fabric_grpc_pb";

function hasPort(url: string): boolean {
  return /:\d+$/.test(url);
}

export function newClient(
  fabricDns: string,
  accessToken: string
): fabric.FabricControllerClient {
  const serviceConfig: grpc.ServiceConfig = {
    loadBalancingConfig: [],
    methodConfig: [
      {
        name: [{ service: "io.defang.v1.FabricController" }],
        retryPolicy: {
          maxAttempts: 5,
          initialBackoff: "1s",
          maxBackoff: "10s",
          backoffMultiplier: 2,
          retryableStatusCodes: [Status.UNAVAILABLE, Status.INTERNAL],
        },
      },
    ],
  };
  const noTenant = fabricDns.replace(/^.*@/, "");
  const withPort = hasPort(noTenant) ? noTenant : `${noTenant}:443`;
  return new fabric.FabricControllerClient(
    withPort,
    grpc.credentials.combineChannelCredentials(
      grpc.credentials.createSsl(),
      grpc.credentials.createFromMetadataGenerator((_, callback) => {
        const metadata = new grpc.Metadata();
        // TODO: automatically generate a new token once it expires
        metadata.set("authorization", "Bearer " + accessToken);
        callback(null, metadata);
      })
    ),
    { "grpc.service_config": JSON.stringify(serviceConfig) }
  );
}
