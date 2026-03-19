import * as defangaws from "@defang-io/pulumi-defang-aws";

const project = new defangaws.Project("aws-nodejs", {
    services: {
        app: {
            image: "nginx",
            ports: [{
                target: 80,
                mode: "ingress",
                appProtocol: "http",
            }],
        },
    },
});

export const endpoints = project.endpoints;
