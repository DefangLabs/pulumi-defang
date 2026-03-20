import * as pulumi from "@pulumi/pulumi";
import * as defang_azure from "@defang-io/pulumi-defang-azure";

const azureYaml = new defang_azure.defangazure.Project("azure-yaml", {services: {
    app: {
        image: "nginx",
        ports: [{
            target: 80,
            mode: "ingress",
            appProtocol: "http",
        }],
    },
}});
export const endpoints = azureYaml.endpoints;
