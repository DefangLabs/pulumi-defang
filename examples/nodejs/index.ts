import * as pulumi from "@pulumi/pulumi";
import * as defang from "@pulumi/defang";

const myProject = new defang.Project("myProject", {
    providerID: "aws",
    name: "my-project",
    configPaths: ["../../compose.yaml.example"],
});
export const output = {
    albArn: myProject.albArn,
    etag: myProject.etag,
};
