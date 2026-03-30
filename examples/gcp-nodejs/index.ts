import * as pulumi from "@pulumi/pulumi";
import * as defang_gcp from "@defang-io/pulumi-defang-gcp";

const gcpDemo = new defang_gcp.Project("gcp-demo", {services: {
    app: {
        image: "nginx",
        ports: [{
            target: 80,
            mode: "ingress",
            appProtocol: "http",
        }],
    },
}});
export const endpoints = gcpDemo.endpoints;
