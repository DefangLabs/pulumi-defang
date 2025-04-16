import * as pulumi from "@pulumi/pulumi";
import * as defang from "@defang-io/pulumi-defang";

const myProject = new defang.Project("myProject", {configPaths: ["../../compose.yaml.example"]});
export const output = {
    albArn: myProject.albArn,
    etag: myProject.etag,
    services: {
        service1: {
            id: myProject.services.apply(services => services.service1.id),
            task_role: myProject.services.apply(services => services.service1.task_role),
        },
    },
};
