import * as pulumi from "@pulumi/pulumi";
import * as defang_gcp from "@defang-io/pulumi-defang-gcp";

const gcpYaml = new defang_gcp.defanggcp.Project("gcp-yaml", {services: {
    app: {
        image: "nginx",
        ports: [{
            target: 80,
            mode: "ingress",
            appProtocol: "http",
        }],
    },
}});
export const endpoints = gcpYaml.endpoints;
