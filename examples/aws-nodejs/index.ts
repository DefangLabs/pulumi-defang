import * as pulumi from "@pulumi/pulumi";
import * as defang_aws from "@defang-io/pulumi-defang-aws";

const awsDemo = new defang_aws.Project("aws-demo", {services: {
    app: {
        image: "nginx",
        ports: [{
            target: 80,
            mode: "ingress",
            appProtocol: "http",
        }],
    },
}});
export const endpoints = awsDemo.endpoints;
