import * as pulumi from "@pulumi/pulumi";
import * as defang from "@defang-io/pulumi-defang";

const myProject = new defang.Project("myProject", {
    providerID: "aws",
    configPaths: ["../../compose.yaml.example"],
});
export const output = {
    albArn: myProject.albArn,
    etag: myProject.etag,
};
