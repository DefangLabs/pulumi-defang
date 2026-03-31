import * as pulumi from "@pulumi/pulumi";
import * as defang_azure from "@defang-io/pulumi-defang-azure";

const azureDemo = new defang_azure.Project("azure-demo", {services: {
    app: {
        image: "nginx",
        ports: [{
            target: 80,
            mode: "ingress",
            appProtocol: "http",
        }],
    },
}});
export const endpoints = azureDemo.endpoints;
