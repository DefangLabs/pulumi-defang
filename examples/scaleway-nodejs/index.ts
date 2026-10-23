import * as pulumi from "@pulumi/pulumi";
import * as defang_scaleway from "@defang-io/pulumi-defang-scaleway";

const scalewayDemo = new defang_scaleway.Project("scaleway-demo", {services: {
    app: {
        image: "nginx",
        ports: [{
            target: 80,
            mode: "ingress",
            appProtocol: "http",
        }],
    },
}});

export const endpoints = scalewayDemo.endpoints;
